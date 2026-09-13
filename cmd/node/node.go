package node

const (
	BNODE_NODE = 1
	BNODE_LEAF = 2
)

const (
	BTREE_PAGE_SIZE    = 4096 // 4KB
	BTREE_MAX_KEY_SIZE = 1000 // size in bits
	BTREE_MAX_VAL_SIZE = 3000 // size in bits
)

type Node struct {
	keys     [][]byte
	vals     [][]byte // only leaf nodes will have value
	childres []*Node
}
