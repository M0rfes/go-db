package node

import "encoding/binary"

const (
	HEADER_BYTES  = 4
	POINTER_BYTES = 8
	OFFSET_BYTES  = 2
	K_SIZE        = 2
	V_SIZE        = 2
)

/*
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
		panic("idx outof bound")
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
		panic("idx outof bound")
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
		panic("idx out of bounds")
	}
	pos := HEADER_BYTES + POINTER_BYTES*node.nkeys() + OFFSET_BYTES*(idx-1)
	binary.LittleEndian.PutUint16(node[pos:], offset)
}

/*
gets the size of the node
*/
func (node BNode) nbytes() uint16 {
	return node.kvPos(node.nkeys())
}

/*
gets the start of KV at index idx
*/
func (node BNode) kvPos(idx uint16) uint16 {
	if idx > node.nkeys() {
		panic("idx outof bound")
	}
	return HEADER_BYTES + POINTER_BYTES*node.nkeys() + OFFSET_BYTES*node.nkeys() + node.getOffset(idx)
}

func (node BNode) getKey(idx uint16) []byte {
	if idx >= node.nkeys() {
		panic("idx outof bound")
	}
	pos := node.kvPos(idx)
	klen := binary.LittleEndian.Uint16(node[pos:])
	return node[pos+K_SIZE+V_SIZE:][:klen]
}

func (node BNode) getVal(idx uint16) []byte {
	if idx >= node.nkeys() {
		panic("idx outof bound")
	}

	pos := node.kvPos(idx)
	klen := binary.LittleEndian.Uint16(node[pos:])
	vlen := binary.LittleEndian.Uint16(node[pos+K_SIZE:])
	return node[pos+K_SIZE+V_SIZE+klen:][:vlen]
}

func nodeAppendKV(new BNode, idx uint16, ptr uint64, key []byte, val []byte) {
	new.setPtr(idx, ptr)
	pos := new.kvPos(idx)

	binary.LittleEndian.PutUint16(new[pos:], uint16(len(key)))        // bites in key
	binary.LittleEndian.PutUint16(new[pos+K_SIZE:], uint16(len(val))) // bites in value

	copy(new[pos+K_SIZE+V_SIZE:], key)
	copy(new[pos+K_SIZE+V_SIZE+uint16(len(key)):], val)

	new.setOffset(idx+1, new.getOffset(idx)+K_SIZE+V_SIZE+uint16(len(key)+len(val))) // start position for the next node
}
