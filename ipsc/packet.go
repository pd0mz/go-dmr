package ipsc

import (
	"encoding/hex"
	"fmt"

	"github.com/tehmaze/go-dmr/bit"
	"github.com/tehmaze/go-dmr/dmr"
)

const (
	VoiceLCHeader    uint16 = 0x1111
	TerminatorWithLC uint16 = 0x2222
	CSBK             uint16 = 0x3333
	DataHeader       uint16 = 0x4444
	Rate12Data       uint16 = 0x5555
	Rate34Data       uint16 = 0x6666
	VoiceDataA       uint16 = 0xbbbb
	VoiceDataB       uint16 = 0xcccc
	VoiceDataC       uint16 = 0x7777
	VoiceDataD       uint16 = 0x8888
	VoiceDataE       uint16 = 0x9999
	VoiceDataF       uint16 = 0xaaaa
	IPSCSync         uint16 = 0xeeee
	UnknownSlotType  uint16 = 0x0000
)

const (
	CallTypePrivate uint8 = iota
	CallTypeGroup
)

var (
	TimeslotName = map[uint8]string{
		0x00: "TS1",
		0x01: "TS2",
	}
	FrameTypeName = map[uint8]string{
		0x00: "voice",
		0x01: "voice sync",
		0x02: "data sync",
		0x03: "unused",
	}
	SlotTypeName = map[uint16]string{
		0x0000: "unknown",
		0x1111: "voice LC header",
		0x2222: "terminator with LC",
		0x3333: "CBSK",
		0x4444: "data header",
		0x5555: "rate 1/2 data",
		0x6666: "rate 3/4 data",
		0x7777: "voice data C",
		0x8888: "voice data D",
		0x9999: "voice data E",
		0xaaaa: "voice data F",
		0xbbbb: "voice data A",
		0xcccc: "voice data B",
		0xeeee: "IPSC sync",
	}
	CallTypeName = map[uint8]string{
		0x00: "private",
		0x01: "group",
	}
)

type Packet struct {
	Timeslot    uint8 // 0=ts1, 1=ts2
	FrameType   uint8
	SlotType    uint16
	CallType    uint8 // 0=private, 1=group
	SrcID       uint32
	DstID       uint32
	Payload     []byte   // 34 bytes
	PayloadBits bit.Bits // 264 bits
	Sequence    uint8
}

func (p *Packet) Dump() string {
	var s string
	s += fmt.Sprintf("timeslot..: 0b%02b (%s)\n", p.Timeslot, TimeslotName[p.Timeslot])
	s += fmt.Sprintf("frame type: 0b%02b (%s)\n", p.FrameType, FrameTypeName[p.FrameType])
	s += fmt.Sprintf("slot type.: 0x%04x (%s)\n", p.SlotType, SlotTypeName[p.SlotType])
	s += fmt.Sprintf("call type.: 0x%02x (%s)\n", p.CallType, CallTypeName[p.CallType])
	s += fmt.Sprintf("source....: %d\n", p.SrcID)
	s += fmt.Sprintf("target....: %d\n", p.DstID)
	s += fmt.Sprintf("payload...: %d bytes (swapped):\n", len(p.Payload))
	s += hex.Dump(p.Payload)
	s += fmt.Sprintf("payload...: %d bits:\n", len(p.PayloadBits))
	s += p.PayloadBits.Dump()
	return s
}

func (p *Packet) InfoBits() bit.Bits {
	var b = make(bit.Bits, dmr.InfoBits)
	copy(b[0:dmr.InfoHalfBits], p.PayloadBits[0:dmr.InfoHalfBits])
	copy(b[dmr.InfoHalfBits:], p.PayloadBits[dmr.InfoHalfBits+dmr.SlotTypeBits+dmr.SignalBits:])
	return b
}

func (p *Packet) VoiceBits() bit.Bits {
	var b = make(bit.Bits, dmr.VoiceBits)
	copy(b[:dmr.VoiceHalfBits], p.PayloadBits[:dmr.VoiceHalfBits])
	copy(b[dmr.VoiceHalfBits:], p.PayloadBits[dmr.VoiceHalfBits+dmr.SignalBits:])
	return b
}
