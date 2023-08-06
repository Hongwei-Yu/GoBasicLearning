package goroutinepool

import "testing"

func TestPool_Close(t *testing.T) {
	tests := []struct {
		name string
		p    *Pool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.p.Close()
		})
	}
}
