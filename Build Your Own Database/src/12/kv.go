package byodb12

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"os"
	"sort"
	"sync"
	"syscall"
)

type KV struct {
	Path   string
	NoSync bool // for testing
	// internals
	fp   *os.File
	tree struct {
		root uint64
	}
	free FreeListData
	mmap struct {
		file   int      // file size, can be larger than the database size
		total  int      // mmap size, can be larger than the file size
		chunks [][]byte // multiple mmaps, can be non-continuous
	}
	page struct {
		flushed uint64 // database size in number of pages
	}
	// concurrency
	mu      sync.Mutex
	writer  sync.Mutex
	version uint64
	readers ReaderList // heap, for tracking the minimum reader version
}

// implements heap.Interface
type ReaderList []*KVReader

func (h ReaderList) Len() int {
	return len(h)
}
func (h ReaderList) Less(i, j int) bool {
	return h[i].version < h[j].version
}
func (h ReaderList) Swap(i, j int) {
	h[i].index, h[j].index = j, i
	h[i], h[j] = h[j], h[i]
}
func (h *ReaderList) Push(x interface{}) {
	reader := x.(*KVReader)
	reader.index = len(*h)
	*h = append(*h, reader)
}
func (h *ReaderList) Pop() interface{} {
	x := (*h)[len(*h)-1]
	*h = (*h)[:len(*h)-1]
	return x
}

// create the initial mmap that covers the whole file.
func mmapInit(fp *os.File) (int, []byte, error) {
	fi, err := fp.Stat()
	if err != nil {
		return 0, nil, fmt.Errorf("stat: %w", err)
	}

	if fi.Size()%BTREE_PAGE_SIZE != 0 {
		return 0, nil, errors.New("File size is not a multiple of page size.")
	}

	mmapSize := 64 << 20
	assert(mmapSize%BTREE_PAGE_SIZE == 0)
	for mmapSize < int(fi.Size()) {
		mmapSize *= 2
	}
	// mmapSize can be larger than the file

	chunk, err := syscall.Mmap(
		int(fp.Fd()), 0, mmapSize,
		syscall.PROT_READ|syscall.PROT_WRITE, syscall.MAP_SHARED,
	)
	if err != nil {
		return 0, nil, fmt.Errorf("mmap: %w", err)
	}

	return int(fi.Size()), chunk, nil
}

func (db *KV) Open() error {
	// open or create the DB file
	fp, err := os.OpenFile(db.Path, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		return fmt.Errorf("OpenFile: %w", err)
	}
	db.fp = fp

	// create the initial mmap
	sz, chunk, err := mmapInit(db.fp)
	if err != nil {
		goto fail
	}
	db.mmap.file = sz
	db.mmap.total = len(chunk)
	db.mmap.chunks = [][]byte{chunk}

	// read the master page
	err = masterLoad(db)
	if err != nil {
		goto fail
	}

	// done
	return nil

fail:
	db.Close()
	return fmt.Errorf("KV.Open: %w", err)
}

const DB_SIG = "BuildYourOwnDB12"

// the master page format.
// it contains the pointer to the root and other important bits.
// | sig | btree_root | page_used | free_list | version |
// | 16B |     8B     |     8B    |     8B    |    8B   |
func masterLoad(db *KV) error {
	if db.mmap.file == 0 {
		// empty file, the master page will be created on the first write.
		db.page.flushed = 1 // reserved for the master page
		return nil
	}

	data := db.mmap.chunks[0]
	root := binary.LittleEndian.Uint64(data[16:])
	used := binary.LittleEndian.Uint64(data[24:])
	free := binary.LittleEndian.Uint64(data[32:])
	version := binary.LittleEndian.Uint64(data[40:])

	// verify the page
	if !bytes.Equal([]byte(DB_SIG), data[:16]) {
		return errors.New("Bad signature.")
	}
	bad := !(1 <= used && used <= uint64(db.mmap.file/BTREE_PAGE_SIZE))
	bad = bad || !(0 <= root && root < used)
	bad = bad || !(0 <= free && free < used)
	if bad {
		return errors.New("Bad master page.")
	}

	db.tree.root = root
	db.free.head = free
	db.page.flushed = used
	db.version = version
	return nil
}

// update the master page. it must be atomic.
func masterStore(db *KV) error {
	var data [48]byte
	copy(data[:16], []byte(DB_SIG))
	binary.LittleEndian.PutUint64(data[16:], db.tree.root)
	binary.LittleEndian.PutUint64(data[24:], db.page.flushed)
	binary.LittleEndian.PutUint64(data[32:], db.free.head)
	binary.LittleEndian.PutUint64(data[40:], db.version)
	// NOTE: Updating the page via mmap is not atomic.
	//       Use the `pwrite()` syscall instead.
	_, err := db.fp.WriteAt(data[:], 0)
	if err != nil {
		return fmt.Errorf("write master page: %w", err)
	}
	return nil
}

// extend the file to at least `npages`.
func extendFile(db *KV, npages int) error {
	filePages := db.mmap.file / BTREE_PAGE_SIZE
	if filePages >= npages {
		return nil
	}

	for filePages < npages {
		// the file size is increased exponentially,
		// so that we don't have to extend the file for every update.
		inc := filePages / 8
		if inc < 1 {
			inc = 1
		}
		filePages += inc
	}

	fileSize := filePages * BTREE_PAGE_SIZE
	err := syscall.Fallocate(int(db.fp.Fd()), 0, 0, int64(fileSize))
	if err != nil {
		return fmt.Errorf("fallocate: %w", err)
	}

	db.mmap.file = fileSize
	return nil
}

// extend the mmap by adding new mappings.
func extendMmap(db *KV, npages int) error {
	if db.mmap.total >= npages*BTREE_PAGE_SIZE {
		return nil
	}

	// double the address space
	chunk, err := syscall.Mmap(
		int(db.fp.Fd()), int64(db.mmap.total), db.mmap.total,
		syscall.PROT_READ|syscall.PROT_WRITE, syscall.MAP_SHARED,
	)
	if err != nil {
		return fmt.Errorf("mmap: %w", err)
	}

	db.mmap.total += db.mmap.total
	db.mu.Lock()
	db.mmap.chunks = append(db.mmap.chunks, chunk)
	db.mu.Unlock()
	return nil
}

// persist the newly allocated pages after updates
func writePages(tx *KVTX) error {
	// update the free list
	freed := []uint64{}
	for ptr, page := range tx.page.updates {
		if page == nil {
			freed = append(freed, ptr)
		}
	}
	sort.Slice(freed, func(i, j int) bool { return freed[i] < freed[j] })
	tx.free.Add(freed)

	// extend the file & mmap if needed
	db := tx.db
	npages := int(db.page.flushed) + tx.page.nappend
	if err := extendFile(db, npages); err != nil {
		return err
	}
	if err := extendMmap(db, npages); err != nil {
		return err
	}

	// copy pages to the file
	tx.mmap.chunks = db.mmap.chunks
	for ptr, page := range tx.page.updates {
		if page != nil {
			copy(tx.pageGetMapped(ptr).data, page)
		}
	}
	return nil
}

// cleanups
func (db *KV) Close() {
	for _, chunk := range db.mmap.chunks {
		err := syscall.Munmap(chunk)
		assert(err == nil)
	}
	_ = db.fp.Close()
}
