package onions

import (
	"net/netip"
	"reflect"
	
	"git-indra.lan/indra-labs/indra/pkg/engine/coding"
	"git-indra.lan/indra-labs/indra/pkg/engine/magic"
	"git-indra.lan/indra-labs/indra/pkg/engine/sess"
	"git-indra.lan/indra-labs/indra/pkg/engine/sessions"
	"git-indra.lan/indra-labs/indra/pkg/splice"
)

const (
	ForwardMagic = "fw"
	ForwardLen   = magic.Len + 1 + splice.AddrLen
)

type Forward struct {
	AddrPort *netip.AddrPort
	Onion
}

func forwardGen() coding.Codec           { return &Forward{} }
func init()                              { Register(ForwardMagic, forwardGen) }
func (x *Forward) Magic() string         { return ForwardMagic }
func (x *Forward) Len() int              { return ForwardLen + x.Onion.Len() }
func (x *Forward) Wrap(inner Onion)      { x.Onion = inner }
func (x *Forward) GetOnion() interface{} { return x }

func (x *Forward) Encode(s *splice.Splice) error {
	log.T.F("encoding %s %s", reflect.TypeOf(x),
		x.AddrPort.String(),
	)
	return x.Onion.Encode(s.Magic(ForwardMagic).AddrPort(x.AddrPort))
}

func (x *Forward) Decode(s *splice.Splice) (e error) {
	if e = magic.TooShort(s.Remaining(), ForwardLen-magic.Len,
		ForwardMagic); fails(e) {
		return
	}
	s.ReadAddrPort(&x.AddrPort)
	return
}

func (x *Forward) Handle(s *splice.Splice, p Onion, ng Ngin) (e error) {
	// Forward the whole buffer received onwards. Usually there will be a
	// crypt.Layer under this which will be unwrapped by the receiver.
	if x.AddrPort.String() == ng.Mgr().GetLocalNodeAddress().String() {
		// it is for us, we want to unwrap the next part.
		ng.HandleMessage(splice.BudgeUp(s), x)
	} else {
		switch on1 := p.(type) {
		case *Crypt:
			sess := ng.Mgr().FindSessionByHeader(on1.ToPriv)
			if sess != nil {
				ng.Mgr().DecSession(sess.ID,
					ng.Mgr().GetLocalNodeRelayRate()*s.Len(),
					false, "forward")
			}
		}
		// we need to forward this message onion.
		ng.Mgr().Send(x.AddrPort, splice.BudgeUp(s))
	}
	return e
}

func (x *Forward) Account(res *sess.Data, sm *sess.Manager,
	s *sessions.Data, last bool) (skip bool, sd *sessions.Data) {
	
	res.Billable = append(res.Billable, s.ID)
	res.PostAcct = append(res.PostAcct,
		func() {
			sm.DecSession(s.ID, s.Node.RelayRate*len(res.B),
				true, "forward")
		})
	return
}