package repeater

import (
	"errors"
	"fmt"

	"github.com/tehmaze/go-dmr/bit"
	"github.com/tehmaze/go-dmr/bptc"
	"github.com/tehmaze/go-dmr/dmr"
	"github.com/tehmaze/go-dmr/fec"
	"github.com/tehmaze/go-dmr/ipsc"
)

type LC struct {
	CallType     uint8
	DstID, SrcID uint32
}

func (r *Repeater) HandleTerminatorWithLC(p *ipsc.Packet) error {
	r.DataCallEnd(p)

	var (
		err     error
		payload = make([]byte, 12)
	)
	if err = bptc.Process(dmr.ExtractInfoBits(p.PayloadBits), payload); err != nil {
		return err
	}

	// CRC mask to the checksum. See DMR AI. spec. page 143.
	payload[9] ^= 0x99
	payload[10] ^= 0x99
	payload[11] ^= 0x99

	var lc *LC
	if lc, err = r.lcDecodeFullLC(payload); err != nil {
		return err
	}

	fmt.Printf("  lc: %d -> %d\n", lc.SrcID, lc.DstID)
	return nil
}

func (r *Repeater) HandleVoiceLCHeader(p *ipsc.Packet) error {
	r.DataCallEnd(p)

	var (
		err     error
		payload = make([]byte, 12)
	)
	if err = bptc.Process(dmr.ExtractInfoBits(p.PayloadBits), payload); err != nil {
		return err
	}

	// CRC mask to the checksum. See DMR AI. spec. page 143
	for i := 9; i < 12; i++ {
		payload[i] ^= 0x99
	}

	var lc *LC
	if lc, err = r.lcDecodeFullLC(payload); err != nil {
		return err
	}

	fmt.Printf("  lc: %d -> %d\n", lc.SrcID, lc.DstID)
	return nil
}

func (r *Repeater) lcDecode(payload []byte) (*LC, error) {
	if payload[0]&bit.B10000000 > 0 {
		return nil, errors.New("dmr/lc: protect flag is not 0")
	}
	if payload[1] != 0 {
		return nil, errors.New("dmr/lc: feature set ID is not 0")
	}

	lc := &LC{}
	switch payload[0] & bit.B00111111 {
	case 3:
		lc.CallType = ipsc.CallTypePrivate
	case 0:
		lc.CallType = ipsc.CallTypeGroup
	default:
		return nil, fmt.Errorf("dmr/lc: invalid FCLO; unknown call type %#02x", payload[0]&bit.B00111111)
	}

	lc.DstID = uint32(payload[3])<<16 | uint32(payload[4])<<8 | uint32(payload[5])
	lc.SrcID = uint32(payload[6])<<16 | uint32(payload[7])<<8 | uint32(payload[8])
	return lc, nil
}

func (r *Repeater) lcDecodeFullLC(payload []byte) (*LC, error) {
	var (
		err      error
		syndrome = fec.RS_12_9_Poly{}
	)
	if err = fec.RS_12_9_CalcSyndrome(payload, &syndrome); err != nil {
		return nil, err
	}

	if fec.RS_12_9_CheckSyndrome(&syndrome) {
		if _, err = fec.RS_12_9_Correct(payload, &syndrome); err != nil {
			return nil, err
		}
	}

	return r.lcDecode(payload)
}
