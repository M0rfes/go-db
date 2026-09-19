package node

import (
	"bytes"
	"encoding/binary"
	"log"
)

const (
	HEADER_BYTES  = 4
	POINTER_BYTES = 8
	OFFSET_BYTES  = 2
	K_SIZE        = 2
	V_SIZE        = 2
)

/*
| type | nkeys | pointers | offsets | key-values   |
|  1   |   2   | 101      |  6      | 2 0 "k1"     |

assuming the bekow node is stored at 101

| type | nkeys | pointers | offsets    |            key-values           | unused |
|   2  |   2   | nil nil  |  8 19      | 2 2 "k1" "hi"  2 5 "k3" "hello" |        |
|  2B  |  2B   |   2×8B   |  2×2Bx2B   | 4B + 2B + 2B + 4B + 2B + 5B     |        |
|   Header     | nKeys*8  | nkeys*2    | KV                              |        |
*/
type BNode []byte

/*
reads bytes at inedx 0 and 1
which is a 16 bit/2 byte int
encoding the type of node
*/
func (node BNode) btype() uint16 {
	return binary.LittleEndian.Uint16(node[0:2])
}

/*
reads bytes at index 2 and 3
which is 16 bit/2 byte int
representing the number of kays
*/
func (node BNode) nkeys() uint16 {
	return binary.LittleEndian.Uint16(node[2:4])
}

/*
writes bytes at index 0 1 2 and 3
sets the type of node
and how many keays it can have
*/
func (node BNode) setHeader(btype, nkeys uint16) {
	binary.LittleEndian.PutUint16(node[0:2], btype)
	binary.LittleEndian.PutUint16(node[2:4], nkeys)
}

/*
skipes the first 4 bytes
and reads the 64 bit/8 byte address of child at index idx
*/
func (node BNode) getPtr(idx uint16) uint64 {
	if idx >= node.nkeys() {
		log.Fatalf("idx outof bound\n")
	}
	pos := HEADER_BYTES + POINTER_BYTES*idx
	return binary.LittleEndian.Uint64(node[pos:])
}

/*
skips the first 4 bytes
and updates the 64 bit/8 bytes address of child at index idx
*/
func (node BNode) setPtr(idx uint16, val uint64) {
	if idx >= node.nkeys() {
		log.Fatalf("idx outof bound\n")
	}
	pos := HEADER_BYTES + POINTER_BYTES*idx
	binary.LittleEndian.PutUint64(node[pos:], val)
}

/*
gets the offset for KV at index idx
*/
func (node BNode) getOffset(idx uint16) uint16 {
	if idx == 0 {
		return 0 // first KV pair always starts at 0
	}

	pos := HEADER_BYTES + POINTER_BYTES*node.nkeys() + OFFSET_BYTES*(idx-1) // sice we dont put offet for 0th KV pair each idx has to be subtracted by 1
	return binary.LittleEndian.Uint16(node[pos:])
}

/*
sets the offset for KV at index idx
*/
func (node BNode) setOffset(idx uint16, offset uint16) {
	if idx == 0 {
		return
	}
	if idx > node.nkeys() {
		log.Fatalf("idx out of bounds\n")
	}
	pos := HEADER_BYTES + POINTER_BYTES*node.nkeys() + OFFSET_BYTES*(idx-1)
	binary.LittleEndian.PutUint16(node[pos:], offset)
}

/*
gets the size of the node
*/
func (node BNode) NBytes() uint16 {
	return node.kvPos(node.nkeys())
}

func (node BNode) nbytes() uint16 {
	return node.NBytes()
}

/*
gets the start of KV at index idx
*/
func (node BNode) kvPos(idx uint16) uint16 {
	if idx > node.nkeys() {
		log.Fatalf("idx outof bound\n")
	}
	return HEADER_BYTES + POINTER_BYTES*node.nkeys() + OFFSET_BYTES*node.nkeys() + node.getOffset(idx)
}

func (node BNode) getKey(idx uint16) []byte {
	if idx >= node.nkeys() {
		log.Fatalf("idx outof bound\n")
	}
	pos := node.kvPos(idx)
	klen := binary.LittleEndian.Uint16(node[pos:])
	return node[pos+K_SIZE+V_SIZE:][:klen]
}

func (node BNode) getVal(idx uint16) []byte {
	if idx >= node.nkeys() {
		log.Fatalf("idx outof bound\n")
	}

	pos := node.kvPos(idx)
	klen := binary.LittleEndian.Uint16(node[pos:])
	vlen := binary.LittleEndian.Uint16(node[pos+K_SIZE:])
	return node[pos+K_SIZE+V_SIZE+klen:][:vlen]
}

// adds KV to node
func nodeAppendKV(new BNode, idx uint16, ptr uint64, key []byte, val []byte) {
	new.setPtr(idx, ptr)
	pos := new.kvPos(idx)

	binary.LittleEndian.PutUint16(new[pos:], uint16(len(key)))        // bites in key
	binary.LittleEndian.PutUint16(new[pos+K_SIZE:], uint16(len(val))) // bites in value

	copy(new[pos+K_SIZE+V_SIZE:], key)
	copy(new[pos+K_SIZE+V_SIZE+uint16(len(key)):], val)

	new.setOffset(idx+1, new.getOffset(idx)+K_SIZE+V_SIZE+uint16(len(key)+len(val))) // start position for the next node
}

// copy KV and pointers of old from idx=oldSatrt to end in new from idx=newStart to end
func nodeAppendRange(new, old BNode, newStart, oldStart, end uint16) {
	for i := range end {
		dist, src := newStart+i, oldStart+i
		nodeAppendKV(new, dist, old.getPtr(src), old.getKey(src), old.getVal(src))
	}
}

// CoW for leaf nodes
func leafInsert(new, old BNode, idx uint16, key, val []byte) {
	new.setHeader(BNODE_LEAF, old.nkeys()+1)
	nodeAppendRange(new, old, 0, 0, idx)
	nodeAppendKV(new, idx, 0, key, val)
	nodeAppendRange(new, old, idx+1, idx, old.nkeys()-idx)
}

// update a key
func leafUpdate(new, old BNode, idx uint16, key, val []byte) {
	new.setHeader(BNODE_LEAF, old.nkeys())
	nodeAppendRange(new, old, 0, 0, idx)
	nodeAppendKV(new, idx, 0, key, val)
	nodeAppendRange(new, old, idx+1, idx+1, old.nkeys()-(idx+1))
}

func leafDelete(new, old BNode, idx uint16) {
	new.setHeader(BNODE_LEAF, old.nkeys()-1)
	nodeAppendRange(new, old, 0, 0, idx)
	nodeAppendRange(new, old, idx, idx+1, old.nkeys()-(idx+1))

}

func nodeLookupLE(node BNode, key []byte) uint16 {
	nkeys := node.nkeys()
	var i uint16
	for i = 0; i < nkeys; i++ {
		cmp := bytes.Compare(node.getKey(i), key)
		if cmp == 0 {
			return i
		}
		if cmp > 0 {
			return i - 1
		}
	}
	return i - 1
}

func nodeSplit2(left, right, old BNode) {
	if old.nkeys() < 2 {
		log.Fatalf("expected nkeys to be more than 2, got %d\n", old.nkeys())
	}

	nleft := old.nkeys() / 2
	left_bytes := func() uint16 {
		return HEADER_BYTES + POINTER_BYTES*nleft + OFFSET_BYTES*nleft + old.getOffset(nleft)
	}

	for left_bytes() > BTREE_PAGE_SIZE {
		nleft--
	}

	if nleft <= 0 {
		log.Fatalf("can't slip in 2 nleft=%d\n", nleft)
	}

	right_bytes := func() uint16 {
		return old.nbytes() - left_bytes() + HEADER_BYTES
	}

	for right_bytes() > BTREE_PAGE_SIZE {
		nleft++
	}

	if nleft > old.nkeys() {
		log.Fatalf("can't slip in 2 nleft=%d\n", nleft)
	}

	nright := old.nkeys() - nleft

	left.setHeader(old.btype(), nleft)
	right.setHeader(old.btype(), nright)

	nodeAppendRange(left, old, 0, 0, nleft)
	nodeAppendRange(right, old, 0, nleft, nright)

	if right.nbytes() > BTREE_PAGE_SIZE {
		log.Fatalf("slip 2 failed right still too big %d=\n", right.nbytes())
	}
}

func nodeSplit3(old BNode) (uint16, [3]BNode) {
	if old.nbytes() <= BTREE_PAGE_SIZE {
		old = old[:BTREE_PAGE_SIZE]
		return 1, [3]BNode{old}
	}

	left := BNode(make([]byte, BTREE_PAGE_SIZE))
	right := BNode(make([]byte, BTREE_PAGE_SIZE))
	nodeSplit2(old, left, right)
	if left.nbytes() <= BTREE_PAGE_SIZE {
		left = left[:BTREE_PAGE_SIZE]
		return 2, [3]BNode{left, right}
	}

	leftleft := BNode(make([]byte, BTREE_PAGE_SIZE))
	middle := BNode(make([]byte, BTREE_PAGE_SIZE))
	nodeSplit2(leftleft, middle, left)
	if leftleft.nbytes() > BTREE_PAGE_SIZE {
		log.Fatalf("slip3 failed leftleft too big %d\n", leftleft.nbytes())
	}

	return 3, [3]BNode{leftleft, middle, right}
}

func nodeReplaceChildren(tree *BTree, new, old BNode, idx uint16, children ...BNode) {
	inc := uint16(len(children))
	new.setHeader(BNODE_NODE, old.nkeys()+inc-1) // -1 cause its 0 indexed
	nodeAppendRange(new, old, 0, 0, idx)
	for i, node := range children {
		ptr, err := tree.new(node)
		if err != nil {
			log.Fatalf("faile to make new node\n")
		}
		nodeAppendKV(new, idx+uint16(i), ptr, node.getKey(0), nil) // val is nil cause its an internal node and we only keep the primaery key of child node
	}
	nodeAppendRange(new, old, idx+inc, idx+1, old.nkeys()-(idx+1))
}

func nodeMerge(new, left, right BNode) {
	new.setHeader(right.btype(), right.nkeys()+left.nkeys())
	nodeAppendRange(new, left, 0, 0, left.nkeys())
	nodeAppendRange(new, right, 0, left.nkeys(), right.nkeys())
}

func nodeReplace2Kid(new, old BNode, idx uint16, ptr uint64, key []byte) {
	new.setHeader(old.btype(), old.nkeys()-1)
	nodeAppendRange(new, old, 0, 0, idx)
	nodeAppendKV(new, idx, ptr, key, nil)
	nodeAppendRange(new, old, idx+1, idx+2, old.nkeys()-(idx+2))
}
