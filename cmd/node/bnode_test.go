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
