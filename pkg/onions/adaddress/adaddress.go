// Package adaddress defines the message format that provides the network multi-address of a peer with a given public identity key.
package adaddress

import (
	"fmt"
	"github.com/indra-labs/indra/pkg/crypto"
	"github.com/indra-labs/indra/pkg/crypto/nonce"
	"github.com/indra-labs/indra/pkg/crypto/sha256"
	"github.com/indra-labs/indra/pkg/engine/coding"
	"github.com/indra-labs/indra/pkg/engine/magic"
	"github.com/indra-labs/indra/pkg/onions/adproto"
	"github.com/indra-labs/indra/pkg/onions/reg"
	log2 "github.com/indra-labs/indra/pkg/proc/log"
	"github.com/indra-labs/indra/pkg/util/multi"
	"github.com/indra-labs/indra/pkg/util/splice"
	"github.com/multiformats/go-multiaddr"
	"net/netip"
	"reflect"
	"time"
)

var (
	log   = log2.GetLogger()
	fails = log.E.Chk
)

const (
	// Magic identifies an address ad.
	Magic = "adad"

	// Len is the number of bytes in an address ad.
	Len = adproto.Len + splice.AddrLen + 1
)

// Ad entries are stored with an index generated by concatenating the bytes
// of the public key with a string path "/address/N" where N is the index of the
// address. This means hidden service introducers for values over zero.
// Hidden services have no value in the zero index, which is "<hash>/address/0".
type Ad struct {
	adproto.Ad
	// Addrs are the addresses listed as contact points for the relay.
	Addrs multiaddr.Multiaddr
}

// Decode an address ad from an identified buffer with the Magic prefix.
func (x *Ad) Decode(s *splice.Splice) (e error) {
	if e = magic.TooShort(s.Remaining(), Len-magic.Len,
		Magic); fails(e) {

		return
	}
	addr := &netip.AddrPort{}
	s.ReadID(&x.ID).
		ReadPubkey(&x.Key).
		ReadAddrPort(&addr).
		ReadTime(&x.Expiry).
		ReadSignature(&x.Sig)
	var ap multiaddr.Multiaddr
	proto := "ip4"
	if addr.Addr().Is6() {
		proto = "ip6"
	}
	if ap, e = multiaddr.NewMultiaddr(
		"/" + proto + "/" + addr.Addr().String() +
			"/tcp/" + fmt.Sprint(addr.Port()),
	); fails(e) {
		return
	}
	x.Addrs = multi.AddKeyToMultiaddr(ap, x.Key)
	return
}

// Encode an address ad into a splice.
func (x *Ad) Encode(s *splice.Splice) (e error) {
	log.T.S("encoding", reflect.TypeOf(x), x)
	x.Splice(s)
	return
}

// GetOnion returns the onion inside. This isn't an onion so there is no onion to return.
func (x *Ad) GetOnion() interface{} { return nil }

// Len returns the length of an address ad in bytes.
func (x *Ad) Len() int { return Len }

// Magic provides the identifying 4 byte prefix of an address ad.
func (x *Ad) Magic() string { return Magic }

// Splice assembles an address ad into an encoded message.
func (x *Ad) Splice(s *splice.Splice) {
	x.SpliceNoSig(s)
	s.Signature(x.Sig)
}

// SpliceNoSig assembles the message but leaves the signature to be populated and serialized.
func (x *Ad) SpliceNoSig(s *splice.Splice) {
	var e error
	var ap netip.AddrPort
	if ap, e = multi.AddrToAddrPort(x.Addrs); fails(e) {
		return
	}
	s.Magic(Magic).
		ID(x.ID).
		Pubkey(x.Key).
		AddrPort(&ap).
		Time(x.Expiry)

}

// Validate proves that the signature matches the public key.
//
// TODO: could we use an "address" form where the ads contain a shorter hash (eg
//
//	ripemd) to reduce the payload size here?
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

// New generates a new address ad and signs it with the provided private key.
//
// TODO: create a schnorr based signature that recovers the pubkey so the address in the adproto can be shorter.
func New(id nonce.ID, key *crypto.Prv,
	ma multiaddr.Multiaddr, expiry time.Time) (peerAd *Ad) {

	pub := crypto.DerivePub(key)
	ma = multi.AddKeyToMultiaddr(ma, pub)
	peerAd = &Ad{
		Ad: adproto.Ad{
			ID:     id,
			Key:    crypto.DerivePub(key),
			Expiry: expiry,
		},
		Addrs: ma,
	}
	s := splice.New(Len)
	peerAd.SpliceNoSig(s)
	hash := sha256.Single(s.GetUntil(s.GetCursor()))
	var e error
	if peerAd.Sig, e = crypto.Sign(key, hash); fails(e) {
		return nil
	}
	return
}

// addrGen is the factory function for this message type.
func addrGen() coding.Codec { return &Ad{} }

// Add this magic and generator to the registry.
func init() { reg.Register(Magic, addrGen) }
