package byodb12

import (
	"fmt"
	"math/rand"
	"os"
	"reflect"
	"sort"
	"testing"

	is "github.com/stretchr/testify/require"
)

type D struct {
	db  KV
	ref map[string]string
}

func newD() *D {
	os.Remove("test.db")

	d := &D{}
	d.ref = map[string]string{}
	d.db.Path = "test.db"
	d.db.NoSync = true // faster
	err := d.db.Open()
	assert(err == nil)
	return d
}

func (d *D) reopen() {
	d.db.Close()
	d.db = KV{Path: d.db.Path}
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
	tx.Update(&InsertReq{Key: []byte(key), Val: []byte(val)})
	err := d.db.Commit(&tx)
	assert(err == nil)
	d.ref[key] = val
}

func (d *D) del(key string) bool {
	delete(d.ref, key)
	tx := KVTX{}
	d.db.Begin(&tx)
	deleted := tx.Del(&DeleteReq{Key: []byte(key)})
	err := d.db.Commit(&tx)
	assert(err == nil)
	return deleted
}

func (d *D) deref(ptr uint64) BNode {
	tx := KVTX{}
	d.db.Begin(&tx)
	node := tx.pageGet(ptr)
	d.db.Abort(&tx)
	return node
}

func (d *D) dump() ([]string, []string) {
	keys := []string{}
	vals := []string{}

	var nodeDump func(uint64)
	nodeDump = func(ptr uint64) {
		node := d.deref(ptr)
		nkeys := node.nkeys()
		if node.btype() == BNODE_LEAF {
			for i := uint16(0); i < nkeys; i++ {
				keys = append(keys, string(node.getKey(i)))
				vals = append(vals, string(node.getVal(i)))
			}
		} else {
			assert(node.btype() == BNODE_NODE)
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

func flDump(fl *FreeListData, pageget func(uint64) BNode) []uint64 {
	ptrs := []uint64{}
	version := []uint64{}
	nodes := []uint64(nil)

	head := fl.head
	for head != 0 && len(nodes) < len(fl.nodes) {
		nodes = append(nodes, head)
		node := pageget(head)
		assert(node.data[0] == BNODE_FREE_LIST)
		size := flnSize(node)
		start := 0
		if len(nodes) == len(fl.nodes) {
			start = fl.offset
		}
		for i := size - 1; i >= start; i-- {
			ptr, ver := flnItem(node, i)
			ptrs = append(ptrs, ptr)
			version = append(version, ver)
		}
		head = flnNext(node)
	}

	assert(fl.total == len(ptrs))
	for i := 0; i < len(nodes)/2; i++ {
		j := len(nodes) - i - 1
		nodes[i], nodes[j] = nodes[j], nodes[i]
	}
	assert(reflect.DeepEqual(nodes, fl.nodes))

	for i := 0; i < len(version)/2; i++ {
		j := len(version) - i - 1
		version[i], version[j] = version[j], version[i]
		ptrs[i], ptrs[j] = ptrs[j], ptrs[i]
	}

	for i := 1; i < len(version); i++ {
		assert(!versionBefore(version[i], version[i-1]))
	}

	return ptrs
}

func (d *D) verify(t *testing.T) {
	// KVs
	keys, vals := d.dump()
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

	is.Equal(t, rkeys, keys)
	is.Equal(t, rvals, vals)

	// node structures
	pages := make([]uint8, d.db.page.flushed)
	pages[0] = 1
	pages[d.db.tree.root] = 1
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
			pages[ptr] = 1
			key := node.getKey(i)
			kid := d.deref(ptr)
			is.Equal(t, key, kid.getKey(0))
			nodeVerify(kid)
		}
	}

	nodeVerify(d.deref(d.db.tree.root))

	// free list
	idx := len(d.db.free.nodes) - 1
	for head := d.db.free.head; idx >= 0; head = flnNext(d.deref(head)) {
		is.Equal(t, d.db.free.nodes[idx], head)
		is.Zero(t, pages[head])
		pages[head] = 2
		idx--
	}
	freed := flDump(&d.db.free, d.deref)
	for _, ptr := range freed {
		is.Zero(t, pages[ptr])
		pages[ptr] = 3
	}
	for _, flag := range pages {
		is.NotZero(t, flag)
	}
}

func TestKVBasic(t *testing.T) {
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
	t.Log("insertion done")

	// del
	for i := 2000; i < 25000; i++ {
		key := fmt.Sprintf("key%d", fmix32(uint32(i)))
		is.True(t, c.del(key))
	}
	c.verify(t)
	t.Log("deletion done")

	// overwrite
	for i := 0; i < 2000; i++ {
		key := fmt.Sprintf("key%d", fmix32(uint32(i)))
		val := fmt.Sprintf("vvv%d", fmix32(uint32(+i)))
		c.add(key, val)
		c.verify(t)
	}

	is.False(t, c.del("kk"))

	for i := 0; i < 2000; i++ {
		key := fmt.Sprintf("key%d", fmix32(uint32(i)))
		is.True(t, c.del(key))
		c.verify(t)
	}

	c.add("k", "v2")
	c.verify(t)
	c.del("k")
	c.verify(t)
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
