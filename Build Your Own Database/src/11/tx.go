package byodb11

import (
	"fmt"
	"runtime"
)

// KV transaction
type KVTX struct {
	db *KV
	// for the rollback
	tree struct {
		root uint64
	}
	free struct {
		head uint64
	}
	// check misuses
	done bool
}

func txCheck(tx *KVTX) {
	assert(tx.done)
}

// begin a transaction
func (kv *KV) Begin(tx *KVTX) {
	tx.db = kv
	tx.tree.root = kv.tree.root
	tx.free.head = kv.free.head
	assert(kv.page.nfree == 0)
	assert(kv.page.nappend == 0)
	assert(len(kv.page.updates) == 0)
	runtime.SetFinalizer(tx, txCheck)
}

// end a transaction: commit updates
func (kv *KV) Commit(tx *KVTX) error {
	assert(!tx.done)
	tx.done = true

	if kv.tree.root == tx.tree.root {
		return nil // no updates?
	}

	// phase 1: persist the page data to disk.
	if err := writePages(kv); err != nil {
		rollbackTX(tx)
		return err
	}

	// the page data must reach disk before the master page.
	// the `fsync` serves as a barrier here.
	if err := kv.fp.Sync(); err != nil {
		rollbackTX(tx)
		return fmt.Errorf("fsync: %w", err)
	}

	// the transaction is visible at this point.
	kv.page.flushed += uint64(kv.page.nappend)
	kv.page.nfree = 0
	kv.page.nappend = 0
	kv.page.updates = map[uint64][]byte{}

	// phase 2: update the master page to point to the new tree.
	// NOTE: Cannot rollback the tree to the old version if phase 2 fails.
	//       Because there is no way to know the state of the master page.
	//       Updating from an old root can cause corruption.
	if err := masterStore(kv); err != nil {
		return err
	}
	if err := kv.fp.Sync(); err != nil {
		return fmt.Errorf("fsync: %w", err)
	}
	return nil
}

// rollback the tree and other in-memory data structures.
func rollbackTX(tx *KVTX) {
	kv := tx.db
	kv.tree.root = tx.tree.root
	kv.free.head = tx.free.head
	kv.page.nfree = 0
	kv.page.nappend = 0
	kv.page.updates = map[uint64][]byte{}
}

// end a transaction: rollback
func (kv *KV) Abort(tx *KVTX) {
	assert(!tx.done)
	tx.done = true
	rollbackTX(tx)
}

// KV operations
func (tx *KVTX) Get(key []byte) ([]byte, bool) {
	return tx.db.tree.Get(key)
}
func (tx *KVTX) Seek(key []byte, cmp int) *BIter {
	return tx.db.tree.Seek(key, cmp)
}
func (tx *KVTX) Update(req *InsertReq) bool {
	tx.db.tree.InsertEx(req)
	return req.Added
}
func (tx *KVTX) Del(req *DeleteReq) bool {
	return tx.db.tree.DeleteEx(req)
}
