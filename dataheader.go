package dmr

import (
	"fmt"
	"strings"
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

// Service Access Point
const (
	ServiceAccessPointUDT                    uint8 = iota // 0b0000
	_                                                     // 0b0001
	ServiceAccessPointTCPIPHeaderCompression              // 0b0010
	ServiceAccessPointUDPIPHeaderCompression              // 0b0011
	ServiceAccessPointIPBasedPacketData                   // 0b0100
	ServiceAccessPointARP                                 // 0b0101
	_                                                     // 0b0110
	_                                                     // 0b0111
	_                                                     // 0b1000
	ServiceAccessPointProprietaryData                     // 0b1001
	ServiceAccessPointShortData                           // 0b1010
)

var ServiceAccessPointName = map[uint8]string{
	ServiceAccessPointUDT:                    "UDT",
	ServiceAccessPointTCPIPHeaderCompression: "TCP/IP header compression",
	ServiceAccessPointUDPIPHeaderCompression: "UDP/IP header compression",
	ServiceAccessPointIPBasedPacketData:      "IP based packet data",
	ServiceAccessPointARP:                    "ARP",
	ServiceAccessPointProprietaryData:        "proprietary data",
	ServiceAccessPointShortData:              "short data",
}

// Response Data Header Response Type, encodes class and type
const (
	_                            uint8 = iota // Class 0b00, Type 0b000
	ResponseTypeACK                           // Class 0b00, Type 0b001
	_                                         // Class 0b00, Type 0b010
	_                                         // Class 0b00, Type 0b011
	_                                         // Class 0b00, Type 0b100
	_                                         // Class 0b00, Type 0b101
	_                                         // Class 0b00, Type 0b110
	_                                         // Class 0b00, Type 0b111
	ResponseTypeIllegalFormat                 // Class 0b01, Type 0b000
	ResponseTypePacketCRCFailed               // Class 0b01, Type 0b001
	ResponseTypeMemoryFull                    // Class 0b01, Type 0b010
	ResponseTypeRecvFSVNOutOfSeq              // Class 0b01, Type 0b011
	ResponseTypeUndeliverable                 // Class 0b01, Type 0b100
	ResponseTypeRecvPktOutOfSeq               // Class 0b01, Type 0b101
	ResponseTypeDisallowed                    // Class 0b01, Type 0b110
	_                                         // Class 0b01, Type 0b111
	ResponseTypeSelectiveACK                  // Class 0b10, Type 0b000
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

// http://www.etsi.org/images/files/DMRcodes/dmrs-mfid.xls
var ManufacturerName = map[uint8]string{
	0x00: "Reserved",
	0x01: "Reserved",
	0x02: "Reserved",
	0x03: "Reserved",
	0x04: "Flyde Micro Ltd.",
	0x05: "PROD-EL SPA",
	0x06: "Trident Datacom DBA Trident Micro Systems",
	0x07: "RADIODATA GmbH",
	0x08: "HYT science tech",
	0x09: "ASELSAN Elektronik Sanayi ve Ticaret A.S.",
	0x0a: "Kirisun Communications Co. Ltd",
	0x0b: "DMR Association Ltd.",
	0x10: "Motorola Ltd.",
	0x13: "EMC S.p.A. (Electronic Marketing Company)",
	0x1c: "EMC S.p.A. (Electronic Marketing Company)",
	0x20: "JVCKENWOOD Corporation",
	0x33: "Radio Activity Srl",
	0x3c: "Radio Activity Srl",
	0x58: "Tait Electronics Ltd",
	0x68: "HYT science tech",
	0x77: "Vertex Standard",
}

type DataHeaderData interface {
	String() string
	Write([]byte) error
}

type DataHeader struct {
	PacketFormat       uint8
	DstIsGroup         bool
	ResponseRequested  bool
	HeaderCompression  bool
	ServiceAccessPoint uint8
	DstID              uint32
	SrcID              uint32
	CRC                uint16
	Data               DataHeaderData
}

func (h *DataHeader) Bytes() ([]byte, error) {
	var data = make([]byte, 12)

	data[0] = (h.PacketFormat & B00001111)
	if h.DstIsGroup {
		data[0] |= B10000000
	}
	if h.ResponseRequested {
		data[0] |= B01000000
	}
	if h.HeaderCompression {
		data[0] |= B00100000
	}
	data[1] = (h.ServiceAccessPoint << 4) & B11110000
	data[2] = uint8(h.DstID >> 16)
	data[3] = uint8(h.DstID >> 8)
	data[4] = uint8(h.DstID)
	data[5] = uint8(h.SrcID >> 16)
	data[6] = uint8(h.SrcID >> 8)
	data[7] = uint8(h.SrcID)

	if h.Data != nil {
		if err := h.Data.Write(data); err != nil {
			return nil, err
		}
	}

	h.CRC = 0
	for i := 0; i < 10; i++ {
		crc16(&h.CRC, data[i])
	}
	crc16end(&h.CRC)

	// Inverting according to the inversion polynomial.
	h.CRC = ^h.CRC
	// Applying CRC mask, see DMR AI spec. page 143.
	h.CRC ^= 0xcccc

	data[10] = uint8(h.CRC >> 8)
	data[11] = uint8(h.CRC)

	return data, nil
}

func (h DataHeader) String() string {
	var part = []string{"data header"}
	if h.DstIsGroup {
		part = append(part, "group")
	} else {
		part = append(part, "unit")
	}
	part = append(part, fmt.Sprintf("response %t, sap %s (%d), %d->%d",
		h.ResponseRequested, ServiceAccessPointName[h.ServiceAccessPoint], h.ServiceAccessPoint,
		h.SrcID, h.DstID))
	if h.Data != nil {
		part = append(part, h.Data.String())
	}
	return strings.Join(part, ", ")
}

type UDTData struct {
	Format            uint8
	PadNibble         uint8
	AppendedBlocks    uint8
	SupplementaryFlag bool
	Opcode            uint8
}

func (d UDTData) String() string {
	return fmt.Sprintf("UDT, format %s (%d), pad nibble %d, appended blocks %d, supplementary %t, opcode %d",
		UDTFormatName[d.Format], d.Format, d.PadNibble, d.AppendedBlocks, d.SupplementaryFlag, d.Opcode)
}

func (d UDTData) Write(data []byte) error {
	data[1] |= (d.Format & B00001111)
	data[8] = (d.AppendedBlocks & B00000011) | (d.PadNibble << 3)
	data[9] = (d.Opcode & B00111111)
	if d.SupplementaryFlag {
		data[9] |= B10000000
	}
	return nil
}

type UnconfirmedData struct {
	PadOctetCount          uint8
	FullMessage            bool
	BlocksToFollow         uint8
	FragmentSequenceNumber uint8
}

func (d UnconfirmedData) String() string {
	return fmt.Sprintf("unconfirmed, pad octet %d, full %t, blocks %d, sequence %d",
		d.PadOctetCount, d.FullMessage, d.BlocksToFollow, d.FragmentSequenceNumber)
}

func (d UnconfirmedData) Write(data []byte) error {
	data[0] |= d.PadOctetCount & B00010000
	data[1] |= d.PadOctetCount & B00001111
	data[8] = d.BlocksToFollow & B01111111
	if d.FullMessage {
		data[8] |= B10000000
	}
	data[9] = d.FragmentSequenceNumber & B00001111
	return nil
}

type ConfirmedData struct {
	PadOctetCount          uint8
	FullMessage            bool
	BlocksToFollow         uint8
	Resync                 bool
	SendSequenceNumber     uint8
	FragmentSequenceNumber uint8
}

func (d ConfirmedData) String() string {
	return fmt.Sprintf("confirmed, pad octet %d, full %t, blocks %d, resync %t, send sequence %d, sequence %d",
		d.PadOctetCount, d.FullMessage, d.BlocksToFollow, d.Resync, d.SendSequenceNumber, d.FragmentSequenceNumber)
}

func (d ConfirmedData) Write(data []byte) error {
	data[0] |= d.PadOctetCount & B00010000
	data[1] |= d.PadOctetCount & B00001111
	data[8] = d.BlocksToFollow & B01111111
	if d.FullMessage {
		data[8] |= B10000000
	}
	data[9] = (d.FragmentSequenceNumber&B00000111)<<0 | (d.SendSequenceNumber&B00000111)<<4
	if d.Resync {
		data[9] |= B10000000
	}
	return nil
}

type ResponseData struct {
	BlocksToFollow uint8
	ClassType      uint8 // See ResponseType map above
	Status         uint8
}

func (d ResponseData) String() string {
	return fmt.Sprintf("response, blocks %d, type %s (%02b %03b), status %d",
		d.BlocksToFollow, ResponseTypeName[d.ClassType], (d.ClassType >> 3), (d.ClassType & 0x07), d.Status)
}

func (d ResponseData) Write(data []byte) error {
	data[8] = d.BlocksToFollow & B01111111
	data[9] = d.Status | d.ClassType<<3
	return nil
}

type ProprietaryData struct {
	ManufacturerID uint8
}

func (d ProprietaryData) String() string {
	return fmt.Sprintf("proprietary, manufacturer %s (%d)",
		ManufacturerName[d.ManufacturerID], d.ManufacturerID)
}

func (d ProprietaryData) Write(data []byte) error {
	data[1] = (d.ManufacturerID & B01111111)
	return nil
}

type ShortDataRawData struct {
	AppendedBlocks uint8
	SrcPort        uint8
	DstPort        uint8
	Resync         bool
	FullMessage    bool
	BitPadding     uint8
}

func (d ShortDataRawData) String() string {
	return fmt.Sprintf("short data raw, blocks %d, src port %d, dst port %d, rsync %t, full %t, padding %d",
		d.AppendedBlocks, d.SrcPort, d.DstPort, d.Resync, d.FullMessage, d.BitPadding)
}

func (d ShortDataRawData) Write(data []byte) error {
	data[0] |= d.AppendedBlocks & B00110000
	data[1] |= d.AppendedBlocks & B00001111
	data[8] = (d.SrcPort&B00000111)<<5 | (d.DstPort&B00000111)<<2
	if d.Resync {
		data[8] |= B00000010
	}
	if d.FullMessage {
		data[8] |= B00000001
	}
	data[9] = d.BitPadding
	return nil
}

type ShortDataDefinedData struct {
	AppendedBlocks uint8
	DDFormat       uint8
	Resync         bool
	FullMessage    bool
	BitPadding     uint8
}

func (d ShortDataDefinedData) String() string {
	return fmt.Sprintf("short data defined, blocks %d, dd format %s (%d), resync %t, full %t, padding %d",
		d.AppendedBlocks, DDFormatName[d.DDFormat], d.DDFormat, d.Resync, d.FullMessage, d.BitPadding)
}

func (d ShortDataDefinedData) Write(data []byte) error {
	data[0] |= d.AppendedBlocks & B00110000
	data[1] |= d.AppendedBlocks & B00001111
	data[8] = (d.DDFormat & B00111111) << 2
	if d.Resync {
		data[8] |= B00000010
	}
	if d.FullMessage {
		data[8] |= B00000001
	}
	data[9] = d.BitPadding
	return nil
}

var _ (DataHeaderData) = (*ShortDataDefinedData)(nil)

func ParseDataHeader(data []byte, proprietary bool) (*DataHeader, error) {
	if len(data) != 12 {
		return nil, fmt.Errorf("data must be 12 bytes, got %d", len(data))
	}
	var (
		ccrc = (uint16(data[10]) << 8) | uint16(data[11])
		hcrc = dataHeaderCRC(data)
	)
	if ccrc != hcrc {
		return nil, fmt.Errorf("data CRC mismatch, %#04x != %#04x", ccrc, hcrc)
	}

	h := &DataHeader{
		DstIsGroup:         (data[0] & B10000000) > 0,
		ResponseRequested:  (data[0] & B01000000) > 0,
		HeaderCompression:  (data[0] & B00100000) > 0,
		PacketFormat:       (data[0] & B00001111),
		ServiceAccessPoint: (data[1] & B11110000) >> 4,
		DstID:              uint32(data[2])<<16 | uint32(data[3])<<8 | uint32(data[4]),
		SrcID:              uint32(data[5])<<16 | uint32(data[6])<<8 | uint32(data[7]),
		CRC:                ccrc,
	}

	if proprietary {
		h.Data = ProprietaryData{
			ManufacturerID: data[1] & B01111111,
		}

	} else {
		switch h.PacketFormat {
		case PacketFormatUDT:
			h.Data = &UDTData{
				Format:            (data[1] & B00001111),
				PadNibble:         (data[8] & B11111000) >> 3,
				AppendedBlocks:    (data[8] & B00000011),
				SupplementaryFlag: (data[9] & B10000000) > 0,
				Opcode:            (data[9] & B00111111),
			}
			break

		case PacketFormatResponse:
			h.Data = &ResponseData{
				BlocksToFollow: (data[8] & B01111111),
				ClassType:      (data[9] & B11111000) >> 3,
				Status:         (data[9] & B00000111),
			}
			break

		case PacketFormatUnconfirmedData:
			h.Data = &UnconfirmedData{
				PadOctetCount:          (data[0] & B00010000) | (data[1] & B00001111),
				FullMessage:            (data[8] & B10000000) > 0,
				BlocksToFollow:         (data[8] & B01111111),
				FragmentSequenceNumber: (data[9] & B00001111),
			}
			break

		case PacketFormatConfirmedData:
			h.Data = &ConfirmedData{
				PadOctetCount:          (data[0] & B00010000) | (data[1] & B00001111),
				FullMessage:            (data[8] & B10000000) > 0,
				BlocksToFollow:         (data[8] & B01111111),
				Resync:                 (data[9] & B10000000) > 0,
				SendSequenceNumber:     (data[9] & B01110000) >> 4,
				FragmentSequenceNumber: (data[9] & B00001111),
			}
			break

		case PacketFormatShortDataRaw:
			h.Data = &ShortDataRawData{
				AppendedBlocks: (data[0] & B00110000) | (data[1] & B00001111),
				SrcPort:        (data[8] & B11100000) >> 5,
				DstPort:        (data[8] & B00011100) >> 2,
				Resync:         (data[8] & B00000010) > 0,
				FullMessage:    (data[8] & B00000001) > 0,
				BitPadding:     (data[9]),
			}
			break

		case PacketFormatShortDataDefined:
			h.Data = &ShortDataDefinedData{
				AppendedBlocks: (data[0] & B00110000) | (data[1] & B00001111),
				DDFormat:       (data[8] & B11111100) >> 2,
				Resync:         (data[8] & B00000010) > 0,
				FullMessage:    (data[8] & B00000001) > 0,
				BitPadding:     (data[9]),
			}
			break

		default:
			return nil, fmt.Errorf("dmr: unknown data data packet format %#02x (%d)", h.PacketFormat, h.PacketFormat)
		}
	}

	return h, nil
}

func dataHeaderCRC(data []byte) uint16 {
	var crc uint16
	if len(data) < 10 {
		return crc
	}

	for i := 0; i < 10; i++ {
		crc16(&crc, data[i])
	}
	crc16end(&crc)

	return (^crc) ^ 0xcccc
}
