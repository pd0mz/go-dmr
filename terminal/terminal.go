package terminal

import (
	"encoding/hex"
	"fmt"
	"time"

	"github.com/op/go-logging"
	"github.com/pd0mz/go-dmr"
	"github.com/pd0mz/go-dmr/bptc"
	"github.com/pd0mz/go-dmr/trellis"
)

var log = logging.MustGetLogger("dmr/terminal")

const (
	idle uint8 = iota
	dataCallActive
	voideCallActive
)

type Slot struct {
	call struct {
		start time.Time
		end   time.Time
	}
	dstID, srcID uint32
	dataType     uint8
	data         struct {
		packetHeaderValid bool
		blocks            [64]dmr.DataBlock
		blocksExpected    uint8
		blocksReceived    uint8
	}
	selectiveAckRequestsSent int
	rxSequence               int
	fullMessageBlocks        uint8
	last                     struct {
		packetReceived time.Time
	}
}

func NewSlot() Slot {
	return Slot{}
}

type Terminal struct {
	ID       uint32
	Call     string
	Repeater dmr.Repeater
	slot     []Slot
	state    uint8
}

func New(id uint32, call string, r dmr.Repeater) *Terminal {
	t := &Terminal{
		ID:       id,
		Call:     call,
		Repeater: r,
		slot:     []Slot{NewSlot(), NewSlot()},
	}
	r.SetPacketFunc(t.handlePacket)
	return t
}

func (t *Terminal) dataCallEnd(p *dmr.Packet) error {
	slot := t.slot[p.Timeslot]

	if t.state != dataCallActive {
		return nil
	}

	log.Debugf("[%d->%d] data call ended", slot.srcID, slot.dstID)

	return nil
}

func (t *Terminal) dataCallStart(p *dmr.Packet) error {
	slot := t.slot[p.Timeslot]

	if slot.dstID != p.DstID || slot.srcID != p.SrcID || slot.dataType != p.DataType {
		if err := t.dataCallEnd(p); err != nil {
			return err
		}
	}

	slot.data.packetHeaderValid = false
	slot.call.start = time.Now()
	slot.call.end = time.Time{}
	slot.dstID = p.DstID
	slot.srcID = p.SrcID
	t.state = dataCallActive

	log.Debugf("[%d->%d] data call started", slot.srcID, slot.dstID)

	return nil
}

func (t *Terminal) handlePacket(r dmr.Repeater, p *dmr.Packet) error {
	var err error
	if p.DstID != t.ID {
		//log.Debugf("[%d->%d] (%s, %#04b): ignored, not sent to me", p.SrcID, p.DstID, dmr.DataTypeName[p.DataType], p.DataType)
		return nil
	}

	switch p.DataType {
	case dmr.VoiceBurstA, dmr.VoiceBurstB, dmr.VoiceBurstC, dmr.VoiceBurstD, dmr.VoiceBurstE, dmr.VoiceBurstF:
		return nil
	case dmr.CBSK:
		return nil
	}

	log.Debugf("[%d->%d] (%s, %#04b): sent to me", p.SrcID, p.DstID, dmr.DataTypeName[p.DataType], p.DataType)

	switch p.DataType {
	case dmr.CBSK:
		err = t.handleControlBlock(p)
		break
	case dmr.Data:
		err = t.handleData(p)
		break
	case dmr.Rate34Data:
		err = t.handleRate34Data(p)
	default:
		log.Debug(hex.Dump(p.Data))
	}

	if err != nil {
		log.Errorf("handle packet error: %v", err)
	}

	return err
}

func (t *Terminal) handleControlBlock(p *dmr.Packet) error {
	slot := t.slot[p.Timeslot]
	slot.last.packetReceived = time.Now()

	var (
		bits = p.InfoBits()
		data = make([]byte, 12)
	)

	if err := bptc.Decode(bits, data); err != nil {
		return err
	}
	cb, err := dmr.ParseControlBlock(data)
	if err != nil {
		return err
	}

	log.Debugf("[%d->%d] control block %T", cb.SrcID, cb.DstID, cb.Data)

	return nil
}

func (t *Terminal) handleData(p *dmr.Packet) error {
	slot := t.slot[p.Timeslot]
	slot.last.packetReceived = time.Now()

	var (
		bits = p.InfoBits()
		data = make([]byte, 12)
	)

	if err := bptc.Decode(bits, data); err != nil {
		return err
	}

	h, err := dmr.ParseDataHeader(data, false)
	if err != nil {
		return err
	}

	slot.data.packetHeaderValid = false
	slot.data.blocksReceived = 0
	slot.selectiveAckRequestsSent = 0
	slot.rxSequence = 0

	c := h.CommonHeader()
	log.Debugf("[%d->%d] data header %T", c.SrcID, c.DstID, h)

	switch ht := h.(type) {
	case dmr.ShortDataDefinedDataHeader:
		if ht.FullMessage {
			slot.data.blocks = [64]dmr.DataBlock{}
			slot.fullMessageBlocks = ht.AppendedBlocks
			log.Debugf("[%d->%d] expecting %d data blocks", c.SrcID, c.DstID, slot.fullMessageBlocks)
		}
		slot.data.blocksExpected = ht.AppendedBlocks
		return t.dataCallStart(p)
	default:
		log.Warningf("unhandled data header %T", h)
	}

	return nil
}

func (t *Terminal) handleRate34Data(p *dmr.Packet) error {
	slot := t.slot[p.Timeslot]
	slot.last.packetReceived = time.Now()

	var (
		bits = p.InfoBits()
		data = make([]byte, 18)
	)

	if err := trellis.Decode(bits, data); err != nil {
		return err
	}
	fmt.Print(hex.Dump(data))

	return nil
}
