package lc

import (
	"fmt"

	"github.com/pd0mz/go-dmr/lc/serviceoptions"
)

// VoiceChannelUserPDU Conforms to ETSI TS 102-361-2 7.1.1.(1 and 2)
type VoiceChannelUserPDU struct {
	ServiceOptions serviceoptions.ServiceOptions
	DstID          uint32
	SrcID          uint32
}

// ParseVoiceChannelUserPDU Parses either Group Voice Channel User or
// Unit to Unit Channel User PDUs
func ParseVoiceChannelUserPDU(data []byte) (*VoiceChannelUserPDU, error) {
	if len(data) != 7 {
		return nil, fmt.Errorf("dmr/lc/voicechanneluser: expected 7 bytes, got %d", len(data))
	}

	return &VoiceChannelUserPDU{
		ServiceOptions: serviceoptions.ParseServiceOptions(data[0]),
		DstID:          uint32(data[1])<<16 | uint32(data[2])<<8 | uint32(data[3]),
		SrcID:          uint32(data[4])<<16 | uint32(data[5])<<8 | uint32(data[6]),
	}, nil
}

// Bytes packs the Voice Channel User PDU message to bytes.
func (v *VoiceChannelUserPDU) Bytes() []byte {
	return []byte{
		v.ServiceOptions.Byte(),
		uint8(v.DstID >> 16),
		uint8(v.DstID >> 8),
		uint8(v.DstID),
		uint8(v.SrcID >> 16),
		uint8(v.SrcID >> 8),
		uint8(v.SrcID),
	}
}

func (v *VoiceChannelUserPDU) String() string {
	return fmt.Sprintf("VoiceChannelUser: [ %d->%d, service options %s ]",
		v.SrcID, v.DstID, v.ServiceOptions.String())
}
