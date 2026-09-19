package inmemory

import (
	"errors"
	"fmt"
	"unsafe"

	"github.com/M0rfes/go-db/node"
)

type InMemory struct {
	Tree  *node.BTree
	Ref   map[string]string
	pages map[uint64]node.BNode
}

func New() *InMemory {
	pages := map[uint64]node.BNode{}
	return &InMemory{
		Tree: node.NewBTree(
			func(ptr uint64) ([]byte, error) {
				bnode, ok := pages[ptr]
				if !ok {
					return nil, errors.New(fmt.Sprintf("page with ptr = %d not found\n", ptr))
				}
				return bnode, nil
			},
			func(bytes []byte) (uint64, error) {
				bnode := node.BNode(bytes)
				if bnode.NBytes() > node.BTREE_PAGE_SIZE {
					return 0, errors.New(fmt.Sprintf("new node has more types then allowd %d\n", bnode.NBytes()))
				}
				ptr := uint64(uintptr(unsafe.Pointer(&bnode[0])))
				if pages[ptr] != nil {
					return 0, errors.New(fmt.Sprintf("ptr = %d alredy in use\n", ptr))
				}
				pages[ptr] = bnode
				return ptr, nil
			},
			func(ptr uint64) error {
				if pages[ptr] == nil {
					return errors.New(fmt.Sprintf("cant delte node with ptr = %d\n", ptr))
				}
				delete(pages, ptr)
				return nil
			},
		),
		Ref:   make(map[string]string),
		pages: pages,
	}
}

func (m *InMemory) Add(key, val string) error {
	err := m.Tree.Insert([]byte(key), []byte(val))
	if err == nil {
		m.Ref[key] = val
	}
	return err
}

func (m *InMemory) Get(key string) (string, bool) {
	val, ok := m.Tree.Get([]byte(key))
	if !ok {
		return "", false
	}
	return string(val), true
}

func (m *InMemory) Delete(key string) bool {
	ok := m.Tree.Delete([]byte(key))
	if ok {
		delete(m.Ref, key)
	}
	return ok
}
