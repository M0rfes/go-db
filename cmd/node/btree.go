package node

import (
	"bytes"
	"errors"
	"log"
)

type BTree struct {
	root uint64
	get  func(uint64) []byte
	new  func([]byte) uint64
	del  func(uint64)
}

func treeInsert(tree *BTree, node BNode, key, val []byte) BNode {
	new := BNode(make([]byte, 2*BTREE_PAGE_SIZE)) // CoW
	idx := nodeLookupLE(node, key)
	switch node.btype() {
	case BNODE_NODE:
		children := node.getPtr(idx)
		knode := treeInsert(tree, tree.get(children), key, val)
		slipts, nodes := nodeSplit3(knode)
		tree.del(children) // CoW
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

func (tree *BTree) Insert(key []byte, val []byte) error {
	if len(key) > BTREE_MAX_KEY_SIZE || len(val) > BTREE_MAX_VAL_SIZE {
		return errors.New("key or val too big")
	}

	if tree.root == 0 {
		root := BNode(make([]byte, BTREE_PAGE_SIZE))
		root.setHeader(BNODE_LEAF, 2)
		nodeAppendKV(root, 0, 0, nil, nil)
		nodeAppendKV(root, 1, 0, key, val)
		tree.root = tree.new(root)
		return nil
	}
	node := treeInsert(tree, tree.get(tree.root), key, val)
	splits, nodes := nodeSplit3(node)
	tree.del(tree.root)

	if splits > 1 {
		root := BNode(make([]byte, BTREE_PAGE_SIZE))
		root.setHeader(BNODE_NODE, splits)
		for i, node := range nodes[:splits] {
			ptr, key := tree.new(node), node.getKey(0)
			nodeAppendKV(root, uint16(i), ptr, key, nil)
		}
		tree.root = tree.new(root)
	} else {
		tree.root = tree.new(nodes[0])
	}
	return nil
}

func shouldMeger(tree *BTree, node BNode, idx uint16, update BNode) (int16, BNode) {
	if update.nbytes() > BTREE_PAGE_SIZE/4 {
		return 0, BNode{}
	}

	if idx > 0 {
		sibling := BNode(tree.get(node.getPtr(idx - 1)))
		merged := sibling.nbytes() - update.nbytes() - HEADER_BYTES
		if merged <= BTREE_PAGE_SIZE {
			return -1, sibling
		}
	}

	if idx+1 < node.nkeys() {
		sibling := BNode(tree.get(node.getPtr(idx + 1)))
		merged := sibling.nbytes() - update.nbytes() - HEADER_BYTES
		if merged <= BTREE_PAGE_SIZE {
			return 1, sibling
		}
	}

	return 0, BNode{}
}

func treeDelete(tree *BTree, node BNode, key []byte) BNode {
	idx := nodeLookupLE(node, key)
	switch node.btype() {
	case BNODE_LEAF:
		if !bytes.Equal(node.getKey(idx), key) {
			return BNode{}
		}
		new := BNode(make([]byte, BTREE_PAGE_SIZE))
		leafDelete(new, node, idx)
		return new
	case BNODE_NODE:
		return nodeDelete(tree, node, idx, key)
	}
	return BNode{}
}

func nodeDelete(tree *BTree, node BNode, idx uint16, key []byte) BNode {
	// recurse into the kid
	kptr := node.getPtr(idx)
	updated := treeDelete(tree, tree.get(kptr), key)
	if len(updated) == 0 {
		return BNode{} // not found
	}
	tree.del(kptr)
	// check for merging
	new := BNode(make([]byte, BTREE_PAGE_SIZE))
	mergeDir, sibling := shouldMeger(tree, node, idx, updated)
	switch {
	case mergeDir < 0: // left
		merged := BNode(make([]byte, BTREE_PAGE_SIZE))
		nodeMerge(merged, sibling, updated)
		tree.del(node.getPtr(idx - 1))
		nodeReplace2Kid(new, node, idx-1, tree.new(merged), merged.getKey(0))
	case mergeDir > 0: // right
		merged := BNode(make([]byte, BTREE_PAGE_SIZE))
		nodeMerge(merged, updated, sibling)
		tree.del(node.getPtr(idx + 1))
		nodeReplace2Kid(new, node, idx, tree.new(merged), merged.getKey(0))
	case mergeDir == 0 && updated.nkeys() == 0:
		if node.nkeys() != 1 && idx != 0 {
			// 1 empty child but no sibling
			log.Fatalf("1 empty child but no sibling\n")
		}
		new.setHeader(BNODE_NODE, 0) // the parent becomes empty too
	case mergeDir == 0 && updated.nkeys() > 0: // no merge
		nodeReplaceChildren(tree, new, node, idx, updated)
	}
	return new
}
