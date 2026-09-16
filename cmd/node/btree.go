package node

import (
	"bytes"
	"errors"
	"log"
)

type BTree struct {
	root uint64
	get  func(uint64) ([]byte, error)
	new  func([]byte) (uint64, error)
	del  func(uint64) error
}

func treeInsert(tree *BTree, node BNode, key, val []byte) BNode {
	new := BNode(make([]byte, 2*BTREE_PAGE_SIZE)) // CoW
	idx := nodeLookupLE(node, key)
	switch node.btype() {
	case BNODE_NODE:
		children := node.getPtr(idx)
		childNode, err := tree.get(children)
		if err != nil {
			log.Fatalf("child with id = %d node not found\n", children)
		}
		knode := treeInsert(tree, childNode, key, val)
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
		var err error
		tree.root, err = tree.new(root)
		if err != nil {
			log.Fatalf("making root failed\n")
		}
		return nil
	}
	root, err := tree.get(tree.root)
	if err != nil {
		log.Fatalf("getting root with id = %d failed\n", tree.root)
	}
	node := treeInsert(tree, root, key, val)
	splits, nodes := nodeSplit3(node)
	tree.del(tree.root)

	if splits > 1 {
		root := BNode(make([]byte, BTREE_PAGE_SIZE))
		root.setHeader(BNODE_NODE, splits)
		for i, node := range nodes[:splits] {
			ptr, err := tree.new(node)
			if err != nil {
				log.Fatalf("faile to make new node\n")
			}
			key := node.getKey(0)

			nodeAppendKV(root, uint16(i), ptr, key, nil)
		}
		tree.root, err = tree.new(root)
		if err != nil {
			log.Fatalf("failed to create new root with id = %d\n", root)
		}
	} else {
		tree.root, err = tree.new(nodes[0])
		if err != nil {
			log.Fatalf("failed to create new root")
		}
	}
	return nil
}

func shouldMeger(tree *BTree, node BNode, idx uint16, update BNode) (int16, BNode) {
	if update.nbytes() > BTREE_PAGE_SIZE/4 {
		return 0, BNode{}
	}

	if idx > 0 {
		bytes, err := tree.get(node.getPtr(idx - 1))
		if err != nil {
			log.Fatalf("cant find sibling\n")
		}
		sibling := BNode(bytes)
		merged := sibling.nbytes() - update.nbytes() - HEADER_BYTES
		if merged <= BTREE_PAGE_SIZE {
			return -1, sibling
		}
	}

	if idx+1 < node.nkeys() {
		bytes, err := tree.get(node.getPtr(idx + 1))
		if err != nil {
			log.Fatalf("cant find sibling\n")
		}
		sibling := BNode(bytes)
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
	newNode, err := tree.get(kptr)
	if err != nil {
		log.Fatalf("cant find node with ptr = %d\n", kptr)
	}
	updated := treeDelete(tree, newNode, key)
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
		newNode, err := tree.new(merged)
		if err != nil {
			log.Fatalf("can't merge the nodes\n")
		}
		nodeReplace2Kid(new, node, idx-1, newNode, merged.getKey(0))
	case mergeDir > 0: // right
		merged := BNode(make([]byte, BTREE_PAGE_SIZE))
		nodeMerge(merged, updated, sibling)
		tree.del(node.getPtr(idx + 1))
		newNode, err := tree.new(merged)
		if err != nil {
			log.Fatalf("can't merge the nodes\n")
		}
		nodeReplace2Kid(new, node, idx, newNode, merged.getKey(0))
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
