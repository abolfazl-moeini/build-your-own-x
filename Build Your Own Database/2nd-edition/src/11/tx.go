package byodb11

import (
	"runtime"
)

// KV transaction
type KVTX struct {
	db   *KV
	meta []byte // for the rollback
	root uint64 // the saved root pointer for skipping empty transactions
	done bool   // check misuses
}

// begin a transaction
func (kv *KV) Begin(tx *KVTX) {
	tx.db = kv
	tx.meta = saveMeta(tx.db)
	tx.root = tx.db.tree.root
	assert(kv.page.nappend == 0 && len(kv.page.updates) == 0)
	// XXX: sanity check; unreliable
	runtime.SetFinalizer(tx, func(tx *KVTX) { assert(tx.done) })
}

// end a transaction: commit updates
func (kv *KV) Commit(tx *KVTX) error {
	assert(!tx.done)
	tx.done = true
	if kv.tree.root == tx.root {
		return nil // no updates?
	}
	return updateOrRevert(tx.db, tx.meta)
}

// end a transaction: rollback
func (kv *KV) Abort(tx *KVTX) {
	assert(!tx.done)
	tx.done = true
	// nothing has written, just revert the in-memory states
	loadMeta(tx.db, tx.meta)
	// discard temporaries
	tx.db.page.nappend = 0
	tx.db.page.updates = map[uint64][]byte{}
}

// KV interfaces
func (tx *KVTX) Get(key []byte) ([]byte, bool) {
	return tx.db.tree.Get(key)
}
func (tx *KVTX) Seek(key []byte, cmp int) *BIter {
	return tx.db.tree.Seek(key, cmp)
}
func (tx *KVTX) Update(req *UpdateReq) (bool, error) {
	return tx.db.tree.Update(req)
}
func (tx *KVTX) Set(key []byte, val []byte) (bool, error) {
	return tx.Update(&UpdateReq{Key: key, Val: val})
}
func (tx *KVTX) Del(req *DeleteReq) (bool, error) {
	return tx.db.tree.Delete(req)
}
