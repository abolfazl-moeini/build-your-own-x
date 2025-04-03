package byodb13

import (
	"fmt"
	"math/rand"
	"os"
	"sort"
	"testing"

	is "github.com/stretchr/testify/require"
)

type D struct {
	db  KV
	ref map[string]string
}

func nofsync(int) error {
	return nil
}

func newD() *D {
	os.Remove("test.db")

	d := &D{}
	d.ref = map[string]string{}
	d.db.Path = "test.db"
	d.db.Fsync = nofsync // faster
	err := d.db.Open()
	assert(err == nil)
	return d
}

func (d *D) reopen() {
	d.db.Close()
	d.db = KV{Path: d.db.Path, Fsync: d.db.Fsync}
	err := d.db.Open()
	assert(err == nil)
}

func (d *D) dispose() {
	d.db.Close()
	os.Remove("test.db")
}

func (d *D) add(key string, val string) {
	tx := KVTX{}
	d.db.Begin(&tx)
	_, err := tx.Set([]byte(key), []byte(val))
	assert(err == nil)
	err = d.db.Commit(&tx)
	assert(err == nil)
	d.ref[key] = val
}

func (d *D) del(key string) bool {
	delete(d.ref, key)
	tx := KVTX{}
	d.db.Begin(&tx)
	deleted, err := tx.Del(&DeleteReq{Key: []byte(key)})
	assert(err == nil)
	err = d.db.Commit(&tx)
	assert(err == nil)
	return deleted
}

func (d *D) dump() ([]string, []string) {
	keys := []string{}
	vals := []string{}

	var nodeDump func(uint64)
	nodeDump = func(ptr uint64) {
		node := BNode(d.db.tree.get(ptr))
		nkeys := node.nkeys()
		if node.btype() == BNODE_LEAF {
			for i := uint16(0); i < nkeys; i++ {
				keys = append(keys, string(node.getKey(i)))
				vals = append(vals, string(node.getVal(i)))
			}
		} else {
			for i := uint16(0); i < nkeys; i++ {
				ptr := node.getPtr(i)
				nodeDump(ptr)
			}
		}
	}

	nodeDump(d.db.tree.root)
	assert(keys[0] == "")
	assert(vals[0] == "")
	return keys[1:], vals[1:]
}

func (d *D) verify(t *testing.T) {
	// KV data
	keys, vals := d.dump()
	// reference data
	rkeys, rvals := []string{}, []string{}
	for k, v := range d.ref {
		rkeys = append(rkeys, k)
		rvals = append(rvals, v)
	}
	is.Equal(t, len(rkeys), len(keys))
	sort.Stable(sortIF{
		len:  len(rkeys),
		less: func(i, j int) bool { return rkeys[i] < rkeys[j] },
		swap: func(i, j int) {
			k, v := rkeys[i], rvals[i]
			rkeys[i], rvals[i] = rkeys[j], rvals[j]
			rkeys[j], rvals[j] = k, v
		},
	})
	// compare with the reference
	is.Equal(t, rkeys, keys)
	is.Equal(t, rvals, vals)

	// track visited pages
	pages := make([]uint8, d.db.page.flushed)
	pages[0] = 1
	pages[d.db.tree.root] = 1
	// verify node structures
	var nodeVerify func(BNode)
	nodeVerify = func(node BNode) {
		nkeys := node.nkeys()
		assert(nkeys >= 1)
		if node.btype() == BNODE_LEAF {
			return
		}
		for i := uint16(0); i < nkeys; i++ {
			ptr := node.getPtr(i)
			is.Zero(t, pages[ptr])
			pages[ptr] = 1 // tree node
			key := node.getKey(i)
			kid := BNode(d.db.tree.get(node.getPtr(i)))
			is.Equal(t, key, kid.getKey(0))
			nodeVerify(kid)
		}
	}

	nodeVerify(d.db.tree.get(d.db.tree.root))

	// free list
	list, nodes := flDump(&d.db.free)
	for _, ptr := range nodes {
		is.Zero(t, pages[ptr])
		pages[ptr] = 2 // free list node
	}
	for _, ptr := range list {
		is.Zero(t, pages[ptr])
		pages[ptr] = 3 // free list content
	}
	for _, flag := range pages {
		is.NotZero(t, flag) // every page is accounted for
	}
}

func funcTestKVBasic(t *testing.T, reopen bool) {
	c := newD()
	defer c.dispose()

	c.add("k", "v")
	c.verify(t)

	// insert
	for i := 0; i < 25000; i++ {
		key := fmt.Sprintf("key%d", fmix32(uint32(i)))
		val := fmt.Sprintf("vvv%d", fmix32(uint32(-i)))
		c.add(key, val)
		if i < 2000 {
			c.verify(t)
		}
	}
	c.verify(t)
	if reopen {
		c.reopen()
		c.verify(t)
	}
	t.Log("insertion done")

	// del
	for i := 2000; i < 25000; i++ {
		key := fmt.Sprintf("key%d", fmix32(uint32(i)))
		is.True(t, c.del(key))
	}
	c.verify(t)
	if reopen {
		c.reopen()
		c.verify(t)
	}
	t.Log("deletion done")

	// overwrite
	for i := 0; i < 2000; i++ {
		key := fmt.Sprintf("key%d", fmix32(uint32(i)))
		val := fmt.Sprintf("vvv%d", fmix32(uint32(+i)))
		c.add(key, val)
		c.verify(t)
	}

	is.False(t, c.del("kk"))

	// remove all
	for i := 0; i < 2000; i++ {
		key := fmt.Sprintf("key%d", fmix32(uint32(i)))
		is.True(t, c.del(key))
		c.verify(t)
	}
	if reopen {
		c.reopen()
		c.verify(t)
	}

	c.add("k", "v2")
	c.verify(t)
	c.del("k")
	c.verify(t)
}

func TestKVBasic(t *testing.T) {
	funcTestKVBasic(t, false)
	funcTestKVBasic(t, true)
}

func fsyncErr(errlist ...int) func(int) error {
	return func(int) error {
		fail := errlist[0]
		errlist = errlist[1:]
		if fail != 0 {
			return fmt.Errorf("fsync error!")
		} else {
			return nil
		}
	}
}

func TestKVFsyncErr(t *testing.T) {
	c := newD()
	defer c.dispose()

	set := func(key []byte, val []byte) error {
		tx := KVTX{}
		c.db.Begin(&tx)
		tx.Set(key, val)
		return c.db.Commit(&tx)
	}
	get := func(key []byte) ([]byte, bool) {
		tx := KVTX{}
		c.db.Begin(&tx)
		val, ok := tx.Get(key)
		c.db.Abort(&tx)
		return val, ok
	}

	err := set([]byte("k"), []byte("1"))
	assert(err == nil)
	val, ok := get([]byte("k"))
	assert(ok && string(val) == "1")

	c.db.Fsync = fsyncErr(1)
	err = set([]byte("k"), []byte("2"))
	assert(err != nil)
	val, ok = get([]byte("k"))
	assert(ok && string(val) == "1")

	c.db.Fsync = nofsync
	err = set([]byte("k"), []byte("3"))
	assert(err == nil)
	val, ok = get([]byte("k"))
	assert(ok && string(val) == "3")

	c.db.Fsync = fsyncErr(0, 1)
	err = set([]byte("k"), []byte("4"))
	assert(err != nil)
	val, ok = get([]byte("k"))
	assert(ok && string(val) == "3")

	c.db.Fsync = nofsync
	err = set([]byte("k"), []byte("5"))
	assert(err == nil)
	val, ok = get([]byte("k"))
	assert(ok && string(val) == "5")

	c.db.Fsync = fsyncErr(0, 1)
	err = set([]byte("k"), []byte("6"))
	assert(err != nil)
	val, ok = get([]byte("k"))
	assert(ok && string(val) == "5")
}

func TestKVRandLength(t *testing.T) {
	c := newD()
	defer c.dispose()

	for i := 0; i < 2000; i++ {
		klen := fmix32(uint32(2*i+0)) % BTREE_MAX_KEY_SIZE
		vlen := fmix32(uint32(2*i+1)) % BTREE_MAX_VAL_SIZE
		if klen == 0 {
			continue
		}

		key := make([]byte, klen)
		rand.Read(key)
		val := make([]byte, vlen)
		// rand.Read(val)
		c.add(string(key), string(val))
		c.verify(t)
	}
}

func TestKVIncLength(t *testing.T) {
	for l := 1; l < BTREE_MAX_KEY_SIZE+BTREE_MAX_VAL_SIZE; l++ {
		c := newD()

		klen := l
		if klen > BTREE_MAX_KEY_SIZE {
			klen = BTREE_MAX_KEY_SIZE
		}
		vlen := l - klen
		key := make([]byte, klen)
		val := make([]byte, vlen)

		factor := BTREE_PAGE_SIZE / l
		size := factor * factor * 2
		if size > 4000 {
			size = 4000
		}
		if size < 10 {
			size = 10
		}
		for i := 0; i < size; i++ {
			rand.Read(key)
			c.add(string(key), string(val))
		}
		c.verify(t)

		c.dispose()
	}
}

func fileSize(path string) int64 {
	finfo, err := os.Stat(path)
	assert(err == nil)
	return finfo.Size()
}

// test the free list: file size do not increase under various operations
func TestKVFileSize(t *testing.T) {
	c := newD()
	fill := func(seed int) {
		for i := 0; i < 2000; i++ {
			key := fmt.Sprintf("key%d", fmix32(uint32(i)))
			val := fmt.Sprintf("vvv%010d", fmix32(uint32(seed*2000+i)))
			c.add(key, val)
		}
	}
	fill(0)
	fill(1)
	size := fileSize(c.db.Path)

	// update the same key
	fill(2)
	assert(size == fileSize(c.db.Path))

	// remove everything
	for i := 0; i < 2000; i++ {
		key := fmt.Sprintf("key%d", fmix32(uint32(i)))
		c.del(key)
	}
	assert(size == fileSize(c.db.Path))

	// add them back
	fill(3)
	assert(size == fileSize(c.db.Path))
}
