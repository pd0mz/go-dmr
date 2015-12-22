package dmr

import (
	"errors"
	"fmt"
	"strings"

	"github.com/pd0mz/go-dmr/crc/quadres_16_7"
	"github.com/pd0mz/go-dmr/fec"
)

// Priority Levels
const (
	NoPriority uint8 = iota
	Priority1
	Priority2
	Priority3
)

// PriorityName is a map of priority level to string.
var PriorityName = map[uint8]string{
	NoPriority: "no priority",
	Priority1:  "priority 1",
	Priority2:  "priority 2",
	Priority3:  "priority 3",
}

// ServiceOptions as per DMR part 2, section 7.2.1.
type ServiceOptions struct {
	// Emergency service
	Emergency bool
	// Not defined in document
	Privacy bool
	// Broadcast service (only defined in group calls)
	Broadcast bool
	// Open Voice Call Mode
	OpenVoiceCallMode bool
	// Priority 3 (0b11) is the highest priority
	Priority uint8
}

// Byte packs the service options to a single byte.
func (so *ServiceOptions) Byte() byte {
	var b byte
	if so.Emergency {
		b |= B00000001
	}
	if so.Privacy {
		b |= B00000010
	}
	if so.Broadcast {
		b |= B00010000
	}
	if so.OpenVoiceCallMode {
		b |= B00100000
	}
	b |= (so.Priority << 6)
	return b
}

// String representatation of the service options.
func (so *ServiceOptions) String() string {
	var part = []string{}
	if so.Emergency {
		part = append(part, "emergency")
	}
	if so.Privacy {
		part = append(part, "privacy")
	}
	if so.Broadcast {
		part = append(part, "broadcast")
	}
	if so.OpenVoiceCallMode {
		part = append(part, "Open Voice Call Mode")
	}
	part = append(part, fmt.Sprintf("%s (%d)", PriorityName[so.Priority], so.Priority))
	return strings.Join(part, ", ")
}

// ParseServiceOptions parses the service options byte.
func ParseServiceOptions(data byte) ServiceOptions {
	return ServiceOptions{
		Emergency:         (data & B00000001) > 0,
		Privacy:           (data & B00000010) > 0,
		Broadcast:         (data & B00010000) > 0,
		OpenVoiceCallMode: (data & B00100000) > 0,
		Priority:          (data & B11000000) >> 6,
	}
}

// Full Link Control Opcode
const (
	GroupVoiceChannelUser      uint8 = 0x00 // B000000
	UnitToUnitVoiceChannelUser uint8 = 0x03 // B000011
)

// LC is a Link Control message.
type LC struct {
	CallType       uint8
	Opcode         uint8
	FeatureSetID   uint8
	ServiceOptions ServiceOptions
	DstID          uint32
	SrcID          uint32
}

// Bytes packs the Link Control message to bytes.
func (lc *LC) Bytes() []byte {
	var fclo uint8
	switch lc.CallType {
	case CallTypeGroup:
		fclo = GroupVoiceChannelUser
		break
	case CallTypePrivate:
		fclo = UnitToUnitVoiceChannelUser
		break
	}

	return []byte{
		fclo,
		lc.FeatureSetID,
		lc.ServiceOptions.Byte(),
		uint8(lc.DstID >> 16),
		uint8(lc.DstID >> 8),
		uint8(lc.DstID),
		uint8(lc.SrcID >> 16),
		uint8(lc.SrcID >> 8),
		uint8(lc.SrcID),
	}
}

func (lc *LC) String() string {
	return fmt.Sprintf("call type %s, feature set id %d, %d->%d, service options %s",
		CallTypeName[lc.CallType], lc.FeatureSetID, lc.SrcID, lc.DstID, lc.ServiceOptions.String())
}

// ParseLC parses a packed Link Control message.
func ParseLC(data []byte) (*LC, error) {
	if data == nil {
		return nil, errors.New("dmr/lc: data can't be nil")
	}
	if len(data) != 9 {
		return nil, fmt.Errorf("dmr/lc: expected 9 LC bytes, got %d", len(data))
	}

	if data[0]&B10000000 > 0 {
		return nil, errors.New("dmr/lc: protect flag is not 0")
	}

	var (
		ct   uint8
		fclo = data[0] & B00111111
	)
	switch fclo {
	case GroupVoiceChannelUser:
		ct = CallTypeGroup
		break
	case UnitToUnitVoiceChannelUser:
		ct = CallTypePrivate
		break
	default:
		return nil, fmt.Errorf("dmr/lc: unknown FCLO %06b (%d)", fclo, fclo)
	}

	return &LC{
		CallType:       ct,
		FeatureSetID:   data[1],
		ServiceOptions: ParseServiceOptions(data[2]),
		DstID:          uint32(data[3])<<16 | uint32(data[4])<<8 | uint32(data[5]),
		SrcID:          uint32(data[6])<<16 | uint32(data[7])<<8 | uint32(data[8]),
	}, nil
}

// ParseFullLC parses a packed Link Control message and checks/corrects the Reed-Solomon check data.
func ParseFullLC(data []byte) (*LC, error) {
	if data == nil {
		return nil, errors.New("dmr/full lc: data can't be nil")
	}
	if len(data) != 12 {
		return nil, fmt.Errorf("dmr/full lc: expected 12 bytes, got %d", len(data))
	}

	syndrome := &fec.RS_12_9_Poly{}
	if err := fec.RS_12_9_CalcSyndrome(data, syndrome); err != nil {
		return nil, err
	}
	if !fec.RS_12_9_CheckSyndrome(syndrome) {
		if _, err := fec.RS_12_9_Correct(data, syndrome); err != nil {
			return nil, err
		}
	}

	return ParseLC(data[:9])
}

// EMB LCSS fragments.
const (
	SingleFragment uint8 = iota
	FirstFragment
	LastFragment
	Continuation
)

// LCSSName is a map of LCSS fragment type to string.
var LCSSName = map[uint8]string{
	SingleFragment: "single fragment",
	FirstFragment:  "first fragment",
	LastFragment:   "last fragment",
	Continuation:   "continuation",
}

// EMB contains embedded signalling.
type EMB struct {
	ColorCode uint8
	LCSS      uint8
}

func (emb *EMB) String() string {
	return fmt.Sprintf("color code %d, %s (%d)", emb.ColorCode, LCSSName[emb.LCSS], emb.LCSS)
}

// ParseEMB parses embedded signalling
func ParseEMB(bits []byte) (*EMB, error) {
	if len(bits) != EMBBits {
		return nil, fmt.Errorf("dmr/emb: expected %d bits, got %d", EMBBits, len(bits))
	}

	if !quadres_16_7.Check(bits) {
		return nil, errors.New("dmr/emb: checksum error")
	}

	if bits[4] != 0 {
		return nil, errors.New("dmr/emb: pi is not 0")
	}

	return &EMB{
		ColorCode: uint8(bits[0])<<3 | uint8(bits[1])<<2 | uint8(bits[2])<<1 | uint8(bits[3]),
		LCSS:      uint8(bits[5])<<1 | uint8(bits[6]),
	}, nil
}

// ParseEMBBitsFromSync extracts the embedded signalling bits from the SYNC bits.
func ParseEMBBitsFromSync(sync []byte) ([]byte, error) {
	if sync == nil {
		return nil, errors.New("dmr/emb from sync: bits can't be nil")
	}
	if len(sync) != 48 {
		return nil, fmt.Errorf("dmr/emb from sync: expected 48 sync bits, got %d", len(sync))
	}

	var bits = make([]byte, 16)
	copy(bits[:8], sync[:8])
	copy(bits[8:], sync[8+32:])
	return bits, nil
}

// ParseEmbeddedSignallingLCFromSyncBits extracts the embedded signalling LC from the SYNC bits.
func ParseEmbeddedSignallingLCFromSyncBits(sync []byte) ([]byte, error) {
	if sync == nil {
		return nil, errors.New("dmr/emb lc from sync: bits can't be nil")
	}
	if len(sync) != 48 {
		return nil, fmt.Errorf("dmr/emb lc from sync: expected 48 sync bits, got %d", len(sync))
	}

	var bits = make([]byte, 32)
	copy(bits, sync[8:40])
	return bits, nil
}

// EmbeddedSignallingLC contains the embedded signalling LC and checksum.
type EmbeddedSignallingLC struct {
	Bits     []byte
	Checksum []byte
}

// Check verifies the checksum in the embedded signalling LC.
func (eslc *EmbeddedSignallingLC) Check() bool {
	var checksum uint8
	checksum |= eslc.Checksum[0] << 4
	checksum |= eslc.Checksum[1] << 3
	checksum |= eslc.Checksum[2] << 2
	checksum |= eslc.Checksum[3] << 1
	checksum |= eslc.Checksum[4] << 0

	var data = BitsToBytes(eslc.Bits)
	var verify uint16
	for _, b := range data {
		verify += uint16(b)
	}

	var calculated = uint8(verify % 31)
	return calculated == checksum
}

// Interleave packs the embedded signalling LC to interleaved bits.
func (eslc *EmbeddedSignallingLC) Interleave() []byte {
	var bits = make([]byte, 77)
	var j int
	for i := range bits {
		switch i {
		case 32:
			bits[i] = eslc.Checksum[0]
			break
		case 43:
			bits[i] = eslc.Checksum[1]
			break
		case 54:
			bits[i] = eslc.Checksum[2]
			break
		case 65:
			bits[i] = eslc.Checksum[3]
			break
		case 76:
			bits[i] = eslc.Checksum[4]
			break
		default:
			bits[i] = eslc.Bits[j]
			j++
		}
	}

	return bits
}

// DeinterleaveEmbeddedSignallingLC deinterleaves the embedded signalling LC bits.
func DeinterleaveEmbeddedSignallingLC(bits []byte) (*EmbeddedSignallingLC, error) {
	if bits == nil {
		return nil, errors.New("dmr/emb lc deinterleave: bits can't be nil")
	}
	if len(bits) != 77 {
		return nil, fmt.Errorf("dmr/emb lc deinterleave: expected 77 bits, got %d", len(bits))
	}

	var eslc = &EmbeddedSignallingLC{
		Bits:     make([]byte, 72),
		Checksum: make([]byte, 5),
	}
	var j int
	for i, b := range bits {
		switch i {
		case 32:
			eslc.Checksum[0] = b
			break
		case 43:
			eslc.Checksum[1] = b
			break
		case 54:
			eslc.Checksum[2] = b
			break
		case 65:
			eslc.Checksum[3] = b
			break
		case 76:
			eslc.Checksum[4] = b
			break
		default:
			eslc.Bits[j] = b
			j++
		}
	}

	return eslc, nil
}
