package route

import (
	"crypto/cipher"
	"github.com/indra-labs/indra"
	"github.com/indra-labs/indra/pkg/crypto"
	"github.com/indra-labs/indra/pkg/crypto/ciph"
	"github.com/indra-labs/indra/pkg/crypto/nonce"
	"github.com/indra-labs/indra/pkg/crypto/sha256"
	"github.com/indra-labs/indra/pkg/engine/coding"
	"github.com/indra-labs/indra/pkg/engine/magic"
	"github.com/indra-labs/indra/pkg/engine/sess"
	"github.com/indra-labs/indra/pkg/engine/sessions"
	"github.com/indra-labs/indra/pkg/onions/crypt"
	"github.com/indra-labs/indra/pkg/onions/end"
	"github.com/indra-labs/indra/pkg/onions/exit"
	"github.com/indra-labs/indra/pkg/onions/forward"
	"github.com/indra-labs/indra/pkg/onions/hidden"
	"github.com/indra-labs/indra/pkg/onions/ont"
	"github.com/indra-labs/indra/pkg/onions/ready"
	"github.com/indra-labs/indra/pkg/onions/reg"
	"github.com/indra-labs/indra/pkg/onions/reverse"
	log2 "github.com/indra-labs/indra/pkg/proc/log"
	"github.com/indra-labs/indra/pkg/util/splice"
	"reflect"
)

var (
	log   = log2.GetLogger(indra.PathBase)
	fails = log.E.Chk
)

const (
	RouteMagic = "rout"
	RouteLen   = magic.Len +
		crypto.CloakLen +
		crypto.PubKeyLen +
		nonce.IVLen +
		nonce.IDLen +
		3*sha256.Len +
		3*nonce.IVLen
)

type Route struct {
	HiddenService *crypto.Pub
	HiddenCloaked crypto.CloakedPubKey
	Sender        *crypto.Prv
	SenderPub     *crypto.Pub
	nonce.IV
	// ------- the rest is encrypted to the HiddenService/Sender keys.
	ID nonce.ID
	// Ciphers is a set of 3 symmetric ciphers that are to be used in their
	// given order over the reply message from the service.
	Ciphers crypto.Ciphers
	// Nonces are the nonces to use with the cipher when creating the
	// encryption for the reply message,
	// they are common with the crypts in the header.
	crypto.Nonces
	hidden.RoutingHeaderBytes
	ont.Onion
}

func NewRoute(id nonce.ID, k *crypto.Pub, ks *crypto.KeySet,
	ep *exit.ExitPoint) ont.Onion {
	oo := &Route{
		HiddenService: k,
		Sender:        ks.Next(),
		IV:            nonce.New(),
		ID:            id,
		Ciphers:       crypto.GenCiphers(ep.Keys, ep.ReturnPubs),
		Nonces:        ep.Nonces,
		Onion:         &end.End{},
	}
	oo.SenderPub = crypto.DerivePub(oo.Sender)
	oo.HiddenCloaked = crypto.GetCloak(k)
	return oo
}

func (x *Route) Account(res *sess.Data, sm *sess.Manager,
	s *sessions.Data, last bool) (skip bool, sd *sessions.Data) {
	copy(res.ID[:], x.ID[:])
	res.Billable = append(res.Billable, s.Header.Bytes)
	return
}

func (x *Route) Decode(s *splice.Splice) (e error) {
	if e = magic.TooShort(s.Remaining(), RouteLen-magic.Len,
		RouteMagic); fails(e) {
		return
	}
	s.ReadCloak(&x.HiddenCloaked).
		ReadPubkey(&x.SenderPub).
		ReadIV(&x.IV)
	return
}

// Decrypt decrypts the rest of a message after the Route segment if the
// recipient has the hidden service private key.
func (x *Route) Decrypt(prk *crypto.Prv, s *splice.Splice) {
	ciph.Encipher(ciph.GetBlock(prk, x.SenderPub, "route decrypt"), x.IV,
		s.GetRest())
	// And now we can see the reply field for the return trip.
	s.ReadID(&x.ID).ReadCiphers(&x.Ciphers).ReadNonces(&x.Nonces)
	hidden.ReadRoutingHeader(s, &x.RoutingHeaderBytes)
}

func (x *Route) Encode(s *splice.Splice) (e error) {
	log.T.S("encoding", reflect.TypeOf(x),
		x.ID, x.HiddenService, x.Sender, x.IV, x.Ciphers, x.Nonces,
		x.RoutingHeaderBytes,
	)
	s.Magic(RouteMagic).
		Cloak(x.HiddenService).
		Pubkey(crypto.DerivePub(x.Sender)).
		IV(x.IV)
	start := s.GetCursor()
	s.ID(x.ID).Ciphers(x.Ciphers).Nonces(x.Nonces)
	if e = x.Onion.Encode(s); fails(e) {
		return
	}
	var blk cipher.Block
	// Encrypt the message!
	if blk = ciph.GetBlock(x.Sender, x.HiddenService, "route"); fails(e) {
		return
	}
	ciph.Encipher(blk, x.IV, s.GetFrom(start))
	return
}

func RouteGen() coding.Codec           { return &Route{} }
func (x *Route) GetOnion() interface{} { return x }

func (x *Route) Handle(s *splice.Splice, p ont.Onion, ng ont.Ngin) (e error) {
	log.D.Ln(ng.Mgr().GetLocalNodeAddressString(), "handling route")
	hc := ng.GetHidden().FindCloakedHiddenService(x.HiddenCloaked)
	if hc == nil {
		log.T.Ln("no matching hidden service key found from cloaked key")
		return
	}
	if x.HiddenService, e = crypto.PubFromBytes((*hc)[:]); fails(e) {
		return
	}
	log.D.Ln("route key", *hc)
	hcl := *hc
	if hh, ok := ng.GetHidden().Services[hcl]; ok {
		log.D.F("we are the hidden service %s - decrypting...",
			hh.CurrentIntros[0].Key.ToBased32Abbreviated())
		// We have the keys to unwrap this one.
		x.Decrypt(hh.Prv, s)
		log.D.Ln(s)
		n := crypto.GenNonces(5)
		rvKeys := ng.Keyset().Next3()
		hops := []byte{3, 4, 5, 0, 1}
		ss := make(sessions.Sessions, len(hops))
		ng.Mgr().SelectHops(hops, ss, "route reply header")
		rt := &exit.Routing{
			Sessions: [3]*sessions.Data{ss[0], ss[1], ss[2]},
			Keys:     crypto.Privs{rvKeys[0], rvKeys[1], rvKeys[2]},
			Nonces:   crypto.Nonces{n[0], n[1], n[2]},
		}
		rh := []ont.Onion{
			reverse.NewReverse(rt.Sessions[0].Node.AddrPort),
			crypt.NewCrypt(rt.Sessions[0].Header.Pub, rt.Sessions[0].Payload.Pub, rt.Keys[0],rt.Nonces[0], 3),
			reverse.NewReverse(rt.Sessions[1].Node.AddrPort),
			crypt.NewCrypt(rt.Sessions[1].Header.Pub, rt.Sessions[1].Payload.Pub, rt.Keys[1],rt.Nonces[1], 2),
			reverse.NewReverse(rt.Sessions[2].Node.AddrPort),
			crypt.NewCrypt(rt.Sessions[2].Header.Pub, rt.Sessions[2].Payload.Pub, rt.Keys[2],rt.Nonces[2], 1),
		}
		//.RoutingHeader(rt)
		rHdr := ont.Encode(ont.Assemble(rh))
		rHdr.SetCursor(0)
		ep := exit.ExitPoint{
			Routing: rt,
			ReturnPubs: crypto.Pubs{
				crypto.DerivePub(ss[0].Payload.Prv),
				crypto.DerivePub(ss[1].Payload.Prv),
				crypto.DerivePub(ss[2].Payload.Prv),
			},
		}
		mr := []ont.Onion{
			forward.NewForward(ss[3].Node.AddrPort),
			crypt.NewCrypt(ss[3].Header.Pub, ss[3].Payload.Pub, ng.Keyset().Next(), n[3], 0),
			forward.NewForward(ss[4].Node.AddrPort),
			crypt.NewCrypt(ss[4].Header.Pub, ss[4].Payload.Pub, ng.Keyset().Next(), n[4], 0),
			ready.NewReady(x.ID, x.HiddenService,
				x.RoutingHeaderBytes,
				hidden.GetRoutingHeaderFromCursor(rHdr),
				x.Ciphers,
				crypto.GenCiphers(ep.Keys, ep.ReturnPubs),
				x.Nonces,
				ep.Nonces),
		}
		assembled := ont.Assemble(mr)
		reply := ont.Encode(assembled)
		ng.HandleMessage(reply, x)
	}
	return
}

func (x *Route) Len() int             { return RouteLen + x.Onion.Len() }
func (x *Route) Magic() string        { return RouteMagic }
func (x *Route) Wrap(inner ont.Onion) { x.Onion = inner }
func init()                           { reg.Register(RouteMagic, RouteGen) }