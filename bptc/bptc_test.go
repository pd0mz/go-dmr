package bptc

import (
	"math/rand"
	"pd0mz/go-dmr/bit"
)
import "testing"

func Test(t *testing.T) {
	for i := 0; i < 100; i++ {
		test := make(bit.Bits, 96)
		for b := 0; b < 96; b++ {
			if rand.Uint32() > 0x7fffffff {
				test[b].Flip()
			}
		}
		bptc := New(test)
		if len(bptc) != 196 {
			t.Fatalf("expected 196 bits, got %d", len(bptc))
		}
		if testing.Verbose() {
			Dump(bptc)
		}
		corrupt := rand.Intn(10)
		bptc[corrupt].Flip()
		if ok, err := CheckAndRepair(bptc); !ok {
			t.Fatalf("check and repair failed: %v", err)
		}
		bptc[corrupt].Flip()
		back := Extract(bptc)
		if len(test) != len(back) {
			t.Fatalf("expected %d bits, got %d", len(test), len(back))
		}
	}
}
