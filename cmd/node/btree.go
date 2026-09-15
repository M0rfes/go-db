package node

import (
	"bytes"
	"log"
)

type BTree struct {
	root uint64
	get  func(uint64) []byte
	new  func([]byte) uint64
	del  func(uint64)
}

func treeInsert(tree *BTree, node BNode, key, val []byte) BNode {
	new := BNode(make([]byte, 2*BTREE_PAGE_SIZE))
	idx := nodeLookupLE(node, key)
	switch node.btype() {
	case BNODE_NODE:
		children := node.getPtr(idx)
		knode := treeInsert(tree, tree.get(children), key, val)
		slipts, nodes := nodeSplit3(knode)
		tree.del(children)
		nodeReplaceChildren(tree, new, node, idx, nodes[:slipts]...)

	case BNODE_LEAF:
		if bytes.Equal(key, node.getKey(idx)) {
			leafUpdate(new, node, idx, key, val)
		} else {
			leafInsert(new, node, idx+1, key, val)
		}
	default:
		log.Fatalf("unknown node type %d=\n", node.btype())
	}

	return new
}
