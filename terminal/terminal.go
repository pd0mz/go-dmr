package terminal

import (
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
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
	voiceCallActive
)

const VoiceFrameDuration = time.Millisecond * 60

type Slot struct {
	call struct {
		start time.Time
		end   time.Time
	}
	dstID, srcID uint32
	dataType     uint8
	data         struct {
		packetHeaderValid bool
		blocks            []*dmr.DataBlock
		blocksExpected    int
		blocksReceived    int
		header            *dmr.DataHeader
	}
	voice struct {
		lastFrame uint8
		streamID  uint32
	}
	selectiveAckRequestsSent int
	rxSequence               int
	fullMessageBlocks        int
	last                     struct {
		packetReceived time.Time
	}
}

func NewSlot() *Slot {
	return &Slot{}
}

type VoiceFrameFunc func(*dmr.Packet, []byte)

type Terminal struct {
	ID            uint32
	Call          string
	CallMap       map[uint32]string
	Repeater      dmr.Repeater
	TalkGroup     []uint32
	SoftwareDelay bool

	accept map[uint32]bool
	slot   []*Slot
	state  uint8
	vff    VoiceFrameFunc
}

func New(id uint32, call string, r dmr.Repeater) *Terminal {
	t := &Terminal{
		ID:       id,
		Call:     call,
		Repeater: r,
		slot:     []*Slot{NewSlot(), NewSlot(), NewSlot()},
		accept:   map[uint32]bool{id: true},
	}

	r.SetPacketFunc(t.handlePacket)
	return t
}

func (t *Terminal) SetTalkGroups(tg []uint32) {
	t.accept = map[uint32]bool{t.ID: true}
	if tg != nil {
		for _, id := range tg {
			t.accept[id] = true
		}
	}
}

func (t *Terminal) SetVoiceFrameFunc(f VoiceFrameFunc) {
	t.vff = f
}

func (t *Terminal) Send(p *dmr.Packet) error {
	return t.Repeater.Send(p)
}

func (t *Terminal) fmt(p *dmr.Packet, f string) string {
	var fp = []string{
		fmt.Sprintf("[slot %d][%02x][",
			p.Timeslot+1,
			p.Sequence),
	}
	if t.CallMap != nil {
		if call, ok := t.CallMap[p.SrcID]; ok {
			fp = append(fp, fmt.Sprintf("%-6s->", call))
		} else {
			fp = append(fp, fmt.Sprintf("%-6d->", p.SrcID))
		}
		if call, ok := t.CallMap[p.DstID]; ok {
			fp = append(fp, fmt.Sprintf("%-6s] ", call))
		} else {
			fp = append(fp, fmt.Sprintf("%-6d] ", p.DstID))
		}
	} else {
		fp = append(fp, fmt.Sprintf("%-6d->%-6d] ", p.SrcID, p.DstID))
	}
	fp = append(fp, f)
	return strings.Join(fp, "")
}

func (t *Terminal) debugf(p *dmr.Packet, f string, v ...interface{}) {
	ff := t.fmt(p, f)
	if len(v) > 0 {
		log.Debugf(ff, v...)
	} else {
		log.Debug(ff)
	}
}

func (t *Terminal) infof(p *dmr.Packet, f string, v ...interface{}) {
	ff := t.fmt(p, f)
	if len(v) > 0 {
		log.Infof(ff, v...)
	} else {
		log.Info(ff)
	}
}

func (t *Terminal) warningf(p *dmr.Packet, f string, v ...interface{}) {
	ff := t.fmt(p, f)
	if len(v) > 0 {
		log.Warningf(ff, v...)
	} else {
		log.Warning(ff)
	}
}

func (t *Terminal) errorf(p *dmr.Packet, f string, v ...interface{}) {
	ff := t.fmt(p, f)
	if len(v) > 0 {
		log.Errorf(ff, v...)
	} else {
		log.Error(ff)
	}
}

func (t *Terminal) dataBlock(p *dmr.Packet, db *dmr.DataBlock) error {
	slot := t.slot[p.Timeslot]

	if slot.data.header == nil {
		return errors.New("terminal: logic error, header is nil?!")
	}
	if slot.data.header.ResponseRequested {
		// Only confirmed data blocks have serial numbers stored in them.
		if int(db.Serial) < len(slot.data.blocks) {
			slot.data.blocks[db.Serial] = db
		} else {
			t.warningf(p, "data block %d out of bounds (%d >= %d)", db.Serial, db.Serial, len(slot.data.blocks))
			return nil
		}
	} else {
		slot.data.blocks[slot.data.blocksReceived] = db
	}

	slot.data.blocksReceived++
	if slot.data.blocksReceived == slot.data.blocksExpected {
		return t.dataBlockAssemble(p)
	}

	return nil
}

func (t *Terminal) dataBlockAssemble(p *dmr.Packet) error {
	slot := t.slot[p.Timeslot]

	var (
		errorsFound bool
		selective   = make([]bool, len(slot.data.blocks))
	)
	for i := 0; i < slot.fullMessageBlocks; i++ {
		if slot.data.blocks[i] == nil || !slot.data.blocks[i].OK {
			selective[i] = true
			errorsFound = true
		}
	}

	if errorsFound {
		_, responseOk := slot.data.header.Data.(*dmr.ResponseData)
		switch {
		case responseOk:
			t.debugf(p, "found erroneous blocks, not sending out ACK for response")
			return nil
		case slot.selectiveAckRequestsSent > 25:
			t.warningf(p, "found erroneous blocks, max selective ACK reached")
			return nil
		default:
			//t.sendSelectiveAck()
			return nil
		}
	}

	fragment, err := dmr.CombineDataBlocks(slot.data.blocks)
	if err != nil {
		return err
	}

	if fragment.Stored > 0 {
		// Response with data blocks? That must be a selective ACK
		if _, ok := slot.data.header.Data.(*dmr.ResponseData); ok {
			// FIXME(pd0mz): deal with this shit
			return nil
		}

		if err := t.dataBlockComplete(p, fragment); err != nil {
			return err
		}

		// If we are not waiting for an ack, then the data session ended
		if !slot.data.header.ResponseRequested {
			return t.dataCallEnd(p)
		}
	}
	return nil
}

func (t *Terminal) dataBlockComplete(p *dmr.Packet, f *dmr.DataFragment) error {
	slot := t.slot[p.Timeslot]

	var (
		data     []byte
		size     int
		ddformat = dmr.DDFormatUTF16
	)

	switch slot.data.header.ServiceAccessPoint {
	case dmr.ServiceAccessPointIPBasedPacketData:
		t.debugf(p, "SAP IP based packet data (not implemented)")
		break

	case dmr.ServiceAccessPointShortData:
		t.debugf(p, "SAP short data")

		data = f.Data[2:]       // Hytera has a 2 byte pre-padding
		size = f.Stored - 2 - 4 // Leave out the CRC

		if sdd, ok := slot.data.header.Data.(*dmr.ShortDataDefinedData); ok {
			ddformat = sdd.DDFormat
		}
		break
	}

	if data == nil || size == 0 {
		t.warningf(p, "no data in message")
		return nil

	}

	message, err := dmr.ParseMessageData(data[:size], ddformat, true)
	if err != nil {
		return err
	}

	t.infof(p, "message %q", message)
	return nil
}

func (t *Terminal) callEnd(p *dmr.Packet) error {
	switch t.state {
	case dataCallActive:
		return t.dataCallEnd(p)
	case voiceCallActive:
		return t.voiceCallEnd(p)
	}
	return nil
}

func (t *Terminal) dataCallEnd(p *dmr.Packet) error {
	slot := t.slot[p.Timeslot]

	if t.state != dataCallActive {
		return nil
	}

	slot.data.packetHeaderValid = false
	t.state = idle
	t.debugf(p, "data call ended")
	return nil
}

func (t *Terminal) dataCallStart(p *dmr.Packet) error {
	slot := t.slot[p.Timeslot]

	if slot.dstID != p.DstID || slot.srcID != p.SrcID || slot.dataType != p.DataType {
		if t.state == dataCallActive {
			if err := t.dataCallEnd(p); err != nil {
				return err
			}
		}
	}

	slot.data.packetHeaderValid = false
	slot.call.start = time.Now()
	slot.call.end = time.Time{}
	slot.dstID = p.DstID
	slot.srcID = p.SrcID
	t.state = dataCallActive
	t.debugf(p, "data call started")
	return nil
}

func (t *Terminal) voiceCallEnd(p *dmr.Packet) error {
	slot := t.slot[p.Timeslot]

	if t.state != voiceCallActive {
		return nil
	}

	slot.voice.streamID = 0
	t.state = idle
	t.debugf(p, "voice call ended")
	return nil
}

func (t *Terminal) voiceCallStart(p *dmr.Packet) error {
	slot := t.slot[p.Timeslot]

	if slot.dstID != p.DstID || slot.srcID != p.SrcID {
		if err := t.voiceCallEnd(p); err != nil {
			return err
		}
	}

	slot.voice.streamID = p.StreamID
	t.state = voiceCallActive

	t.debugf(p, "voice call started")
	return nil
}

func (t *Terminal) handlePacket(r dmr.Repeater, p *dmr.Packet) error {
	// Ignore packets not addressed to us or any of the talk groups we monitor
	if !t.accept[p.DstID] {
		//log.Debugf("[%d->%d] (%s, %#04b): ignored, not sent to me", p.SrcID, p.DstID, dmr.DataTypeName[p.DataType], p.DataType)
		return nil
	}

	var err error

	t.debugf(p, dmr.DataTypeName[p.DataType])
	switch p.DataType {
	case dmr.CSBK:
		err = t.handleControlBlock(p)
		break
	case dmr.Data:
		err = t.handleData(p)
		break
	case dmr.Rate34Data:
		err = t.handleRate34Data(p)
		break
	case dmr.VoiceBurstA, dmr.VoiceBurstB, dmr.VoiceBurstC, dmr.VoiceBurstD, dmr.VoiceBurstE, dmr.VoiceBurstF:
		err = t.handleVoice(p)
		break
	case dmr.VoiceLC:
		return nil
	case dmr.TerminatorWithLC:
		err = t.handleTerminatorWithLC(p)
		return nil
	default:
		log.Debug(hex.Dump(p.Data))
		return nil
	}

	if err != nil {
		t.errorf(p, "handle packet error: %v", err)
	}

	return err
}

func (t *Terminal) handleControlBlock(p *dmr.Packet) error {
	slot := t.slot[p.Timeslot]
	slot.last.packetReceived = time.Now()

	// This ends both data and voice calls
	if err := t.callEnd(p); err != nil {
		return err
	}

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

	t.debugf(p, cb.String())

	return nil
}

func (t *Terminal) handleData(p *dmr.Packet) error {
	slot := t.slot[p.Timeslot]
	slot.last.packetReceived = time.Now()

	// This ends voice calls (if any)
	if err := t.voiceCallEnd(p); err != nil {
		return err
	}

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

	t.debugf(p, "data header: %T", h)
	switch d := h.Data.(type) {
	case dmr.ShortDataDefinedData:
		if d.FullMessage {
			slot.fullMessageBlocks = int(d.AppendedBlocks)
			slot.data.blocks = make([]*dmr.DataBlock, slot.fullMessageBlocks)
			t.debugf(p, "expecting %d data block", slot.fullMessageBlocks)
		}
		slot.data.blocksExpected = int(d.AppendedBlocks)
		err = t.dataCallStart(p)
		break

	default:
		t.warningf(p, "unhandled data header %T", h)
		return nil
	}

	if err == nil {
		slot.data.header = h
		slot.data.packetHeaderValid = true
	}
	return err
}

func (t *Terminal) handleRate34Data(p *dmr.Packet) error {
	slot := t.slot[p.Timeslot]
	slot.last.packetReceived = time.Now()

	if t.state != dataCallActive {
		t.debugf(p, "no data call in process, ignoring rate ¾ data")
		return nil
	}
	if slot.data.header == nil {
		t.warningf(p, "got rate ¾ data, but no data header stored")
		return nil
	}

	var (
		bits = p.InfoBits()
		data = make([]byte, 18)
	)

	if err := trellis.Decode(bits, data); err != nil {
		return err
	}

	db, err := dmr.ParseDataBlock(data, dmr.Rate34Data, slot.data.header.ResponseRequested)
	if err != nil {
		return err
	}

	return t.dataBlock(p, db)
}

func (t *Terminal) handleTerminatorWithLC(p *dmr.Packet) error {
	// This ends both data and voice calls
	return t.callEnd(p)
}

func (t *Terminal) handleVoice(p *dmr.Packet) error {
	slot := t.slot[p.Timeslot]
	slot.last.packetReceived = time.Now()

	var (
		bits = p.VoiceBits()
	)

	switch t.state {
	case voiceCallActive:
		if p.StreamID != slot.voice.streamID {
			// Only accept voice frames from the same stream
			t.debugf(p, "ignored frame, active stream id: %#08x, this stream id: %#08x", slot.voice.streamID, p.StreamID)
			return nil
		}
	default:
		t.voiceCallStart(p)
		break
	}

	if t.vff != nil {
		t.vff(p, bits)
		if t.SoftwareDelay {
			delta := time.Now().Sub(slot.last.packetReceived)
			if delta < VoiceFrameDuration {
				delay := VoiceFrameDuration - delta
				time.Sleep(delay)
				t.debugf(p, "software delay of %s", delay)
			}
		}
	}

	return nil
}
