package dmr

// Data Type information element definitions, DMR Air Interface (AI) protocol, Table 6.1
const (
	PrivacyIndicator              uint8 = iota // Privacy Indicator information in a standalone burst
	VoiceLC                                    // Indicates the beginning of voice transmission, carries addressing information
	TerminatorWithLC                           // Indicates the end of transmission, carries LC information
	CSBK                                       // Carries a control block
	MultiBlockControl                          // Header for multi-block control
	MultiBlockControlContinuation              // Follow-on blocks for multi-block control
	Data                                       // Carries addressing and numbering of packet data blocks
	Rate12Data                                 // Payload for rate 1/2 packet data
	Rate34Data                                 // Payload for rate 3⁄4 packet data
	Idle                                       // Fills channel when no info to transmit
	VoiceBurstA                                // Burst A marks the start of a superframe and always contains a voice SYNC pattern
	VoiceBurstB                                // Bursts B to F carry embedded signalling in place of the SYNC pattern
	VoiceBurstC                                // Bursts B to F carry embedded signalling in place of the SYNC pattern
	VoiceBurstD                                // Bursts B to F carry embedded signalling in place of the SYNC pattern
	VoiceBurstE                                // Bursts B to F carry embedded signalling in place of the SYNC pattern
	VoiceBurstF                                // Bursts B to F carry embedded signalling in place of the SYNC pattern
	IPSCSync
	UnknownSlotType
)

var DataTypeName = map[uint8]string{
	PrivacyIndicator:              "privacy indicator",
	VoiceLC:                       "voice LC",
	TerminatorWithLC:              "terminator with LC",
	CSBK:                          "control block",
	MultiBlockControl:             "multi-block control",
	MultiBlockControlContinuation: "multi-block control follow-on",
	Data:            "data",
	Rate12Data:      "rate ½ packet data",
	Rate34Data:      "rate ¾ packet data",
	Idle:            "idle",
	VoiceBurstA:     "voice (burst A)",
	VoiceBurstB:     "voice (burst B)",
	VoiceBurstC:     "voice (burst C)",
	VoiceBurstD:     "voice (burst D)",
	VoiceBurstE:     "voice (burst E)",
	VoiceBurstF:     "voice (burst F)",
	IPSCSync:        "IPSC sync",
	UnknownSlotType: "uknown",
}

// Call Type
const (
	CallTypePrivate uint8 = iota
	CallTypeGroup
)

var CallTypeName = map[uint8]string{
	CallTypePrivate: "private",
	CallTypeGroup:   "group",
}

// Packet represents a frame transported by the Air Interface
type Packet struct {
	// 0 for slot 1, 1 for slot 2
	Timeslot uint8

	// Starts at zero for each incoming transmission, wraps back to zero when 256 is reached
	Sequence uint8

	// Source and destination DMR ID
	SrcID uint32
	DstID uint32

	// 3 bytes registered DMR-ID for public repeaters, 4 bytes for private repeaters
	RepeaterID uint32

	// Random or incremented number which stays the same from PTT-press to PTT-release which identifies a stream
	StreamID uint32

	// Data Type or Slot type
	DataType uint8

	// 0 for group call, 1 for unit to unit
	CallType uint8

	// The on-air DMR data with possible FEC fixes to the AMBE data and/or Slot Type and/or EMB, etc
	Data []byte // 34 bytes
	Bits []byte // 264 bits
}

// EMBBits returns the frame EMB bits from the SYNC bits
func (p *Packet) EMBBits() []byte {
	var (
		b    = make([]byte, EMBBits)
		o    = EMBHalfBits + EMBSignallingLCFragmentBits
		sync = p.SyncBits()
	)
	copy(b[:EMBHalfBits], sync[:EMBHalfBits])
	copy(b[EMBHalfBits:], sync[o:o+EMBHalfBits])
	return b
}

// InfoBits returns the frame Info bits
func (p *Packet) InfoBits() []byte {
	var b = make([]byte, InfoBits)
	copy(b[0:InfoHalfBits], p.Bits[0:InfoHalfBits])
	copy(b[InfoHalfBits:], p.Bits[InfoHalfBits+SlotTypeBits+SignalBits:])
	return b
}

// SyncBits returns the frame SYNC bits
func (p *Packet) SyncBits() []byte {
	return p.Bits[SyncOffsetBits : SyncOffsetBits+SyncBits]
}

// SlotType returns the frame Slot Type parsed from the Slot Type bits
func (p *Packet) SlotType() []byte {
	return BitsToBytes(p.SlotTypeBits())
}

// SlotTypeBits returns the SloT Type bits
func (p *Packet) SlotTypeBits() []byte {
	var o = InfoHalfBits + SlotTypeHalfBits + SyncBits
	return append(p.Bits[InfoHalfBits:InfoHalfBits+SlotTypeHalfBits], p.Bits[o:o+SlotTypeHalfBits]...)
}

// VoiceBits returns the bits containing voice data
func (p *Packet) VoiceBits() []byte {
	var b = make([]byte, VoiceBits)
	copy(b[:VoiceHalfBits], p.Bits[:VoiceHalfBits])
	copy(b[VoiceHalfBits:], p.Bits[VoiceHalfBits+SignalBits:])
	return b
}

func (p *Packet) SetData(data []byte) {
	p.Data = data
	p.Bits = BytesToBits(data)
}

// PacketFunc is a callback function that handles DMR packets
type PacketFunc func(Repeater, *Packet) error
