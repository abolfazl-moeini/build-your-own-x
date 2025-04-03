package byodb13

import (
	"container/heap"
	"fmt"
)

// read-only KV transactions
type KVReader struct {
	// the snapshot
	version uint64
	tree    BTree
	mmap    struct {
		chunks [][]byte // copied from struct KV. read-only.
	}
	// for removing from the heap
	index int
	// check misuses
	done bool
}

func (kv *KV) BeginRead(tx *KVReader) {
	kv.mu.Lock()
	tx.mmap.chunks = kv.mmap.chunks
	tx.tree.root = kv.tree.root
	tx.tree.get = tx.pageGetMapped
	tx.version = kv.version
	heap.Push(&kv.readers, tx)
	kv.mu.Unlock()
}

func (kv *KV) EndRead(tx *KVReader) {
	kv.mu.Lock()
	heap.Remove(&kv.readers, tx.index)
	kv.mu.Unlock()
}

// callback for BTree & FreeList, dereference a pointer.
func (tx *KVReader) pageGetMapped(ptr uint64) BNode {
	assert(ptr != 0)
	start := uint64(0)
	for _, chunk := range tx.mmap.chunks {
		end := start + uint64(len(chunk))/BTREE_PAGE_SIZE
		if ptr < end {
			offset := BTREE_PAGE_SIZE * (ptr - start)
			pageEnd := offset + BTREE_PAGE_SIZE
			return BNode{chunk[offset:pageEnd:pageEnd]}
		}
		start = end
	}
	panic("bad ptr")
}

// callback for BTree & FreeList, dereference a pointer.
func (tx *KVTX) pageGet(ptr uint64) BNode {
	assert(ptr != 0)
	if page, ok := tx.page.updates[ptr]; ok {
		assert(page != nil)
		return BNode{page} // for new pages
	}
	return tx.pageGetMapped(ptr) // for written pages
}

// callback for BTree, allocate a new page.
func (tx *KVTX) pageNew(node BNode) uint64 {
	assert(len(node.data) <= BTREE_PAGE_SIZE)
	if ptr := tx.free.Pop(); ptr != 0 {
		// reuse a deallocated page
		tx.page.updates[ptr] = node.data
		return ptr
	}
	return tx.pageAppend(node) // append a new page
}

// callback for BTree, deallocate a page.
func (tx *KVTX) pageDel(ptr uint64) {
	tx.page.updates[ptr] = nil
}

// callback for FreeList, allocate a new page.
func (tx *KVTX) pageAppend(node BNode) uint64 {
	assert(len(node.data) <= BTREE_PAGE_SIZE)
	ptr := tx.db.page.flushed + uint64(tx.page.nappend)
	tx.page.nappend++
	tx.page.updates[ptr] = node.data
	return ptr
}

// callback for FreeList, reuse a page.
func (tx *KVTX) pageUse(ptr uint64, node BNode) {
	tx.page.updates[ptr] = node.data
}

// KV transaction
type KVTX struct {
	KVReader
	db   *KV
	free FreeList
	page struct {
		nappend int // number of pages to be appended
		// newly allocated or deallocated pages keyed by the pointer.
		// nil value denotes a deallocated page.
		updates map[uint64][]byte
	}
}

// begin a transaction
func (kv *KV) Begin(tx *KVTX) {
	tx.db = kv
	tx.page.updates = map[uint64][]byte{}
	tx.mmap.chunks = kv.mmap.chunks
	// XXX: no GC on the tx?
	// runtime.SetFinalizer(tx, func(tx *KVTX) { assert(tx.done) })

	kv.writer.Lock()
	tx.version = kv.version

	// btree
	tx.tree.root = kv.tree.root
	tx.tree.get = tx.pageGet
	tx.tree.new = tx.pageNew
	tx.tree.del = tx.pageDel
	// freelist
	tx.free.FreeListData = kv.free
	tx.free.version = kv.version
	tx.free.get = tx.pageGet
	tx.free.new = tx.pageAppend
	tx.free.use = tx.pageUse

	tx.free.minReader = kv.version
	kv.mu.Lock()
	if len(kv.readers) > 0 {
		tx.free.minReader = kv.readers[0].version
	}
	kv.mu.Unlock()

	assert(tx.page.nappend == 0 && len(tx.page.updates) == 0)
}

// end a transaction: commit updates
func (kv *KV) Commit(tx *KVTX) error {
	assert(!tx.done)
	tx.done = true
	defer kv.writer.Unlock()

	if kv.tree.root == tx.tree.root {
		return nil // no updates?
	}

	// phase 1: persist the page data to disk.
	if err := writePages(tx); err != nil {
		rollbackTX(tx)
		return err
	}

	// the page data must reach disk before the master page.
	// the `fsync` serves as a barrier here.
	if !kv.NoSync {
		if err := kv.fp.Sync(); err != nil {
			rollbackTX(tx)
			return fmt.Errorf("fsync: %w", err)
		}
	}

	// the transaction is visible at this point.
	// save the new version of in-memory data structures.
	kv.page.flushed += uint64(tx.page.nappend)
	kv.free = tx.free.FreeListData
	kv.mu.Lock()
	kv.tree.root = tx.tree.root
	kv.version++
	kv.mu.Unlock()

	// phase 2: update the master page to point to the new tree.
	// NOTE: Cannot rollback the tree to the old version if phase 2 fails.
	//       Because there is no way to know the state of the master page.
	//       Updating from an old root can cause corruption.
	if err := masterStore(kv); err != nil {
		return err
	}
	if !kv.NoSync {
		if err := kv.fp.Sync(); err != nil {
			return fmt.Errorf("fsync: %w", err)
		}
	}
	return nil
}

// rollback the tree and other in-memory data structures.
func rollbackTX(tx *KVTX) {
	// nothing to do
}

// end a transaction: rollback
func (kv *KV) Abort(tx *KVTX) {
	assert(!tx.done)
	tx.done = true
	rollbackTX(tx)
	kv.writer.Unlock()
}

// KV operations
func (tx *KVReader) Get(key []byte) ([]byte, bool) {
	return tx.tree.Get(key)
}
func (tx *KVReader) Seek(key []byte, cmp int) *BIter {
	return tx.tree.Seek(key, cmp)
}
func (tx *KVTX) Update(req *InsertReq) bool {
	tx.tree.InsertEx(req)
	return req.Added
}
func (tx *KVTX) Del(req *DeleteReq) bool {
	return tx.tree.DeleteEx(req)
}
