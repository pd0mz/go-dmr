package bit

import "testing"

func TestBit(t *testing.T) {
	var tests = []struct {
		Test []byte
		Want Bits
	}{
		{
			[]byte{0x2a},
			Bits{0, 0, 1, 0, 1, 0, 1, 0},
		},
		{
			[]byte{0xbe, 0xef},
			Bits{1, 0, 1, 1, 1, 1, 1, 0, 1, 1, 1, 0, 1, 1, 1, 1},
		},
	}

	for _, test := range tests {
		got := NewBits(test.Test)
		if len(got) != len(test.Want) {
			t.Fatalf("expected length %d, got %d [%s]", len(test.Want), len(got), got.String())
		}
		for i, b := range got {
			if b != test.Want[i] {
				t.Fatalf("bit %d is off: %v != %v", i, got, test.Want)
			}
		}
	}
}
