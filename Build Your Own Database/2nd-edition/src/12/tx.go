package byodb12

import (
	"bytes"
	"errors"
	"runtime"
	"slices"
)

// start <= key <= stop
type KeyRange struct {
	start []byte
	stop  []byte
}

// KV transaction
type KVTX struct {
	// a read-only snapshot
	snapshot BTree
	version  uint64
	// captured KV updates:
	// values are prefixed by a 1-byte flag to indicate deleted keys.
	pending BTree
	// a list of involved intervals of keys for detecting conflicts
	reads []KeyRange
	// should check for conflicts even if an update changes nothing
	updateAttempted bool
	// check misuses
	done bool
}

// a < b
func versionBefore(a, b uint64) bool {
	return a-b > 1<<63 // this works even after wraparounds
}

// a prefix for values in KVTX.pending
const (
	FLAG_DELETED = byte(1)
	FLAG_UPDATED = byte(2)
)

// begin a transaction
func (kv *KV) Begin(tx *KVTX) {
	kv.mutex.Lock()
	defer kv.mutex.Unlock()
	// read-only snapshot, just the tree root and the page read callback
	tx.snapshot.root = kv.tree.root
	chunks := kv.mmap.chunks // copied to avoid updates from writers
	tx.snapshot.get = func(ptr uint64) []byte { return mmapRead(ptr, chunks) }
	tx.version = kv.version
	// in-memory tree to capture updates
	pages := [][]byte(nil)
	tx.pending.get = func(ptr uint64) []byte { return pages[ptr-1] }
	tx.pending.new = func(node []byte) uint64 {
		pages = append(pages, node)
		return uint64(len(pages))
	}
	tx.pending.del = func(uint64) {}
	// keep track of concurrent TXs
	kv.ongoing = append(kv.ongoing, tx.version)
	// XXX: sanity check; unreliable
	runtime.SetFinalizer(tx, func(tx *KVTX) { assert(tx.done) })
}

func sortedRangesOverlap(s1, s2 []KeyRange) bool {
	for len(s1) > 0 && len(s2) > 0 {
		if bytes.Compare(s1[0].stop, s2[0].start) < 0 {
			s1 = s1[1:]
		} else if bytes.Compare(s2[0].stop, s1[0].start) < 0 {
			s2 = s2[1:]
		} else {
			return true
		}
	}
	return false
}

func detectConflicts(kv *KV, tx *KVTX) bool {
	// sort the dependency ranges for easier overlap detection
	slices.SortFunc(tx.reads, func(r1, r2 KeyRange) int {
		return bytes.Compare(r1.start, r2.start)
	})
	// do they overlap with newer versions?
	for i := len(kv.history) - 1; i >= 0; i-- {
		if !versionBefore(tx.version, kv.history[i].version) {
			break // sorted
		}
		if sortedRangesOverlap(tx.reads, kv.history[i].writes) {
			return true
		}
	}
	return false
}

var ErrorConflict = errors.New("cannot commit due to conflict")

// end a transaction: commit updates
func (kv *KV) Commit(tx *KVTX) error {
	assert(!tx.done)
	tx.done = true
	kv.mutex.Lock()
	defer kv.mutex.Unlock()
	defer txFinalize(kv, tx)

	// check conflicts
	if tx.updateAttempted && detectConflicts(kv, tx) {
		return ErrorConflict
	}

	// save the meta page
	meta, root := saveMeta(kv), kv.tree.root
	// transfer updates to the current tree
	kv.free.curVer = kv.version + 1 // version in the free list
	writes := []KeyRange(nil)       // collect updated ranges in this TX
	for iter := tx.pending.Seek(nil, CMP_GT); iter.Valid(); iter.Next() {
		modified := false
		key, val := iter.Deref()
		oldVal, isOld := tx.snapshot.Get(key)
		switch val[0] {
		case FLAG_DELETED:
			modified = isOld
			deleted, err := kv.tree.Delete(&DeleteReq{Key: key})
			assert(err == nil)          // can only fail by length limit
			assert(deleted == modified) // assured by conflict detection
		case FLAG_UPDATED:
			modified = (!isOld || !bytes.Equal(oldVal, val[1:]))
			updated, err := kv.tree.Update(&UpdateReq{Key: key, Val: val[1:]})
			assert(err == nil)          // can only fail by length limit
			assert(updated == modified) // assured by conflict detection
		default:
			panic("unreachable")
		}
		if modified && len(kv.ongoing) > 1 {
			writes = append(writes, KeyRange{key, key})
		}
	}

	// commit the update
	if root != kv.tree.root {
		kv.version++
		if err := updateOrRevert(kv, meta); err != nil {
			return err
		}
	}

	// keep a history of updated ranges grouped by each TX
	if len(writes) > 0 {
		// sort the ranges for faster overlap detection
		slices.SortFunc(writes, func(r1, r2 KeyRange) int {
			return bytes.Compare(r1.start, r2.start)
		})
		kv.history = append(kv.history, CommittedTX{kv.version, writes})
	}
	return nil
}

// common routines when exiting a transaction
func txFinalize(kv *KV, tx *KVTX) {
	// remove myself from `kv.ongoing`
	idx := slices.Index(kv.ongoing, tx.version)
	last := len(kv.ongoing) - 1
	kv.ongoing[idx], kv.ongoing = kv.ongoing[last], kv.ongoing[:last]
	// find the oldest in-use version
	minVersion := kv.version
	for _, other := range kv.ongoing {
		if versionBefore(other, minVersion) {
			minVersion = other
		}
	}
	// release the free list
	kv.free.SetMaxVer(minVersion)
	// trim `kv.history` if `minVersion` has been increased
	for idx = 0; idx < len(kv.history); idx++ {
		if versionBefore(minVersion, kv.history[idx].version) {
			break // sorted
		}
	}
	kv.history = kv.history[idx:] // sorted
}

// end a transaction: rollback
func (kv *KV) Abort(tx *KVTX) {
	assert(!tx.done)
	tx.done = true
	// maintain `kv.ongoing` and `kv.history`
	kv.mutex.Lock()
	txFinalize(kv, tx)
	kv.mutex.Unlock()
}

type KVIter interface {
	Deref() (key []byte, val []byte)
	Valid() bool
	Next()
	// TODO: Prev()
}

// an iterator that combines pending updates and the snapshot
type CombinedIter struct {
	top *BIter // KVTX.pending
	bot *BIter // KVTX.snapshot
	dir int    // +1 for greater or greater-than, -1 for less or less-than
	// the end of the range
	cmp int
	end []byte
}

func (iter *CombinedIter) Deref() ([]byte, []byte) {
	var k1, v1, k2, v2 []byte
	top, bot := iter.top.Valid(), iter.bot.Valid()
	assert(top || bot)
	if top {
		k1, v1 = iter.top.Deref()
	}
	if bot {
		k2, v2 = iter.bot.Deref()
	}
	// use the min/max key of the two
	if top && bot && bytes.Compare(k1, k2) == +iter.dir {
		return k2, v2
	}
	if top {
		return k1, v1[1:]
	} else {
		return k2, v2
	}
}

func (iter *CombinedIter) Valid() bool {
	if iter.top.Valid() || iter.bot.Valid() {
		key, _ := iter.Deref()
		return cmpOK(key, iter.cmp, iter.end)
	}
	return false
}

func (iter *CombinedIter) Next() {
	// which B+tree iterator to move?
	top, bot := iter.top.Valid(), iter.bot.Valid()
	if top && bot {
		k1, _ := iter.top.Deref()
		k2, _ := iter.bot.Deref()
		switch bytes.Compare(k1, k2) {
		case -iter.dir:
			top, bot = true, false
		case +iter.dir:
			top, bot = false, true
		case 0: // equal; move both
		}
	}
	assert(top || bot)
	// move B+tree iterators wrt the direction
	if top {
		if iter.dir > 0 {
			iter.top.Next()
		} else {
			iter.top.Prev()
		}
	}
	if bot {
		if iter.dir > 0 {
			iter.bot.Next()
		} else {
			iter.bot.Prev()
		}
	}
}

// point query. combines captured updates with the snapshot
func (tx *KVTX) Get(key []byte) ([]byte, bool) {
	tx.reads = append(tx.reads, KeyRange{key, key}) // dependency
	val, ok := tx.pending.Get(key)
	switch {
	case ok && val[0] == FLAG_UPDATED: // updated in this TX
		return val[1:], true
	case ok && val[0] == FLAG_DELETED: // deleted in this TX
		return nil, false
	case !ok: // read from the snapshot
		return tx.snapshot.Get(key)
	default:
		panic("unreachable")
	}
}

func cmp2dir(cmp int) int {
	if cmp > 0 {
		return +1
	} else {
		return -1
	}
}

// range query. combines captured updates with the snapshot
func (tx *KVTX) Seek(key1 []byte, cmp1 int, key2 []byte, cmp2 int) KVIter {
	assert(cmp2dir(cmp1) != cmp2dir(cmp2))
	lo, hi := key1, key2
	if cmp2dir(cmp1) < 0 {
		lo, hi = hi, lo
	}
	tx.reads = append(tx.reads, KeyRange{lo, hi}) // FIXME: slightly larger
	return &CombinedIter{
		top: tx.pending.Seek(key1, cmp1),
		bot: tx.snapshot.Seek(key1, cmp1),
		dir: cmp2dir(cmp1),
		cmp: cmp2,
		end: key2,
	}
}

// capture updates
func (tx *KVTX) Update(req *UpdateReq) (bool, error) {
	tx.updateAttempted = true
	// check the existing key against the update mode
	old, exists := tx.Get(req.Key) // also add a dependency
	if req.Mode == MODE_UPDATE_ONLY && !exists {
		return false, nil
	}
	if req.Mode == MODE_INSERT_ONLY && exists {
		return false, nil
	}
	if exists && bytes.Equal(old, req.Val) {
		return false, nil
	}
	// insert the flagged KV (reduces the size limit by 1 byte)
	flaggedVal := append([]byte{FLAG_UPDATED}, req.Val...)
	_, err := tx.pending.Update(&UpdateReq{Key: req.Key, Val: flaggedVal})
	if err != nil {
		return false, err // length limit
	}
	req.Added = !exists
	req.Updated = true
	req.Old = old
	return true, nil
}

func (tx *KVTX) Set(key []byte, val []byte) (bool, error) {
	return tx.Update(&UpdateReq{Key: key, Val: val})
}

func (tx *KVTX) Del(req *DeleteReq) (bool, error) {
	tx.updateAttempted = true
	exists := false
	if req.Old, exists = tx.Get(req.Key); !exists {
		return false, nil
	}
	return tx.pending.Update(&UpdateReq{Key: req.Key, Val: []byte{FLAG_DELETED}})
}
