package repeater

import (
	"github.com/tehmaze/go-dmr/bptc"
	"github.com/tehmaze/go-dmr/dmr"
	"github.com/tehmaze/go-dmr/ipsc"
)

func (r *Repeater) HandleDataHeader(p *ipsc.Packet) error {
	var (
		h       dmr.DataHeader
		err     error
		payload = make([]byte, 12)
	)

	if err = bptc.Process(dmr.ExtractInfoBits(p.PayloadBits), payload); err != nil {
		return err
	}
	if h, err = dmr.ParseDataHeader(payload, false); err != nil {
		return err
	}

	// TODO(maze): handle receiving of data blocks
	switch h.(type) {
	}

	return nil
}
