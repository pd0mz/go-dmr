package dmr

import (
	"golang.org/x/text/encoding"
	"golang.org/x/text/transform"
)

type binaryCoder struct{ transform.NopResetter }

func (e binaryCoder) Transform(dst, src []byte, atEOF bool) (nDst, nSrc int, err error) {
	// The decoder can only make the input larger, not smaller.
	n := len(src)
	if len(dst) < n {
		err = transform.ErrShortDst
		n = len(dst)
		atEOF = false
	} else {
		copy(dst[:n], src)
		nDst = n
		nSrc = n
	}
	return
}

type binaryEncoding struct{}

func (e binaryEncoding) NewDecoder() *encoding.Decoder {
	return &encoding.Decoder{Transformer: binaryCoder{}}
}

func (e binaryEncoding) NewEncoder() *encoding.Encoder {
	return &encoding.Encoder{Transformer: binaryCoder{}}
}
