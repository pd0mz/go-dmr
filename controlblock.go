package dmr

import (
	"errors"
	"fmt"
	"strings"
)

// Control Block Opcode
const (
	OutboundActivationOpcode                   = B00111000
	UnitToUnitVoiceServiceRequestOpcode        = B00000100
	UnitToUnitVoiceServiceAnswerResponseOpcode = B00000101
	NegativeAcknowledgeResponseOpcode          = B00100100
	PreambleOpcode                             = B00111101
)

type ControlBlock struct {
	CRC          uint16
	Last         bool
	Opcode       uint8
	SrcID, DstID uint32
	Data         ControlBlockData
}

func (cb *ControlBlock) String() string {
	if cb.Data == nil {
		return fmt.Sprintf("CSBK, last %t, %d->%d, unknown (opcode %d)",
			cb.Last, cb.SrcID, cb.DstID, cb.Opcode)
	}
	return fmt.Sprintf("CSBK, last %t, %d->%d, %s (opcode %d)",
		cb.Last, cb.SrcID, cb.DstID, cb.Data.String(), cb.Opcode)
}

type ControlBlockData interface {
	String() string
	Write([]byte) error
	Parse([]byte) error
}

type OutboundActivation struct{}

func (d *OutboundActivation) String() string { return "outbound activation" }

func (d *OutboundActivation) Parse(data []byte) error {
	if len(data) != InfoSize {
		return fmt.Errorf("dmr: expected %d info bytes, got %d", InfoSize, len(data))
	}
	return nil
}

func (d *OutboundActivation) Write(data []byte) error {
	if len(data) != InfoSize {
		return fmt.Errorf("dmr: expected %d info bytes, got %d", InfoSize, len(data))
	}
	data[0] |= OutboundActivationOpcode
	return nil
}

type UnitToUnitVoiceServiceRequest struct {
	Options uint8
}

func (d *UnitToUnitVoiceServiceRequest) String() string {
	return fmt.Sprintf("unit to unit voice service request, options %d", d.Options)
}

func (d *UnitToUnitVoiceServiceRequest) Parse(data []byte) error {
	if len(data) != InfoSize {
		return fmt.Errorf("dmr: expected %d info bytes, got %d", InfoSize, len(data))
	}
	d.Options = data[2]
	return nil
}

func (d *UnitToUnitVoiceServiceRequest) Write(data []byte) error {
	if len(data) != InfoSize {
		return fmt.Errorf("dmr: expected %d info bytes, got %d", InfoSize, len(data))
	}
	data[0] |= UnitToUnitVoiceServiceRequestOpcode
	data[2] = d.Options
	return nil
}

var _ (ControlBlockData) = (*UnitToUnitVoiceServiceRequest)(nil)

type UnitToUnitVoiceServiceAnswerResponse struct {
	Options  uint8
	Response uint8
}

func (d *UnitToUnitVoiceServiceAnswerResponse) String() string {
	return fmt.Sprintf("unit to unit voice service answer response, options %d, response %d", d.Options, d.Response)
}

func (d *UnitToUnitVoiceServiceAnswerResponse) Parse(data []byte) error {
	if len(data) != InfoSize {
		return fmt.Errorf("dmr: expected %d info bytes, got %d", InfoSize, len(data))
	}
	d.Options = data[2]
	d.Response = data[3]
	return nil
}

func (d *UnitToUnitVoiceServiceAnswerResponse) Write(data []byte) error {
	if len(data) != InfoSize {
		return fmt.Errorf("dmr: expected %d info bytes, got %d", InfoSize, len(data))
	}
	data[0] |= UnitToUnitVoiceServiceAnswerResponseOpcode
	data[2] = d.Options
	data[3] = d.Response
	return nil
}

var _ (ControlBlockData) = (*UnitToUnitVoiceServiceAnswerResponse)(nil)

type NegativeAcknowledgeResponse struct {
	SourceType  bool
	ServiceType uint8
	Reason      uint8
}

func (d *NegativeAcknowledgeResponse) String() string {
	return fmt.Sprintf("negative ACK response, source %t, service %d, reason %d", d.SourceType, d.ServiceType, d.Reason)
}

func (d *NegativeAcknowledgeResponse) Parse(data []byte) error {
	if len(data) != InfoSize {
		return fmt.Errorf("dmr: expected %d info bytes, got %d", InfoSize, len(data))
	}
	d.SourceType = (data[2] & B01000000) > 0
	d.ServiceType = (data[2] & B00011111)
	d.Reason = data[3]
	return nil
}

func (d *NegativeAcknowledgeResponse) Write(data []byte) error {
	if len(data) != InfoSize {
		return fmt.Errorf("dmr: expected %d info bytes, got %d", InfoSize, len(data))
	}
	data[0] |= NegativeAcknowledgeResponseOpcode
	data[2] = d.ServiceType
	if d.SourceType {
		data[2] |= B01000000
	}
	data[3] = d.Reason
	return nil
}

var _ (ControlBlockData) = (*NegativeAcknowledgeResponse)(nil)

type Preamble struct {
	DataFollows bool
	DstIsGroup  bool
	Blocks      uint8
}

func (d *Preamble) String() string {
	var part = []string{"preamble"}
	if d.DataFollows {
		part = append(part, "data folllows")
	}
	if d.DstIsGroup {
		part = append(part, "group")
	} else {
		part = append(part, "unit")
	}
	part = append(part, fmt.Sprintf("%d blocks", d.Blocks))
	return strings.Join(part, ", ")
}

func (d *Preamble) Parse(data []byte) error {
	if len(data) != InfoSize {
		return fmt.Errorf("dmr: expected %d info bytes, got %d", InfoSize, len(data))
	}
	d.DataFollows = (data[2] & B10000000) > 0
	d.DstIsGroup = (data[2] & B01000000) > 0
	d.Blocks = data[3]
	return nil
}

func (d *Preamble) Write(data []byte) error {
	if len(data) != InfoSize {
		return fmt.Errorf("dmr: expected %d info bytes, got %d", InfoSize, len(data))
	}
	data[0] |= PreambleOpcode
	if d.DataFollows {
		data[2] |= B10000000
	}
	if d.DstIsGroup {
		data[2] |= B01000000
	}
	data[3] = d.Blocks
	return nil
}

var _ (ControlBlockData) = (*Preamble)(nil)

func (cb *ControlBlock) Bytes() ([]byte, error) {
	var data = make([]byte, InfoSize)

	if err := cb.Data.Write(data); err != nil {
		return nil, err
	}
	if cb.Last {
		data[0] |= B10000000
	}

	data[4] = uint8(cb.DstID >> 16)
	data[5] = uint8(cb.DstID >> 8)
	data[6] = uint8(cb.DstID)
	data[7] = uint8(cb.SrcID >> 16)
	data[8] = uint8(cb.SrcID >> 8)
	data[9] = uint8(cb.SrcID)

	// Calculate CRC16
	for i := 0; i < 10; i++ {
		crc16(&cb.CRC, data[i])
	}
	crc16end(&cb.CRC)

	// Inverting according to the inversion polynomial.
	cb.CRC = ^cb.CRC
	// Applying CRC mask, see DMR AI spec. page 143.
	cb.CRC ^= 0xa5a5

	data[10] = uint8(cb.CRC >> 8)
	data[11] = uint8(cb.CRC)

	return data, nil
}

func ParseControlBlock(data []byte) (*ControlBlock, error) {
	if len(data) != InfoSize {
		return nil, fmt.Errorf("dmr: expected %d info bytes, got %d", InfoSize, len(data))
	}

	// Calculate CRC16
	var crc uint16
	for i := 0; i < 10; i++ {
		crc16(&crc, data[i])
	}
	crc16end(&crc)

	// Inverting according to the inversion polynomial
	crc = ^crc
	// Applying CRC mask, see DMR AI spec. page 143.
	crc ^= 0xa5a5

	// Check packet
	if data[0]&B01000000 > 0 {
		return nil, errors.New("dmr: CSBK protect flag is set")
	}
	if data[1] != 0 {
		return nil, errors.New("dmr: CSBK feature set ID is set")
	}

	cb := &ControlBlock{
		CRC:    uint16(data[10])<<8 | uint16(data[11]),
		Last:   (data[0] & B10000000) > 0,
		Opcode: (data[0] & B00111111),
		DstID:  uint32(data[4])<<16 | uint32(data[5])<<8 | uint32(data[6]),
		SrcID:  uint32(data[7])<<16 | uint32(data[8])<<8 | uint32(data[9]),
	}

	if crc != cb.CRC {
		return nil, fmt.Errorf("dmr: control block CRC error (%#04x != %#04x)", crc, cb.CRC)
	}

	switch cb.Opcode {
	case OutboundActivationOpcode:
		cb.Data = &OutboundActivation{}
		break
	case UnitToUnitVoiceServiceRequestOpcode:
		cb.Data = &UnitToUnitVoiceServiceRequest{}
		break
	case UnitToUnitVoiceServiceAnswerResponseOpcode:
		cb.Data = &UnitToUnitVoiceServiceAnswerResponse{}
		break
	case NegativeAcknowledgeResponseOpcode:
		cb.Data = &NegativeAcknowledgeResponse{}
		break
	case PreambleOpcode:
		cb.Data = &Preamble{}
		break
	default:
		return nil, fmt.Errorf("dmr: unknown CSBK opcode %#02x (%#06b)", cb.Opcode, cb.Opcode)
	}

	if err := cb.Data.Parse(data); err != nil {
		return nil, err
	}

	return cb, nil
}
