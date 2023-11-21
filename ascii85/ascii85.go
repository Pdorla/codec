// *********************************************************************************************************************
// ***                                        CONFIDENTIAL --- CUSTOM STUDIOS                                        ***
// *********************************************************************************************************************
// * Auth: ColeCai                                                                                                     *
// * Date: 2023/11/21 21:21:43                                                                                         *
// * Proj: codec                                                                                                       *
// * Pack: ascii85                                                                                                     *
// * File: ascii85.go                                                                                                  *
// *-------------------------------------------------------------------------------------------------------------------*
// * Overviews:                                                                                                        *
// *-------------------------------------------------------------------------------------------------------------------*
// * Functions:                                                                                                        *
// * - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - *

package ascii85

import (
	"github.com/caijunjun/codec/base"
	"github.com/pkg/errors"
)

var (
	invalieIllegalFormat = "codec/asii85: illegal ascii85 data at input byte %+v."
	StdCodec             = NewCodec()
)

type ascii85Codec struct{}

func NewCodec() base.IEncoding {
	return &ascii85Codec{}
}

func (c *ascii85Codec) maxEncodeLen(n int) int {
	return (n + 3) / 4 * 5
}

func (c *ascii85Codec) encode(dst, src []byte) int {
	if len(src) == 0 {
		return 0
	}
	n := 0
	for len(src) > 0 {
		dst[0] = 0
		dst[1] = 0
		dst[2] = 0
		dst[3] = 0
		dst[4] = 0

		// Unpack 4 bytes into uint32 to repack into base 85 5-byte.
		var v uint32
		switch len(src) {
		default:
			v |= uint32(src[3])
			fallthrough
		case 3:
			v |= uint32(src[2]) << 8
			fallthrough
		case 2:
			v |= uint32(src[1]) << 16
			fallthrough
		case 1:
			v |= uint32(src[0]) << 24
		}

		// Special case: zero (!!!!!) shortens to z.
		if v == 0 && len(src) >= 4 {
			dst[0] = 'z'
			dst = dst[1:]
			src = src[4:]
			n++
			continue
		}

		// Otherwise, 5 base 85 digits starting at !.
		for i := 4; i >= 0; i-- {
			dst[i] = '!' + byte(v%85)
			v /= 85
		}

		// If src was short, discard the low destination bytes.
		m := 5
		if len(src) < 4 {
			m -= 4 - len(src)
			src = nil
		} else {
			src = src[4:]
		}
		dst = dst[m:]
		n += m
	}
	return n
}

func (c *ascii85Codec) Encode(src []byte) ([]byte, error) {
	dst := make([]byte, c.maxEncodeLen(len(src)))
	n := c.encode(dst, src)
	return dst[:n], nil
}

func (c *ascii85Codec) decode(dst, src []byte, flush bool) (ndst, nsrc int, err error) {
	var v uint32
	var nb int
	for i, b := range src {
		if len(dst)-ndst < 4 {
			return
		}
		switch {
		case b <= ' ':
			continue
		case b == 'z' && nb == 0:
			nb = 5
			v = 0
		case '!' <= b && b <= 'u':
			v = v*85 + uint32(b-'!')
			nb++
		default:
			return 0, 0, errors.Errorf(invalieIllegalFormat, i)
		}
		if nb == 5 {
			nsrc = i + 1
			dst[ndst] = byte(v >> 24)
			dst[ndst+1] = byte(v >> 16)
			dst[ndst+2] = byte(v >> 8)
			dst[ndst+3] = byte(v)
			ndst += 4
			nb = 0
			v = 0
		}
	}
	if flush {
		nsrc = len(src)
		if nb > 0 {
			// The number of output bytes in the last fragment
			// is the number of leftover input bytes - 1:
			// the extra byte provides enough bits to cover
			// the inefficiency of the encoding for the block.
			if nb == 1 {
				return 0, 0, errors.Errorf(invalieIllegalFormat, len(src))
			}
			for i := nb; i < 5; i++ {
				// The short encoding truncated the output value.
				// We have to assume the worst case values (digit 84)
				// in order to ensure that the top bits are correct.
				v = v*85 + 84
			}
			for i := 0; i < nb-1; i++ {
				dst[ndst] = byte(v >> 24)
				v <<= 8
				ndst++
			}
		}
	}
	return
}

func (c *ascii85Codec) Decode(src []byte) ([]byte, error) {
	dst := make([]byte, len(src))
	n, _, err := c.decode(dst, src, true)
	if err != nil {
		return nil, err
	}
	return dst[:n], nil
}
