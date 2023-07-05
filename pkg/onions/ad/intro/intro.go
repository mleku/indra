// Package intro defines a message type that provides information about an introduction point for a hidden service.
package intro

import (
	"github.com/indra-labs/indra/pkg/onions/ad"
	"github.com/indra-labs/indra/pkg/onions/reg"
	log2 "github.com/indra-labs/indra/pkg/proc/log"
	"reflect"
	"time"

	"github.com/indra-labs/indra/pkg/crypto"
	"github.com/indra-labs/indra/pkg/crypto/nonce"
	"github.com/indra-labs/indra/pkg/crypto/sha256"
	"github.com/indra-labs/indra/pkg/engine/coding"
	"github.com/indra-labs/indra/pkg/engine/magic"
	"github.com/indra-labs/indra/pkg/util/slice"
	"github.com/indra-labs/indra/pkg/util/splice"
)

var (
	log   = log2.GetLogger()
	fails = log.E.Chk
)

const (
	Magic = "inad"
	Len   = ad.Len + crypto.PubKeyLen + slice.Uint16Len + slice.Uint32Len
)

// Ad is an Intro message that signals that a hidden service can be accessed from
// a given relay identifiable by its public key.
type Ad struct {

	// Embed ad.Ad for the common fields
	ad.Ad

	// Introducer is the key of the node that can forward a Route message to help
	// establish a connection to a hidden service.
	Introducer *crypto.Pub
	// Port is the well known port of protocol available.
	Port uint16

	// Rate for accessing the hidden service (covers the hidden service routing
	// header relaying).
	RelayRate uint32
}

var _ coding.Codec = &Ad{}

// Decode an Ad out of the next bytes of a splice.Splice.
func (x *Ad) Decode(s *splice.Splice) (e error) {
	if e = magic.TooShort(s.Remaining(), Len-magic.Len,
		Magic); fails(e) {

		return
	}
	s.
		ReadID(&x.ID).
		ReadPubkey(&x.Key).
		ReadPubkey(&x.Introducer).
		ReadUint32(&x.RelayRate).
		ReadUint16(&x.Port).
		ReadTime(&x.Expiry).
		ReadSignature(&x.Sig)
	return
}

// Encode an Ad into the next bytes of a splice.Splice.
func (x *Ad) Encode(s *splice.Splice) (e error) {
	log.T.S("encoding", reflect.TypeOf(x), x)
	x.Splice(s)
	return
}

// GetOnion returns nil because there is no onion inside.
func (x *Ad) GetOnion() interface{} { return nil }

// Len returns the length of the binary encoded Ad.
func (x *Ad) Len() int { return Len }

// Magic is the identifier indicating an Ad is encoded in the following bytes.
func (x *Ad) Magic() string { return Magic }

// Splice serializes an Ad into a splice.Splice.
func (x *Ad) Splice(s *splice.Splice) {
	x.SpliceNoSig(s)
	s.Signature(x.Sig)
}

// SpliceNoSig serializes the Ad but stops at the signature.
func (x *Ad) SpliceNoSig(s *splice.Splice) {
	IntroSplice(s, x.ID, x.Key, x.Introducer, x.RelayRate, x.Port, x.Expiry)
}

// Validate checks the signature matches the public key of the Ad.
func (x *Ad) Validate() bool {
	s := splice.New(Len - magic.Len)
	x.SpliceNoSig(s)
	hash := sha256.Single(s.GetUntil(s.GetCursor()))
	key, e := x.Sig.Recover(hash)
	if fails(e) {
		return false
	}
	if key.Equals(x.Key) && x.Expiry.After(time.Now()) {
		return true
	}
	return false
}

// IntroSplice creates the message part up to the signature for an Ad.
func IntroSplice(
	s *splice.Splice,
	id nonce.ID,
	key *crypto.Pub,
	introducer *crypto.Pub,
	relayRate uint32,
	port uint16,
	expires time.Time,
) {

	s.Magic(Magic).
		ID(id).
		Pubkey(key).
		Pubkey(introducer).
		Uint32(relayRate).
		Uint16(port).
		Time(expires)
}

// New creates a new Ad and signs it.
func New(
	id nonce.ID,
	key *crypto.Prv,
	introducer *crypto.Pub,
	relayRate uint32,
	port uint16,
	expires time.Time,
) (in *Ad) {

	pk := crypto.DerivePub(key)

	in = &Ad{
		Ad: ad.Ad{
			ID:     id,
			Key:    pk,
			Expiry: expires,
		},
		Introducer: introducer,
		RelayRate:  relayRate,
		Port:       port,
	}
	s := splice.New(in.Len())
	in.SpliceNoSig(s)
	hash := sha256.Single(s.GetUntil(s.GetCursor()))
	var e error
	if in.Sig, e = crypto.Sign(key, hash); fails(e) {
		return nil
	}
	return
}

func init() { reg.Register(Magic, Gen) }

// Gen is a factory function for an Ad.
func Gen() coding.Codec { return &Ad{} }
