package hiddenservice

import (
	"git-indra.lan/indra-labs/indra"
	"git-indra.lan/indra-labs/indra/pkg/crypto/nonce"
	"git-indra.lan/indra-labs/indra/pkg/crypto/sha256"
	log2 "git-indra.lan/indra-labs/indra/pkg/proc/log"
	"git-indra.lan/indra-labs/indra/pkg/relay/messages/intro"
	"git-indra.lan/indra-labs/indra/pkg/relay/messages/magicbytes"
	"git-indra.lan/indra-labs/indra/pkg/relay/types"
	"git-indra.lan/indra-labs/indra/pkg/util/slice"
	"git-indra.lan/indra-labs/indra/pkg/util/splice"
)

const (
	MagicString = "hs"
	Len         = magicbytes.Len + nonce.IDLen + intro.Len +
		3*sha256.Len + nonce.IVLen*3
)

var (
	log               = log2.GetLogger(indra.PathBase)
	check             = log.E.Chk
	Magic             = slice.Bytes(MagicString)
	_     types.Onion = &Layer{}
)

// Layer hiddenservice messages deliver an introduction message which they
// advertise they have received and when requested can then request a routing
// header for any client that requests it.
type Layer struct {
	nonce.ID
	intro.Layer
	// Ciphers is a set of 3 symmetric ciphers that are to be used in their
	// given order over the reply message from the service.
	Ciphers [3]sha256.Hash
	// Nonces are the nonces to use with the cipher when creating the
	// encryption for the reply message.
	Nonces [3]nonce.IV
}

// func (x *Layer) String() string {
// 	return spew.Sdump(x.Port, x.Ciphers, x.Nonces, x.Bytes.ToBytes())
// }

func (x *Layer) Insert(o types.Onion) {}
func (x *Layer) Len() int {
	return Len
}

func (x *Layer) Encode(b slice.Bytes, c *slice.Cursor) {
	splice.Splice(b, c).
		Magic(Magic).
		ID(x.ID).
		Pubkey(x.Key).
		AddrPort(x.AddrPort).
		Signature(x.Sig).
		Hash(x.Ciphers[0]).Hash(x.Ciphers[1]).Hash(x.Ciphers[2]).
		IV(x.Nonces[0]).IV(x.Nonces[1]).IV(x.Nonces[2])
}

func (x *Layer) Decode(b slice.Bytes, c *slice.Cursor) (e error) {
	if len(b[*c:]) < Len-magicbytes.Len {
		return magicbytes.TooShort(len(b[*c:]), Len-magicbytes.Len, string(Magic))
	}
	splice.Splice(b, c).
		ReadID(&x.ID).
		ReadPubkey(&x.Layer.Key).
		ReadAddrPort(&x.Layer.AddrPort).
		ReadSignature(&x.Layer.Sig).
		ReadHash(&x.Ciphers[0]).ReadHash(&x.Ciphers[1]).ReadHash(&x.Ciphers[2]).
		ReadIV(&x.Nonces[0]).ReadIV(&x.Nonces[1]).ReadIV(&x.Nonces[2])
	return
}