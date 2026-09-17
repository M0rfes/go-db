package node

import (
	"errors"
	"fmt"
	"unsafe"
)

type inMemory struct {
	tree  BTree
	ref   map[string]string
	pages map[uint64]BNode
}

func New() *inMemory {
	pages := map[uint64]BNode{}
	return &inMemory{
		tree: BTree{
			get: func(ptr uint64) ([]byte, error) {
				node, ok := pages[ptr]
				if !ok {
					return nil, errors.New(fmt.Sprintf("page with ptr = %d not found\n", ptr))
				}
				return node, nil
			},
			new: func(bytes []byte) (uint64, error) {
				node := BNode(bytes)
				if node.nbytes() > BTREE_PAGE_SIZE {
					return 0, errors.New(fmt.Sprintf("new node has more types then allowd %d\n", node.nbytes()))
				}
				ptr := uint64(uintptr(unsafe.Pointer(&node[0])))
				if pages[ptr] != nil {
					return 0, errors.New(fmt.Sprintf("ptr = %d alredy in use\n", ptr))
				}
				pages[ptr] = node
				return ptr, nil
			},
			del: func(ptr uint64) error {
				if pages[ptr] == nil {
					return errors.New(fmt.Sprintf("cant delte node with ptr = %d\n", ptr))
				}
				return nil
			},
		},
		ref:   make(map[string]string),
		pages: pages,
	}
}

func (inMemory *inMemory) Add(key, val string) {
	inMemory.tree.Insert([]byte(key), []byte(val))
	inMemory.ref[key] = val
}
