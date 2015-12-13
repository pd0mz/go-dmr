package repeater

import (
	"errors"
	"fmt"
	"log"

	"github.com/tehmaze/go-dmr/bptc"
	"github.com/tehmaze/go-dmr/dmr"
	"github.com/tehmaze/go-dmr/ipsc"
)

var voiceFrameMap = map[uint16]uint8{
	ipsc.VoiceDataA: 0,
	ipsc.VoiceDataB: 1,
	ipsc.VoiceDataC: 2,
	ipsc.VoiceDataD: 3,
	ipsc.VoiceDataE: 4,
	ipsc.VoiceDataF: 5,
}

func (r *Repeater) HandleVoiceData(p *ipsc.Packet) error {
	r.DataCallEnd(p)

	var (
		err     error
		payload = make([]byte, 12)
	)

	if err = bptc.Process(dmr.ExtractInfoBits(p.PayloadBits), payload); err != nil {
		return err
	}
	if r.Slot[p.Timeslot].State != VoiceCallRunning {
		r.VoiceCallStart(p)
	}

	return r.HandleVoiceFrame(p)
}

func (r *Repeater) HandleVoiceFrame(p *ipsc.Packet) error {
	// This may contain a sync frame
	sync := dmr.ExtractSyncBits(p.PayloadBits)
	patt := dmr.SyncPattern(sync)
	if patt != dmr.SyncPatternUnknown && r.Slot[p.Timeslot].Voice.Frame != 0 {
		fmt.Printf("sync pattern %s\n", dmr.SyncPatternName[patt])
		r.Slot[p.Timeslot].Voice.Frame = 0
		return nil
	} else {
		// This may be a duplicate frame
		var (
			oldFrame = r.Slot[p.Timeslot].Voice.Frame
			newFrame = voiceFrameMap[p.SlotType]
		)
		switch {
		case oldFrame > 5:
			// Ignore, wait for next sync frame
			return nil
		case newFrame == oldFrame:
			return errors.New("dmr/voice: ignored, duplicate frame")
		case newFrame > oldFrame:
			if newFrame-oldFrame > 1 {
				log.Printf("dmr/voice: framedrop, went from %c -> %c", 'A'+oldFrame, 'A'+newFrame)
			}
		case newFrame < oldFrame:
			if newFrame > 0 {
				log.Printf("dmr/voice: framedrop, went from %c -> %c", 'A'+oldFrame, 'A'+newFrame)
			}
		}
		r.Slot[p.Timeslot].Voice.Frame++
	}

	// If it's not a sync frame, then it should have an EMB inside the sync field.
	var (
		emb *dmr.EMB
		err error
	)
	if emb, err = dmr.ParseEMB(dmr.ExtractEMBBitsFromSyncBits(sync)); err != nil {
		fmt.Println("unknown sync pattern, no EMB")
		return err
	}

	fmt.Printf("EMB LCSS %d\n", emb.LCSS)

	// TODO(maze): implement VBPTC matrix handling
	switch emb.LCSS {
	}

	return nil
}
