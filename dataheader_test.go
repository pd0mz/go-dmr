package dmr

import (
	"encoding/hex"
	"testing"
)

func testDataHeader(want *DataHeader, t *testing.T) *DataHeader {
	want.SrcID = 2042214
	want.DstID = 2043044

	t.Logf("encode:\n%s", want.String())

	data, err := want.Bytes()
	if err != nil {
		t.Fatalf("encode failed: %v", err)
	}
	t.Logf("encoded:\n%s", hex.Dump(data))

	test, err := ParseDataHeader(data, false)
	if err != nil {
		t.Fatalf("decode failed: %v", err)
	}
	if test.SrcID != want.SrcID || test.DstID != want.DstID {
		t.Fatal("decode failed, ID wrong")
	}
	t.Logf("decoded:\n%s", test.String())

	return test
}

func TestDataHeaderUDT(t *testing.T) {
	want := &DataHeader{
		PacketFormat: PacketFormatUDT,
		Data: &UDTData{
			Format:         UDTFormatIPAddress,
			PadNibble:      2,
			AppendedBlocks: 3,
			Opcode:         4,
		},
	}
	test := testDataHeader(want, t)

	d, ok := test.Data.(*UDTData)
	switch {
	case !ok:
		t.Fatalf("decode failed: expected UDTData, got %T", test.Data)

	case d.Format != UDTFormatIPAddress:
		t.Fatalf("decode failed: format wrong")

	case d.PadNibble != 2:
		t.Fatalf("decode failed: pad nibble wrong")

	case d.AppendedBlocks != 3:
		t.Fatalf("decode failed: appended blocks wrong")

	case d.Opcode != 4:
		t.Fatalf("decode failed: opcode wrong")
	}
}

func TestDataHeaderResponse(t *testing.T) {
	want := &DataHeader{
		PacketFormat: PacketFormatResponse,
		Data: &ResponseData{
			BlocksToFollow: 0x10,
			ClassType:      ResponseTypeSelectiveACK,
		},
	}
	test := testDataHeader(want, t)

	d, ok := test.Data.(*ResponseData)
	switch {
	case !ok:
		t.Fatalf("decode failed: expected ResponseData, got %T", test.Data)

	case d.BlocksToFollow != 0x10:
		t.Fatalf("decode failed: wrong blocks %d, expected 16", d.BlocksToFollow)

	case d.ClassType != ResponseTypeSelectiveACK:
		t.Fatalf("decode failed: wrong type %s (%d), expected selective ACK", ResponseTypeName[d.ClassType], d.ClassType)
	}
}

func TestDataHeaderUnconfirmedData(t *testing.T) {
	want := &DataHeader{
		PacketFormat: PacketFormatUnconfirmedData,
		Data: &UnconfirmedData{
			PadOctetCount:          2,
			FullMessage:            true,
			BlocksToFollow:         5,
			FragmentSequenceNumber: 3,
		},
	}
	test := testDataHeader(want, t)

	d, ok := test.Data.(*UnconfirmedData)
	switch {
	case !ok:
		t.Fatalf("decode failed: expected UnconfirmedData, got %T", test.Data)

	case d.PadOctetCount != 2:
		t.Fatalf("decode failed: pad octet count wrong")

	case !d.FullMessage:
		t.Fatalf("decode failed: full message bit wrong")

	case d.BlocksToFollow != 5:
		t.Fatalf("decode failed: blocks to follow wrong")

	case d.FragmentSequenceNumber != 3:
		t.Fatalf("decode failed: fragment sequence number wrong")
	}
}

func TestDataHeaderConfirmedData(t *testing.T) {
	want := &DataHeader{
		PacketFormat: PacketFormatConfirmedData,
		Data: &ConfirmedData{
			PadOctetCount:          2,
			FullMessage:            true,
			BlocksToFollow:         5,
			SendSequenceNumber:     4,
			FragmentSequenceNumber: 3,
		},
	}
	test := testDataHeader(want, t)

	d, ok := test.Data.(*ConfirmedData)
	switch {
	case !ok:
		t.Fatalf("decode failed: expected ConfirmedData, got %T", test.Data)

	case d.PadOctetCount != 2:
		t.Fatalf("decode failed: pad octet count wrong")

	case !d.FullMessage:
		t.Fatalf("decode failed: full message bit wrong")

	case d.Resync:
		t.Fatalf("decode failed: resync bit wrong")

	case d.BlocksToFollow != 5:
		t.Fatalf("decode failed: blocks to follow wrong")

	case d.SendSequenceNumber != 4:
		t.Fatalf("decode failed: fragment sequence number wrong")

	case d.FragmentSequenceNumber != 3:
		t.Fatalf("decode failed: fragment sequence number wrong")
	}
}

func TestDataHeaderShortDataRaw(t *testing.T) {
	want := &DataHeader{
		PacketFormat: PacketFormatShortDataRaw,
		Data: &ShortDataRawData{
			AppendedBlocks: 3,
			SrcPort:        4,
			DstPort:        5,
			FullMessage:    true,
			BitPadding:     2,
		},
	}
	test := testDataHeader(want, t)

	d, ok := test.Data.(*ShortDataRawData)
	switch {
	case !ok:
		t.Fatalf("decode failed: expected ShortDataRawData, got %T", test.Data)

	case d.AppendedBlocks != 3:
		t.Fatalf("decode failed: appended blocks wrong")

	case d.SrcPort != 4:
		t.Fatalf("decode failed: src port wrong")

	case d.DstPort != 5:
		t.Fatalf("decode failed: dst port wrong")

	case d.Resync:
		t.Fatalf("decode failed: rsync bit wrong")

	case !d.FullMessage:
		t.Fatalf("decode failed: full message bit wrong")

	case d.BitPadding != 2:
		t.Fatalf("decode failed: bit padding wrong")
	}
}

func TestDataHeaderShortDataDefined(t *testing.T) {
	want := &DataHeader{
		PacketFormat: PacketFormatShortDataDefined,
		Data: &ShortDataDefinedData{
			AppendedBlocks: 3,
			DDFormat:       DDFormatUTF16,
			FullMessage:    true,
			BitPadding:     2,
		},
	}
	test := testDataHeader(want, t)

	d, ok := test.Data.(*ShortDataDefinedData)
	switch {
	case !ok:
		t.Fatalf("decode failed: expected ShortDataDefinedData, got %T", test.Data)

	case d.AppendedBlocks != 3:
		t.Fatalf("decode failed: appended blocks wrong")

	case d.DDFormat != DDFormatUTF16:
		t.Fatalf("decode failed: dd format wrong, expected UTF-16, got %s", DDFormatName[d.DDFormat])

	case d.Resync:
		t.Fatalf("decode failed: rsync bit wrong")

	case !d.FullMessage:
		t.Fatalf("decode failed: full message bit wrong")

	case d.BitPadding != 2:
		t.Fatalf("decode failed: bit padding wrong")
	}
}
