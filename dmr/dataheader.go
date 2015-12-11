package dmr

import (
	"fmt"

	"github.com/pd0mz/go-dmr/bit"
)

// Data Header Packet Format
const (
	PacketFormatUDT              uint8 = iota // 0b0000
	PacketFormatResponse                      // 0b0001
	PacketFormatUnconfirmedData               // 0b0010
	PacketFormatConfirmedData                 // 0b0011
	_                                         // 0b0100
	_                                         // 0b0101
	_                                         // 0b0110
	_                                         // 0b0111
	_                                         // 0b1000
	_                                         // 0b1001
	_                                         // 0b1010
	_                                         // 0b1011
	_                                         // 0b1100
	PacketFormatShortDataDefined              // 0b1101
	PacketFormatShortDataRaw                  // 0b1110
	PacketFormatProprietaryData               // 0b1111
)

// Response Data Header Response Type
const (
	ResponseTypeACK uint8 = iota
	ResponseTypeIllegalFormat
	ResponseTypePacketCRCFailed
	ResponseTypeMemoryFull
	ResponseTypeRecvFSVNOutOfSeq
	ResponseTypeUndeliverable
	ResponseTypeRecvPktOutOfSeq
	ResponseTypeDisallowed
	ResponseTypeSelectiveACK
)

var ResponseTypeName = map[uint8]string{
	ResponseTypeACK:              "ACK",
	ResponseTypeIllegalFormat:    "illegal format",
	ResponseTypePacketCRCFailed:  "packet CRC failed",
	ResponseTypeMemoryFull:       "memory full",
	ResponseTypeRecvFSVNOutOfSeq: "recv FSN out of sequence",
	ResponseTypeUndeliverable:    "undeliverable",
	ResponseTypeRecvPktOutOfSeq:  "recv PKT our of sequence",
	ResponseTypeDisallowed:       "disallowed",
	ResponseTypeSelectiveACK:     "selective ACK",
}

// UDP Response Header UDT Format
const (
	UDTFormatBinary uint8 = iota
	UDTFormatMSAddress
	UDTFormat4BitBCD
	UDTFormatISO_7BitChars
	UDTFormatISO_8BitChars
	UDTFormatNMEALocation
	UDTFormatIPAddress
	UDTFormat16BitUnicodeChars
	UDTFormatCustomCodeD1
	UDTFormatCustomCodeD2
)

var UDTFormatName = map[uint8]string{
	UDTFormatBinary:            "binary",
	UDTFormatMSAddress:         "MS address",
	UDTFormat4BitBCD:           "4-bit BCD",
	UDTFormatISO_7BitChars:     "ISO 7-bit characters",
	UDTFormatISO_8BitChars:     "ISO 8-bit characters",
	UDTFormatNMEALocation:      "NMEA location",
	UDTFormatIPAddress:         "IP address",
	UDTFormat16BitUnicodeChars: "16-bit Unicode characters",
	UDTFormatCustomCodeD1:      "custom code D1",
	UDTFormatCustomCodeD2:      "custom code D2",
}

// UDT Response Header DD Format
const (
	DDFormatBinary uint8 = iota
	DDFormatBCD
	DDFormat7BitChar
	DDFormat8BitISO8859_1
	DDFormat8BitISO8859_2
	DDFormat8BitISO8859_3
	DDFormat8BitISO8859_4
	DDFormat8BitISO8859_5
	DDFormat8BitISO8859_6
	DDFormat8BitISO8859_7
	DDFormat8BitISO8859_8
	DDFormat8BitISO8859_9
	DDFormat8BitISO8859_10
	DDFormat8BitISO8859_11
	DDFormat8BitISO8859_13
	DDFormat8BitISO8859_14
	DDFormat8BitISO8859_15
	DDFormat8BitISO8859_16
	DDFormatUTF8
	DDFormatUTF16
	DDFormatUTF16BE
	DDFormatUTF16LE
	DDFormatUTF32
	DDFormatUTF32BE
	DDFormatUTF32LE
)

var DDFormatName = map[uint8]string{
	DDFormatBinary:         "binary",
	DDFormatBCD:            "BCD",
	DDFormat7BitChar:       "7-bit characters",
	DDFormat8BitISO8859_1:  "8-bit ISO 8859-1",
	DDFormat8BitISO8859_2:  "8-bit ISO 8859-2",
	DDFormat8BitISO8859_3:  "8-bit ISO 8859-3",
	DDFormat8BitISO8859_4:  "8-bit ISO 8859-4",
	DDFormat8BitISO8859_5:  "8-bit ISO 8859-5",
	DDFormat8BitISO8859_6:  "8-bit ISO 8859-6",
	DDFormat8BitISO8859_7:  "8-bit ISO 8859-7",
	DDFormat8BitISO8859_8:  "8-bit ISO 8859-8",
	DDFormat8BitISO8859_9:  "8-bit ISO 8859-9",
	DDFormat8BitISO8859_10: "8-bit ISO 8859-10",
	DDFormat8BitISO8859_11: "8-bit ISO 8859-11",
	DDFormat8BitISO8859_13: "8-bit ISO 8859-13",
	DDFormat8BitISO8859_14: "8-bit ISO 8859-14",
	DDFormat8BitISO8859_15: "8-bit ISO 8859-15",
	DDFormat8BitISO8859_16: "8-bit ISO 8859-16",
	DDFormatUTF8:           "UTF-8",
	DDFormatUTF16:          "UTF-16",
	DDFormatUTF16BE:        "UTF-16 big endian",
	DDFormatUTF16LE:        "UTF-16 little endian",
	DDFormatUTF32:          "UTF-32",
	DDFormatUTF32BE:        "UTF-32 big endian",
	DDFormatUTF32LE:        "UTF-32 little endian",
}

type DataHeader interface {
	CommonHeader() DataHeaderCommon
}

type DataHeaderCommon struct {
	PacketFormat       uint8
	DstIsGroup         bool
	ResponseRequested  bool
	ServiceAccessPoint uint8
	DstID              uint32
	SrcID              uint32
	CRC                uint16
}

func (h *DataHeaderCommon) Parse(header []byte) error {
	h.PacketFormat = header[0] & 0xf
	h.DstIsGroup = (header[0] & 0x80) > 0
	h.ResponseRequested = (header[0] & 0x40) > 0
	h.ServiceAccessPoint = (header[1] & 0xf0) >> 4
	h.DstID = uint32(header[2])<<16 | uint32(header[3])<<8 | uint32(header[4])
	h.SrcID = uint32(header[5])<<16 | uint32(header[6])<<8 | uint32(header[7])
	return nil
}

type UDTDataHeader struct {
	Common            DataHeaderCommon
	Format            uint8
	PadNibble         uint8
	AppendedBlocks    uint8
	SupplementaryFlag bool
	OPCode            uint8
}

func (h UDTDataHeader) CommonHeader() DataHeaderCommon { return h.Common }

type UnconfirmedDataHeader struct {
	Common                 DataHeaderCommon
	PadOctetCount          uint8
	FullMessage            bool
	BlocksToFollow         uint8
	FragmentSequenceNumber uint8
}

func (h UnconfirmedDataHeader) CommonHeader() DataHeaderCommon { return h.Common }

type ConfirmedDataHeader struct {
	Common                 DataHeaderCommon
	PadOctetCount          uint8
	FullMessage            bool
	BlocksToFollow         uint8
	Resync                 bool
	SendSequenceNumber     uint8
	FragmentSequenceNumber uint8
}

func (h ConfirmedDataHeader) CommonHeader() DataHeaderCommon { return h.Common }

type ResponseDataHeader struct {
	Common         DataHeaderCommon
	BlocksToFollow uint8
	Class          uint8
	Type           uint8
	Status         uint8
}

func (h ResponseDataHeader) CommonHeader() DataHeaderCommon { return h.Common }

type ProprietaryDataHeader struct {
	Common         DataHeaderCommon
	ManufacturerID uint8
}

func (h ProprietaryDataHeader) CommonHeader() DataHeaderCommon { return h.Common }

type ShortDataRawDataHeader struct {
	Common         DataHeaderCommon
	AppendedBlocks uint8
	SrcPort        uint8
	DstPort        uint8
	Resync         bool
	FullMessage    bool
	BitPadding     uint8
}

func (h ShortDataRawDataHeader) CommonHeader() DataHeaderCommon { return h.Common }

type ShortDataDefinedDataHeader struct {
	Common         DataHeaderCommon
	AppendedBlocks uint8
	DDFormat       uint8
	Resync         bool
	FullMessage    bool
	BitPadding     uint8
}

func (h ShortDataDefinedDataHeader) CommonHeader() DataHeaderCommon { return h.Common }

func ParseDataHeader(header []byte, proprietary bool) (DataHeader, error) {
	if len(header) != 12 {
		return nil, fmt.Errorf("header must be 12 bytes, got %d", len(header))
	}
	var (
		ccrc = (uint16(header[10]) << 8) | uint16(header[11])
		hcrc = dataHeaderCRC(header)
	)
	if ccrc != hcrc {
		return nil, fmt.Errorf("header CRC mismatch, %#04x != %#04x", ccrc, hcrc)
	}

	if proprietary {
		return ProprietaryDataHeader{
			Common: DataHeaderCommon{
				ServiceAccessPoint: (header[0] & bit.B11110000) >> 4,
				PacketFormat:       (header[0] & bit.B00001111),
				CRC:                ccrc,
			},
			ManufacturerID: header[1],
		}, nil
	}

	common := DataHeaderCommon{
		CRC: ccrc,
	}
	if err := common.Parse(header); err != nil {
		return nil, err
	}

	switch common.PacketFormat {
	case PacketFormatUDT:
		return UDTDataHeader{
			Common:            common,
			Format:            (header[1] & bit.B00001111),
			PadNibble:         (header[8] & bit.B11111000) >> 3,
			AppendedBlocks:    (header[8] & bit.B00000011),
			SupplementaryFlag: (header[9] & bit.B10000000) > 0,
			OPCode:            (header[9] & bit.B00111111),
		}, nil
	case PacketFormatResponse:
		return ResponseDataHeader{
			Common:         common,
			BlocksToFollow: (header[8] & bit.B01111111),
			Class:          (header[9] & bit.B11000000) >> 6,
			Type:           (header[9] & bit.B00111000) >> 3,
			Status:         (header[9] & bit.B00000111),
		}, nil
	case PacketFormatUnconfirmedData:
		return UnconfirmedDataHeader{
			Common:                 common,
			PadOctetCount:          (header[0] & bit.B00010000) | (header[1] & bit.B00001111),
			FullMessage:            (header[8] & bit.B10000000) > 0,
			BlocksToFollow:         (header[8] & bit.B01111111),
			FragmentSequenceNumber: (header[9] & bit.B00001111),
		}, nil
	case PacketFormatConfirmedData:
		return ConfirmedDataHeader{
			Common:                 common,
			PadOctetCount:          (header[0] & bit.B00010000) | (header[1] & bit.B00001111),
			FullMessage:            (header[8] & bit.B10000000) > 0,
			BlocksToFollow:         (header[8] & bit.B01111111),
			Resync:                 (header[9] & bit.B10000000) > 0,
			SendSequenceNumber:     (header[9] & bit.B01110000) >> 4,
			FragmentSequenceNumber: (header[9] & bit.B00001111),
		}, nil
	case PacketFormatShortDataRaw:
		return ShortDataRawDataHeader{
			Common:         common,
			AppendedBlocks: (header[0] & bit.B00110000) | (header[1] & bit.B00001111),
			SrcPort:        (header[8] & bit.B11100000) >> 5,
			DstPort:        (header[8] & bit.B00011100) >> 2,
			Resync:         (header[8] & bit.B00000010) > 0,
			FullMessage:    (header[8] & bit.B00000001) > 0,
			BitPadding:     (header[9]),
		}, nil
	case PacketFormatShortDataDefined:
		return ShortDataDefinedDataHeader{
			Common:         common,
			AppendedBlocks: (header[0] & bit.B00110000) | (header[1] & bit.B00001111),
			DDFormat:       (header[8] & bit.B11111100) >> 2,
			Resync:         (header[8] & bit.B00000010) > 0,
			FullMessage:    (header[8] & bit.B00000001) > 0,
			BitPadding:     (header[9]),
		}, nil
	default:
		return nil, fmt.Errorf("dmr: unknown data header packet format %#02x (%d)", common.PacketFormat, common.PacketFormat)
	}
}

func dataHeaderCRC(header []byte) uint16 {
	var crc uint16
	if len(header) < 10 {
		return crc
	}

	for i := 0; i < 10; i++ {
		crc16(&crc, header[i])
	}
	crc16end(&crc)

	return (^crc) ^ 0xcccc
}

func crc16(crc *uint16, b byte) {
	var v = uint8(0x80)
	for i := 0; i < 8; i++ {
		xor := ((*crc) & 0x8000) > 0
		(*crc) <<= 1
		if b&v > 0 {
			(*crc)++
		}
		if xor {
			(*crc) ^= 0x1021
		}
		v >>= 1
	}
}

func crc16end(crc *uint16) {
	for i := 0; i < 16; i++ {
		xor := ((*crc) & 0x8000) > 0
		(*crc) <<= 1
		if xor {
			(*crc) ^= 0x1021
		}
	}
}
