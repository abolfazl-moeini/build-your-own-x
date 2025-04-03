package byodb12

import "encoding/binary"

// node format:
// | next | pointer + version | unused |
// |  8B  |     n*(8B+8B)     |   ...  |
type LNode []byte

const FREE_LIST_HEADER = 8
const FREE_LIST_CAP = (BTREE_PAGE_SIZE - FREE_LIST_HEADER) / 16

// getters & setters
func (node LNode) getNext() uint64 {
	return binary.LittleEndian.Uint64(node[0:8])
}
func (node LNode) setNext(next uint64) {
	binary.LittleEndian.PutUint64(node[0:8], next)
}
func (node LNode) getItem(idx int) (uint64, uint64) {
	offset := FREE_LIST_HEADER + 16*idx
	return binary.LittleEndian.Uint64(node[offset:]),
		binary.LittleEndian.Uint64(node[offset+8:])
}
func (node LNode) setItem(idx int, ptr uint64, version uint64) {
	assert(idx < FREE_LIST_CAP)
	offset := FREE_LIST_HEADER + 16*idx
	binary.LittleEndian.PutUint64(node[offset+0:], ptr)
	binary.LittleEndian.PutUint64(node[offset+8:], version)
}

type FreeList struct {
	// callbacks for managing on-disk pages
	get func(uint64) []byte // read a page
	new func([]byte) uint64 // append a new page
	set func(uint64) []byte // update an existing page
	// persisted data in the meta page
	headPage uint64 // pointer to the list head node
	headSeq  uint64 // monotonic sequence number to index into the list head
	tailPage uint64
	tailSeq  uint64
	// in-memory states
	maxSeq uint64 // saved `tailSeq` to prevent consuming newly added items
	maxVer uint64 // the oldest reader version
	curVer uint64 // version number when committing
}

func seq2idx(seq uint64) int {
	return int(seq % FREE_LIST_CAP)
}

func (fl *FreeList) check() {
	assert(fl.headPage != 0 && fl.tailPage != 0)
	assert(fl.headSeq != fl.tailSeq || fl.headPage == fl.tailPage)
}

// get 1 item from the list head. return 0 on failure.
func (fl *FreeList) PopHead() uint64 {
	ptr, head := flPop(fl)
	if head != 0 { // the empty head node is recycled
		fl.PushTail(head)
	}
	return ptr
}

// remove 1 item from the head node, and remove the head node if empty.
func flPop(fl *FreeList) (ptr uint64, head uint64) {
	fl.check()
	if fl.headSeq == fl.maxSeq {
		return 0, 0 // cannot advance; empty list or the current version
	}
	node := LNode(fl.get(fl.headPage))
	ptr, version := node.getItem(seq2idx(fl.headSeq))
	if versionBefore(fl.maxVer, version) {
		return 0, 0 // cannot advance; still in-use
	}
	fl.headSeq++
	// move to the next one if the head node is empty
	if seq2idx(fl.headSeq) == 0 {
		head, fl.headPage = fl.headPage, node.getNext()
		assert(fl.headPage != 0)
	}
	return
}

// add 1 item to the tail
func (fl *FreeList) PushTail(ptr uint64) {
	fl.check()
	// add it to the tail node
	LNode(fl.set(fl.tailPage)).setItem(seq2idx(fl.tailSeq), ptr, fl.curVer)
	fl.tailSeq++
	// add a new tail node if it's full (the list is never empty)
	if seq2idx(fl.tailSeq) == 0 {
		// try to reuse from the list head
		next, head := flPop(fl) // may remove the head node
		if next == 0 {
			// or allocate a new node by appending
			next = fl.new(make([]byte, BTREE_PAGE_SIZE))
		}
		// link to the new tail node
		LNode(fl.set(fl.tailPage)).setNext(next)
		fl.tailPage = next
		// also add the head node if it's removed
		if head != 0 {
			LNode(fl.set(fl.tailPage)).setItem(0, head, fl.curVer)
			fl.tailSeq++
		}
	}
}

// make the newly added items available for consumption
func (fl *FreeList) SetMaxVer(maxVer uint64) {
	fl.maxSeq = fl.tailSeq
	fl.maxVer = maxVer
}
