package byodb13

import (
	"encoding/binary"
)

// the in-memory data structure that is updated and committed by transactions
type FreeListData struct {
	head uint64
	// cached pointers to list nodes for accessing both ends.
	nodes []uint64 // from the tail to the head
	// cached total number of items; stored in the head node.
	total int
	// cached number of discarded items in the tail node.
	offset int
}

type FreeList struct {
	FreeListData
	// for each transaction
	version   uint64   // current version
	minReader uint64   // minimum reader version
	freed     []uint64 // pages that will be added to the free list
	// callbacks for managing on-disk pages
	get func(uint64) BNode  // dereference a pointer
	new func(BNode) uint64  // append a new page
	use func(uint64, BNode) // reuse a page
}

// node format:
// | type | size | total | next |  pointer-version-pairs |
// |  2B  |  2B  |   8B  |  8B  |       size * 16B       |

const BNODE_FREE_LIST = 3
const FREE_LIST_HEADER = 4 + 8 + 8
const FREE_LIST_CAP = (BTREE_PAGE_SIZE - FREE_LIST_HEADER) / 16

func flTotal(fl *FreeList) int {
	if fl.head == 0 {
		return 0
	}
	node := fl.get(fl.head)
	return int(binary.LittleEndian.Uint64(node.data[4:]))
}
func flnSize(node BNode) int {
	return int(binary.LittleEndian.Uint16(node.data[2:]))
}
func flnNext(node BNode) uint64 {
	return binary.LittleEndian.Uint64(node.data[12:])
}
func flnItem(node BNode, idx int) (uint64, uint64) {
	offset := FREE_LIST_HEADER + 16*idx
	ptr := binary.LittleEndian.Uint64(node.data[offset+0:])
	ver := binary.LittleEndian.Uint64(node.data[offset+8:])
	return ptr, ver
}
func flnSetItem(node BNode, idx int, ptr uint64, ver uint64) {
	assert(idx < flnSize(node))
	offset := FREE_LIST_HEADER + 16*idx
	binary.LittleEndian.PutUint64(node.data[offset+0:], ptr)
	binary.LittleEndian.PutUint64(node.data[offset+8:], ver)
}
func flnSetHeader(node BNode, size uint16, next uint64) {
	binary.LittleEndian.PutUint16(node.data[0:], BNODE_FREE_LIST)
	binary.LittleEndian.PutUint16(node.data[2:], size)
	binary.LittleEndian.PutUint64(node.data[4:], 0)
	binary.LittleEndian.PutUint64(node.data[12:], next)
}
func flnSetTotal(node BNode, total uint64) {
	binary.LittleEndian.PutUint64(node.data[4:], total)
}

// load cached fields on the first use
func (fl *FreeList) loadCache() {
	if fl.head == 0 || len(fl.nodes) > 0 {
		return
	}

	fl.total = flTotal(fl)

	// XXX: this is O(n)
	head := fl.head
	remain := fl.total
	for remain > 0 {
		node := fl.get(head)
		fl.nodes = append(fl.nodes, head)
		remain -= flnSize(node)
		head = flnNext(node)
	}

	for i := 0; i < len(fl.nodes)/2; i++ {
		j := len(fl.nodes) - i - 1
		fl.nodes[i], fl.nodes[j] = fl.nodes[j], fl.nodes[i]
	}

	fl.offset = -remain
}

// try to remove an item from the tail. returns 0 on failure.
// the removed pointer must not be reachable by the minimum version reader.
func (fl *FreeList) Pop() uint64 {
	fl.loadCache()
	return flPop1(fl)
}

// a < b
func versionBefore(a, b uint64) bool {
	return int64(a-b) < 0
}

func flPop1(fl *FreeList) uint64 {
	if fl.total == 0 {
		return 0
	}

	// remove one item from the tail
	assert(fl.offset < flnSize(fl.get(fl.nodes[0])))
	ptr, ver := flnItem(fl.get(fl.nodes[0]), fl.offset)
	if versionBefore(fl.minReader, ver) {
		// cannot use; possibly reachable by the minimum version reader.
		return 0
	}
	fl.offset++
	fl.total--

	for len(fl.nodes) > 0 && fl.offset == flnSize(fl.get(fl.nodes[0])) {
		// discard the empty node and move to the next node
		fl.offset = 0
		fl.freed = append(fl.freed, fl.nodes[0]) // recyle the node itself
		fl.nodes = fl.nodes[1:]
		if len(fl.nodes) == 0 {
			fl.head = 0
		}
	}
	return ptr
}

// remove the list head (before replacing it)
func flRemoveHead(fl *FreeList) ([]uint64, []uint64) {
	node := fl.get(fl.head)
	sz := flnSize(node)

	start := 0
	if len(fl.nodes) == 1 {
		start = fl.offset
	}
	fl.total -= sz - start

	prepend := []uint64{}
	version := []uint64{}
	for ; start < sz; start++ {
		ptr, ver := flnItem(node, start)
		prepend = append(prepend, ptr)
		version = append(version, ver)
	}

	fl.nodes = fl.nodes[:len(fl.nodes)-1]
	if len(fl.nodes) > 0 {
		fl.head = fl.nodes[len(fl.nodes)-1]
	} else {
		fl.head = 0
		fl.offset = 0
	}
	return prepend, version
}

// add some new pointers to the head and finalize the update
func (fl *FreeList) Add(freed []uint64) {
	assert(fl.head == 0 || len(fl.nodes) > 0)
	fl.freed = append(freed, fl.freed...)

	if len(fl.freed) > 0 {
		// reuse pointers from the free list itself
		reuse := []uint64{}
		for fl.total > 0 {
			remain := flnSize(fl.get(fl.head))
			if len(fl.nodes) == 1 {
				remain -= fl.offset
			}
			if len(reuse)*FREE_LIST_CAP >= remain+len(fl.freed)+1 {
				break
			}

			ptr := flPop1(fl)
			if ptr == 0 {
				break
			}
			reuse = append(reuse, ptr)
		}

		// remove the head node
		var version []uint64
		if fl.total > 0 {
			fl.freed = append(fl.freed, fl.head) // recyle the node itself
			var prepend []uint64
			prepend, version = flRemoveHead(fl)
			fl.freed = append(prepend, fl.freed...)
		}

		// add new head nodes
		flPush(fl, fl.freed, version, reuse)
	}

	// done
	if fl.head != 0 {
		flnSetTotal(fl.get(fl.head), uint64(fl.total))
	}
}

func flPush(fl *FreeList, freed []uint64, version []uint64, reuse []uint64) {
	fl.total += len(freed)
	for len(freed) > 0 {
		new := BNode{make([]byte, BTREE_PAGE_SIZE)}

		// construct a new node
		size := len(freed)
		if size > FREE_LIST_CAP {
			size = FREE_LIST_CAP
		}
		flnSetHeader(new, uint16(size), fl.head)
		for i, ptr := range freed[:size] {
			ver := fl.version + 1 // the updated version
			if len(version) > 0 {
				ver, version = version[0], version[1:]
			}
			flnSetItem(new, i, ptr, ver)
		}
		freed = freed[size:]

		if len(reuse) > 0 {
			// reuse a pointer from the list
			fl.head, reuse = reuse[0], reuse[1:]
			fl.use(fl.head, new)
		} else {
			// or append a page to house the new node
			fl.head = fl.new(new)
		}
		fl.nodes = append(fl.nodes, fl.head)
	}

	if len(reuse) > 0 {
		// edge case: an empty node is left
		assert(len(reuse) == 1)
		new := BNode{make([]byte, BTREE_PAGE_SIZE)}
		flnSetHeader(new, 0, fl.head)
		fl.head = reuse[0]
		fl.use(fl.head, new)
		fl.nodes = append(fl.nodes, fl.head)
	}
}
