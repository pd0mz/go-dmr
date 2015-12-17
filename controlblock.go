package dmr

import (
	"errors"
	"fmt"
)

// Control Block Options
const (
	CBSKOOutboundActivation                   = B00111000
	CBSKOUnitToUnitVoiceServiceRequest        = B00000100
	CBSKOUnitToUnitVoiceServiceAnswerResponse = B00000101
	CBSKONegativeAcknowledgeResponse          = B00100100
	CBSKOPreamble                             = B00111101
)

type ControlBlock struct {
	Last         bool
	CBSKO        uint8
	SrcID, DstID uint32
	Data         ControlBlockData
}

type ControlBlockData interface {
	Write([]byte) error
	Parse([]byte) error
}

type OutboundActivation struct{}

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
	data[0] |= B00111000
	return nil
}

type UnitToUnitVoiceServiceRequest struct {
	Options uint8
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
	data[0] |= B00000100
	data[2] = d.Options
	return nil
}

var _ (ControlBlockData) = (*UnitToUnitVoiceServiceRequest)(nil)

type UnitToUnitVoiceServiceAnswerResponse struct {
	Options  uint8
	Response uint8
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
	data[0] |= B00000101
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
	data[0] |= B00100110
	data[2] = d.ServiceType
	if d.SourceType {
		data[2] |= B01000000
	}
	data[3] = d.Reason
	return nil
}

var _ (ControlBlockData) = (*NegativeAcknowledgeResponse)(nil)

type ControlBlockPreamble struct {
	DataFollows bool
	DstIsGroup  bool
	Blocks      uint8
}

func (d *ControlBlockPreamble) Parse(data []byte) error {
	if len(data) != InfoSize {
		return fmt.Errorf("dmr: expected %d info bytes, got %d", InfoSize, len(data))
	}
	d.DataFollows = (data[2] & B10000000) > 0
	d.DstIsGroup = (data[2] & B01000000) > 0
	d.Blocks = data[3]
	return nil
}

func (d *ControlBlockPreamble) Write(data []byte) error {
	if len(data) != InfoSize {
		return fmt.Errorf("dmr: expected %d info bytes, got %d", InfoSize, len(data))
	}
	data[0] |= B00100110
	if d.DataFollows {
		data[2] |= B10000000
	}
	if d.DstIsGroup {
		data[2] |= B01000000
	}
	data[3] = d.Blocks
	return nil
}

var _ (ControlBlockData) = (*ControlBlockPreamble)(nil)

func ParseControlBlock(data []byte) (*ControlBlock, error) {
	if len(data) != InfoSize {
		return nil, fmt.Errorf("dmr: expected %d info bytes, got %d", InfoSize, len(data))
	}

	// Calculate CRC16

	// Check packet
	if data[0]&B01000000 > 0 {
		return nil, errors.New("dmr: CBSK protect flag is set")
	}
	if data[1] != 0 {
		return nil, errors.New("dmr: CBSK feature set ID is set")
	}

	cb := &ControlBlock{
		Last:  (data[0] & B10000000) > 0,
		CBSKO: (data[0] & B00111111),
		DstID: uint32(data[4])<<16 | uint32(data[5])<<8 | uint32(data[6]),
		SrcID: uint32(data[7])<<16 | uint32(data[8])<<8 | uint32(data[9]),
	}

	switch cb.CBSKO {
	case CBSKOOutboundActivation:
		cb.Data = &OutboundActivation{}
		break
	case CBSKOUnitToUnitVoiceServiceRequest:
		cb.Data = &UnitToUnitVoiceServiceRequest{}
		break
	case CBSKOUnitToUnitVoiceServiceAnswerResponse:
		cb.Data = &UnitToUnitVoiceServiceAnswerResponse{}
		break
	case CBSKONegativeAcknowledgeResponse:
		cb.Data = &NegativeAcknowledgeResponse{}
		break
	case CBSKOPreamble:
		cb.Data = &ControlBlockPreamble{}
		break
	default:
		return nil, fmt.Errorf("dmr: unknown CBSKO %#02x (%#06b)", cb.CBSKO, cb.CBSKO)
	}

	if err := cb.Data.Parse(data); err != nil {
		return nil, err
	}

	return cb, nil
}
