package bptc

import (
	"bytes"
	"encoding/hex"
	"testing"

	"github.com/pd0mz/go-dmr"
)

var (
	encoded = []byte{
		0x4b, 0xb2, 0x1d, 0x6d, 0x82, 0xd4,
		0x23, 0x34, 0x0e, 0xe9, 0x66, 0xf3,
		0xc2, 0x20, 0xc3, 0x87, 0xfd, 0x84,
		0x54, 0x12, 0x4d, 0xb2, 0xd1, 0x40,
		0x70,
	}
	decoded = []byte{
		0xbd, 0x00, 0x80, 0x03, 0x1f, 0x29,
		0x66, 0x1f, 0x2c, 0xa4, 0x66, 0x7e,
	}
)

func TestDecode(t *testing.T) {
	var want = dmr.BytesToBits(encoded)
	var test = make([]byte, 12)

	if err := Decode(want, test); err != nil {
		t.Fatalf("decode failed: %v", err)
	}

	if !bytes.Equal(test, decoded) {
		t.Fatalf("decode failed: not equal")
	}
	t.Logf("input:\n%s", hex.Dump(encoded))
	t.Logf("decoded:\n%s", hex.Dump(test))
}

func TestEncode(t *testing.T) {
	var want = make([]byte, 12)
	var bits = make([]byte, 196)
	copy(want, decoded)

	if err := Encode(want, bits); err != nil {
		t.Fatalf("encode failed: %v", err)
	}

	test := dmr.BitsToBytes(bits)
	if !bytes.Equal(test, encoded) {
		t.Fatalf("encode failed: not equal")
	}

	t.Logf("input:\n%s", hex.Dump(decoded))
	t.Logf("encoded:\n%s", hex.Dump(test))
}
