package homebrew

import (
	"fmt"

	"github.com/pd0mz/go-dmr/bit"
	"github.com/pd0mz/go-dmr/ipsc"
)

// ParseData reads a raw DMR data frame from the homebrew protocol and parses it as it were an IPSC packet.
func ParseData(data []byte) (*ipsc.Packet, error) {
	if len(data) != 53 {
		return nil, fmt.Errorf("invalid packet length %d, expected 53 bytes", len(data))
	}

	var (
		slotType  uint16
		dataType  = (data[15] & bit.B11110000) >> 4
		frameType = (data[15] & bit.B00001100) >> 2
	)
	switch frameType {
	case bit.B00000000, bit.B00000001: // voice/voice sync
		switch dataType {
		case bit.B00000000:
			slotType = ipsc.VoiceDataA
		case bit.B00000001:
			slotType = ipsc.VoiceDataB
		case bit.B00000010:
			slotType = ipsc.VoiceDataC
		case bit.B00000011:
			slotType = ipsc.VoiceDataD
		case bit.B00000100:
			slotType = ipsc.VoiceDataE
		case bit.B00000101:
			slotType = ipsc.VoiceDataF
		}
	case bit.B00000010: // data sync
		switch dataType {
		case bit.B00000001:
			slotType = ipsc.VoiceLCHeader
		case bit.B00000010:
			slotType = ipsc.TerminatorWithLC
		case bit.B00000011:
			slotType = ipsc.CSBK
		case bit.B00000110:
			slotType = ipsc.DataHeader
		case bit.B00000111:
			slotType = ipsc.Rate12Data
		case bit.B00001000:
			slotType = ipsc.Rate34Data
		}
	}

	return &ipsc.Packet{
		Timeslot:    (data[15] & bit.B00000001),
		CallType:    (data[15] & bit.B00000010) >> 1,
		FrameType:   (data[15] & bit.B00001100) >> 2,
		SlotType:    slotType,
		SrcID:       uint32(data[5])<<16 | uint32(data[6])<<8 | uint32(data[7]),
		DstID:       uint32(data[8])<<16 | uint32(data[9])<<8 | uint32(data[10]),
		Payload:     data[20:],
		PayloadBits: bit.NewBits(data[20:]),
		Sequence:    data[4],
	}, nil
}
