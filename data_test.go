package dmr

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"sort"
	"testing"
)

func TestDataBlock(t *testing.T) {
	want := &DataBlock{
		Serial: 123,
		Data:   []byte{0x17, 0x2a},
		Length: 2,
	}

	data := want.Bytes(Rate34Data, true)
	if data == nil {
		t.Fatal("encode failed")
	}
	size := int(dataBlockLength(Rate34Data, true))
	if len(data) != size {
		t.Fatalf("encode failed: expected %d bytes, got %d", size, len(data))
	}

	// Decoding is tested in the DataFragment test
}

func TestDataFragment(t *testing.T) {
	msg, err := BuildMessageData("CQCQCQ PD0MZ", DDFormatUTF16, true)
	if err != nil {
		t.Fatalf("build message failed: %v", err)
	}

	want := &DataFragment{Data: msg}
	blocks, err := want.DataBlocks(Rate34Data, true)
	if err != nil {
		t.Fatalf("encode failed: %v", err)
	}
	if blocks == nil {
		t.Fatal("encode failed: blocks is nil")
	}
	if len(blocks) != 2 {
		t.Fatalf("encode failed: expected 2 blocks, got %d", len(blocks))
	}

	for i, block := range blocks {
		t.Log(fmt.Sprintf("block %02d:\n%s", i, hex.Dump(block.Data)))
	}

	test, err := CombineDataBlocks(blocks)
	if err != nil {
		t.Fatalf("decode failed: %v", err)
	}

	if !bytes.Equal(test.Data[:len(want.Data)], want.Data) {
		t.Log(fmt.Sprintf("want:\n%s", hex.Dump(want.Data)))
		t.Log(fmt.Sprintf("got:\n%s", hex.Dump(test.Data)))
		t.Fatal("decode failed: data is wrong")
	}
}

func TestMessage(t *testing.T) {
	msg := "CQCQCQ PD0MZ"

	var encodings = []int{}
	for i := range encodingMap {
		encodings = append(encodings, int(i))
	}
	sort.Sort(sort.IntSlice(encodings))

	for _, i := range encodings {
		e := encodingMap[uint8(i)]
		n := DDFormatName[uint8(i)]
		t.Logf("testing %s encoding", n)

		enc := e.NewDecoder()
		str, err := enc.String(msg)
		if err != nil {
			t.Fatalf("error encoding to %s: %v", n, err)
		}

		dec := e.NewDecoder()
		out, err := dec.String(str)
		if err != nil {
			t.Fatalf("error decoding from %s: %v", n, err)
		}

		t.Log(fmt.Sprintf("encoder:\n%s", hex.Dump([]byte(str))))
		t.Log(fmt.Sprintf("decoder:\n%s", hex.Dump([]byte(out))))
	}
}
