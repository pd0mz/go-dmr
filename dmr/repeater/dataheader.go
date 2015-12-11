package repeater

import (
	"github.com/pd0mz/go-dmr/bptc"
	"github.com/pd0mz/go-dmr/dmr"
	"github.com/pd0mz/go-dmr/ipsc"
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
