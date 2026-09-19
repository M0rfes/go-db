package node

import "testing"

func Test_nodeAppendRange(t *testing.T) {
	tests := []struct {
		name string // description of this test case
		// Named input parameters for target function.
		new   BNode
		old   BNode
		dist  uint16
		src   uint16
		count uint16
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nodeAppendRange(tt.new, tt.old, tt.dist, tt.src, tt.count)
		})
	}
}

func Test_nodeLookupLE(t *testing.T) {
	// Create a node with keys: "", "b", "d", "f"
	node := BNode(make([]byte, BTREE_PAGE_SIZE))
	node.setHeader(BNODE_LEAF, 4)
	nodeAppendKV(node, 0, 0, nil, nil) // dummy key ""
	nodeAppendKV(node, 1, 0, []byte("b"), []byte("val_b"))
	nodeAppendKV(node, 2, 0, []byte("d"), []byte("val_d"))
	nodeAppendKV(node, 3, 0, []byte("f"), []byte("val_f"))

	cases := []struct {
		key      string
		expected uint16
	}{
		{"", 0},  // exact match on dummy key
		{"a", 0}, // between "" and "b" -> returns index 0 ("")
		{"b", 1}, // exact match on "b"
		{"c", 1}, // between "b" and "d" -> returns index 1 ("b")
		{"d", 2}, // exact match on "d"
		{"e", 2}, // between "d" and "f" -> returns index 2 ("d")
		{"f", 3}, // exact match on "f"
		{"g", 3}, // larger than all -> returns index 3 ("f")
	}

	for _, c := range cases {
		got := nodeLookupLE(node, []byte(c.key))
		if got != c.expected {
			t.Errorf("nodeLookupLE(%q) = %d; want %d", c.key, got, c.expected)
		}
	}
}

