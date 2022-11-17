package delay

import (
	"time"

	"github.com/indra-labs/indra"
	"github.com/indra-labs/indra/pkg/onion/layers/magicbytes"
	log2 "github.com/indra-labs/indra/pkg/proc/log"
	"github.com/indra-labs/indra/pkg/types"
	"github.com/indra-labs/indra/pkg/util/slice"
)

const (
	MagicString = "dl"
	Len         = magicbytes.Len + slice.Uint64Len
)

var (
	log               = log2.GetLogger(indra.PathBase)
	check             = log.E.Chk
	Magic             = slice.Bytes(MagicString)
	_     types.Onion = &Layer{}
)

// A Layer delay is a message to hold for a period of time before relaying.
type Layer struct {
	time.Duration
	types.Onion
}

func (x *Layer) Inner() types.Onion   { return nil }
func (x *Layer) Insert(_ types.Onion) {}
func (x *Layer) Len() int             { return Len }

func (x *Layer) Encode(b slice.Bytes, c *slice.Cursor) {
	copy(b[*c:c.Inc(magicbytes.Len)], Magic)
	slice.EncodeUint64(b[*c:c.Inc(slice.Uint64Len)], uint64(x.Duration))
	x.Onion.Encode(b, c)
}

func (x *Layer) Decode(b slice.Bytes, c *slice.Cursor) (e error) {
	if len(b[*c:]) < Len-magicbytes.Len {
		return magicbytes.TooShort(len(b[*c:]), Len-magicbytes.Len,
			string(Magic))
	}
	x.Duration = time.Duration(
		slice.DecodeUint64(b[*c:c.Inc(slice.Uint64Len)]))
	return
}