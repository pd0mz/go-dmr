package lc

import (
	"errors"
	"fmt"

	"github.com/pd0mz/go-dmr"
	"github.com/pd0mz/go-dmr/fec"
)

// Full Link Control Opcode
const (
	GroupVoiceChannelUser      uint8 = 0x00 // B000000
	UnitToUnitVoiceChannelUser uint8 = 0x03 // B000011
	TalkerAliasHeader          uint8 = 0x04 // B000100
	TalkerAliasBlk1            uint8 = 0x05 // B000101
	TalkerAliasBlk2            uint8 = 0x06 // B000110
	TalkerAliasBlk3            uint8 = 0x07 // B000111
	GpsInfo                    uint8 = 0x08 // B001000
)

// LC is a Link Control message.
type LC struct {
	CallType          uint8
	Opcode            uint8
	FeatureSetID      uint8
	VoiceChannelUser  *VoiceChannelUserPDU
	GpsInfo           *GpsInfoPDU
	TalkerAliasHeader *TalkerAliasHeaderPDU
	TalkerAliasBlocks [3]*TalkerAliasBlockPDU
}

// Bytes packs the Link Control message to bytes.
func (lc *LC) Bytes() []byte {
	var (
		lcHeader = []byte{
			lc.Opcode,
			lc.FeatureSetID,
		}
		innerPdu []byte
	)

	switch lc.Opcode {
	case GroupVoiceChannelUser:
		fallthrough
	case UnitToUnitVoiceChannelUser:
		innerPdu = lc.VoiceChannelUser.Bytes()
	case TalkerAliasHeader:
		innerPdu = lc.TalkerAliasHeader.Bytes()
	case TalkerAliasBlk1:
		innerPdu = lc.TalkerAliasBlocks[0].Bytes()
	case TalkerAliasBlk2:
		innerPdu = lc.TalkerAliasBlocks[1].Bytes()
	case TalkerAliasBlk3:
		innerPdu = lc.TalkerAliasBlocks[2].Bytes()
	case GpsInfo:
		innerPdu = lc.GpsInfo.Bytes()
	}

	return append(lcHeader, innerPdu...)
}

func (lc *LC) String() string {
	var (
		header = fmt.Sprintf("opcode %d, call type %s, feature set id %d",
			lc.Opcode, dmr.CallTypeName[lc.CallType], lc.FeatureSetID)
		r string
	)

	switch lc.Opcode {
	case GroupVoiceChannelUser:
		fallthrough
	case UnitToUnitVoiceChannelUser:
		r = fmt.Sprintf("%s %v", header, lc.VoiceChannelUser)
	case GpsInfo:
		r = fmt.Sprintf("%s %v", header, lc.GpsInfo)
	case TalkerAliasHeader:
		r = fmt.Sprintf("%s %v", header, lc.TalkerAliasHeader)
	case TalkerAliasBlk1:
		r = fmt.Sprintf("%s %v", header, lc.TalkerAliasBlocks[0])
	case TalkerAliasBlk2:
		r = fmt.Sprintf("%s %v", header, lc.TalkerAliasBlocks[1])
	case TalkerAliasBlk3:
		r = fmt.Sprintf("%s %v", header, lc.TalkerAliasBlocks[2])
	}

	return r
}

// ParseLC parses a packed Link Control message.
func ParseLC(data []byte) (*LC, error) {
	if data == nil {
		return nil, errors.New("dmr/lc: data can't be nil")
	}
	if len(data) != 9 {
		return nil, fmt.Errorf("dmr/lc: expected 9 LC bytes, got %d", len(data))
	}

	if data[0]&dmr.B10000000 > 0 {
		return nil, errors.New("dmr/lc: protect flag is not 0")
	}

	var (
		err  error
		fclo = data[0] & dmr.B00111111
		lc   = &LC{
			Opcode:       fclo,
			FeatureSetID: data[1],
		}
	)
	switch fclo {
	case GroupVoiceChannelUser:
		var pdu *VoiceChannelUserPDU
		lc.CallType = dmr.CallTypeGroup
		pdu, err = ParseVoiceChannelUserPDU(data[2:9])
		lc.VoiceChannelUser = pdu
	case UnitToUnitVoiceChannelUser:
		var pdu *VoiceChannelUserPDU
		lc.CallType = dmr.CallTypePrivate
		pdu, err = ParseVoiceChannelUserPDU(data[2:9])
		lc.VoiceChannelUser = pdu
	case TalkerAliasHeader:
		var pdu *TalkerAliasHeaderPDU
		pdu, err = ParseTalkerAliasHeaderPDU(data[2:9])
		lc.TalkerAliasHeader = pdu
	case TalkerAliasBlk1:
		var pdu *TalkerAliasBlockPDU
		pdu, err = ParseTalkerAliasBlockPDU(data[2:9])
		lc.TalkerAliasBlocks[0] = pdu
	case TalkerAliasBlk2:
		var pdu *TalkerAliasBlockPDU
		pdu, err = ParseTalkerAliasBlockPDU(data[2:9])
		lc.TalkerAliasBlocks[1] = pdu
	case TalkerAliasBlk3:
		var pdu *TalkerAliasBlockPDU
		pdu, err = ParseTalkerAliasBlockPDU(data[2:9])
		lc.TalkerAliasBlocks[2] = pdu
	case GpsInfo:
		var pdu *GpsInfoPDU
		pdu, err = ParseGpsInfoPDU(data[2:9])
		lc.GpsInfo = pdu
	default:
		return nil, fmt.Errorf("dmr/lc: unknown FCLO %06b (%d)", fclo, fclo)
	}

	if err != nil {
		return nil, fmt.Errorf("error parsing link control header pdu: %s", err)
	}

	return lc, nil
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
