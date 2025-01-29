package byodb11

import "bytes"

// B-tree iterator
type BIter struct {
	tree *BTree
	path []BNode  // from root to leaf
	pos  []uint16 // indexes into nodes
}

func (iter *BIter) Clone() *BIter {
	return &BIter{
		tree: iter.tree,
		path: append([]BNode(nil), iter.path...),
		pos:  append([]uint16(nil), iter.pos...),
	}
}

// current KV pair
func (iter *BIter) Deref() ([]byte, []byte) {
	assert(iter.Valid())
	last := len(iter.path) - 1
	node := iter.path[last]
	pos := iter.pos[last]
	return node.getKey(pos), node.getVal(pos)
}

func (iter *BIter) Valid() bool {
	// the first key in the tree is not real (dummy)
	dummy := true
	for _, pos := range iter.pos {
		if pos != 0 {
			dummy = false
		}
	}
	if dummy {
		return false
	}

	last := len(iter.path) - 1
	node := iter.path[last]
	return iter.pos[last] < node.nkeys()
}

func iterPrev(iter *BIter, level int) {
	if iter.pos[level] > 0 {
		iter.pos[level]-- // move within this node
	} else if level > 0 {
		iterPrev(iter, level-1) // move to a slibing node
	} else {
		return // dummy key
	}

	if level+1 < len(iter.pos) {
		// update the kid node
		node := iter.path[level]
		kid := iter.tree.get(node.getPtr(iter.pos[level]))
		iter.path[level+1] = kid
		iter.pos[level+1] = kid.nkeys() - 1
	}
}

func iterNext(iter *BIter, level int) {
	if iter.pos[level]+1 < iter.path[level].nkeys() {
		iter.pos[level]++
	} else if level > 0 {
		iterNext(iter, level-1)
	} else {
		iter.pos[len(iter.pos)-1]++ // past the last key
		return
	}

	if level+1 < len(iter.pos) {
		node := iter.path[level]
		kid := iter.tree.get(node.getPtr(iter.pos[level]))
		iter.path[level+1] = kid
		iter.pos[level+1] = 0
	}
}

func (iter *BIter) Prev() {
	iterPrev(iter, len(iter.path)-1)
}

func (iter *BIter) Next() {
	iterNext(iter, len(iter.path)-1)
}

// find the closest position that is less or equal to the input key
func (tree *BTree) SeekLE(key []byte) *BIter {
	iter := &BIter{tree: tree}
	for ptr := tree.root; ptr != 0; {
		node := tree.get(ptr)
		idx := nodeLookupLE(node, key)
		iter.path = append(iter.path, node)
		iter.pos = append(iter.pos, idx)
		if node.btype() == BNODE_NODE {
			ptr = node.getPtr(idx)
		} else {
			ptr = 0
		}
	}
	return iter
}

const (
	CMP_GE = +3 // >=
	CMP_GT = +2 // >
	CMP_LT = -2 // <
	CMP_LE = -3 // <=
)

// key cmp ref
func cmpOK(key []byte, cmp int, ref []byte) bool {
	r := bytes.Compare(key, ref)
	switch cmp {
	case CMP_GE:
		return r >= 0
	case CMP_GT:
		return r > 0
	case CMP_LT:
		return r < 0
	case CMP_LE:
		return r <= 0
	default:
		panic("what?")
	}
}

// find the closest position to a key with respect to the `cmp` relation
func (tree *BTree) Seek(key []byte, cmp int) *BIter {
	iter := tree.SeekLE(key)
	if cmp != CMP_LE && iter.Valid() {
		cur, _ := iter.Deref()
		if !cmpOK(cur, cmp, key) {
			// off by one
			if cmp > 0 {
				iter.Next()
			} else {
				iter.Prev()
			}
		}
	}
	if iter.Valid() {
		cur, _ := iter.Deref()
		assert(cmpOK(cur, cmp, key))
	}
	return iter
}
