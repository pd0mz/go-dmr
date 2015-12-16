package dmr

import (
	"bytes"
	"testing"
)

func TestBit(t *testing.T) {
	var tests = []struct {
		Test []byte
		Want []byte
	}{
		{
			[]byte{0x2a},
			[]byte{0, 0, 1, 0, 1, 0, 1, 0},
		},
		{
			[]byte{0xbe, 0xef},
			[]byte{1, 0, 1, 1, 1, 1, 1, 0, 1, 1, 1, 0, 1, 1, 1, 1},
		},
	}

	for _, test := range tests {
		got := BytesToBits(test.Test)
		if len(got) != len(test.Want) {
			t.Fatalf("expected length %d, got %d [%s]", len(test.Want), len(got), string(got))
		}
		for i, b := range got {
			if b != test.Want[i] {
				t.Fatalf("bit %d is off: %v != %v", i, got, test.Want)
			}
		}

		rev := BitsToBytes(got)
		if !bytes.Equal(rev, test.Test) {
			t.Fatalf("reverse bits to bytes failed, %v != %v", rev, test.Test)
		}
	}
}
