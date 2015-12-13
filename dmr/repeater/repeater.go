package repeater

import (
	"fmt"
	"log"
	"time"

	"github.com/tehmaze/go-dmr/bit"
	"github.com/tehmaze/go-dmr/ipsc"
)

const (
	Idle uint8 = iota
	VoiceCallRunning
	DataCallRunning
)

type SlotData struct {
	BlocksReceived    uint8
	BlocksExpected    uint8
	PacketHeaderValid bool
}

type SlotVoice struct {
	// Last frame number
	Frame uint8
}

type Slot struct {
	State            uint8
	LastCallReceived time.Time
	CallStart        time.Time
	CallEnd          time.Time
	CallType         uint8
	SrcID, DstID     uint32
	Data             SlotData
	Voice            SlotVoice
	LastSequence     uint8
	LastSlotType     uint16
}

type Repeater struct {
	Slot           []*Slot
	DataFrameFunc  func(*ipsc.Packet, bit.Bits)
	VoiceFrameFunc func(*ipsc.Packet, bit.Bits)
}

func New() *Repeater {
	r := &Repeater{
		Slot: make([]*Slot, 2),
	}
	r.Slot[0] = &Slot{}
	r.Slot[1] = &Slot{}
	return r
}

func (r *Repeater) DataCallEnd(p *ipsc.Packet) {
	if p.Timeslot > 1 {
		return
	}

	slot := r.Slot[p.Timeslot]
	if slot.State != DataCallRunning {
		return
	}

	log.Printf("dmr/repeater: data call ended on TS%d, %d -> %d\n", p.Timeslot+1, slot.SrcID, slot.DstID)

	slot.State = Idle
	slot.CallEnd = time.Now()
	slot.Data.PacketHeaderValid = false
}

func (r *Repeater) VoiceCallStart(p *ipsc.Packet) {
	if p.Timeslot > 1 {
		return
	}

	slot := r.Slot[p.Timeslot]
	if slot.State == VoiceCallRunning {
		r.VoiceCallEnd(p)
	}

	log.Printf("dmr/repeater: voice call started on TS%d, %d -> %d\n", p.Timeslot+1, slot.SrcID, slot.DstID)
	slot.CallStart = time.Now()
	slot.CallType = p.CallType
	slot.SrcID = p.SrcID
	slot.DstID = p.DstID
	slot.State = VoiceCallRunning
}

func (r *Repeater) VoiceCallEnd(p *ipsc.Packet) {
	if p.Timeslot > 1 {
		return
	}

	slot := r.Slot[p.Timeslot]
	if slot.State != VoiceCallRunning {
		return
	}

	log.Printf("dmr/repeater: voice call ended on TS%d, %d -> %d\n", p.Timeslot+1, slot.SrcID, slot.DstID)

	slot.State = Idle
	slot.CallEnd = time.Now()
}

func (r *Repeater) Stream(p *ipsc.Packet) {
	// Kill errneous timeslots here
	if p.Timeslot > 1 {
		log.Printf("killed packet with timeslot %d\n", p.Timeslot)
		return
	}
	if p.Sequence == r.Slot[p.Timeslot].LastSequence {
		return
	}
	r.Slot[p.Timeslot].LastSequence = p.Sequence

	var err error

	fmt.Printf("ts%d/dmr[%03d] [%d->%d]: %s: ", p.Timeslot+1, p.Sequence, p.SrcID, p.DstID, ipsc.SlotTypeName[p.SlotType])
	switch p.SlotType {
	case ipsc.VoiceLCHeader:
		err = r.HandleVoiceLCHeader(p)
	case ipsc.TerminatorWithLC:
		err = r.HandleTerminatorWithLC(p)
	case ipsc.DataHeader:
		err = r.HandleDataHeader(p)
	case ipsc.VoiceDataA, ipsc.VoiceDataB, ipsc.VoiceDataC, ipsc.VoiceDataD, ipsc.VoiceDataE, ipsc.VoiceDataF:
		// De-duplicate packets, since we could run in merged TS1/2 mode
		if r.Slot[p.Timeslot].LastSlotType == p.SlotType {
			fmt.Println("(ignored)")
		} else {
			err = r.HandleVoiceData(p)
		}
		r.Slot[p.Timeslot].LastSlotType = p.SlotType
	default:
		fmt.Println("unhandled")
	}

	if err != nil {
		fmt.Printf("error: %v\n", err)
	}
}
