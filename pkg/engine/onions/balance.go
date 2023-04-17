package onions

import (
	"reflect"
	
	"git-indra.lan/indra-labs/lnd/lnd/lnwire"
	
	"git-indra.lan/indra-labs/indra/pkg/crypto/nonce"
	"git-indra.lan/indra-labs/indra/pkg/engine/coding"
	"git-indra.lan/indra-labs/indra/pkg/engine/magic"
	"git-indra.lan/indra-labs/indra/pkg/engine/sess"
	"git-indra.lan/indra-labs/indra/pkg/engine/sessions"
	"git-indra.lan/indra-labs/indra/pkg/splice"
	"git-indra.lan/indra-labs/indra/pkg/util/slice"
)

const (
	BalanceMagic = "ba"
	BalanceLen   = magic.Len + nonce.IDLen*2 + slice.Uint64Len
)

type Balance struct {
	ID     nonce.ID
	ConfID nonce.ID
	lnwire.MilliSatoshi
}

func balanceGen() coding.Codec           { return &Balance{} }
func init()                              { Register(BalanceMagic, balanceGen) }
func (x *Balance) Magic() string         { return BalanceMagic }
func (x *Balance) Len() int              { return BalanceLen }
func (x *Balance) Wrap(inner Onion)      {}
func (x *Balance) GetOnion() interface{} { return x }

func (x *Balance) Encode(s *splice.Splice) (e error) {
	log.T.S("encoding", reflect.TypeOf(x),
		x.ID, x.ConfID, x.MilliSatoshi,
	)
	s.
		Magic(BalanceMagic).
		ID(x.ID).
		ID(x.ConfID).
		Uint64(uint64(x.MilliSatoshi))
	return
}

func (x *Balance) Decode(s *splice.Splice) (e error) {
	if e = magic.TooShort(s.Remaining(), BalanceLen-magic.Len,
		BalanceMagic); fails(e) {
		return
	}
	s.
		ReadID(&x.ID).
		ReadID(&x.ConfID).
		ReadMilliSatoshi(&x.MilliSatoshi)
	return
}

func (x *Balance) Handle(s *splice.Splice, p Onion, ng Ngin) (e error) {
	
	if pending := ng.Pending().Find(x.ID); pending != nil {
		log.D.S("found pending", pending.ID)
		for i := range pending.Billable {
			session := ng.Mgr().FindSession(pending.Billable[i])
			out := session.Node.RelayRate * s.Len()
			if session != nil {
				in := session.Node.RelayRate * pending.SentSize
				switch {
				case i < 2:
					ng.Mgr().DecSession(session.ID, in, true, "reverse")
				case i == 2:
					ng.Mgr().DecSession(session.ID, (in+out)/2, true, "getbalance")
				case i > 2:
					ng.Mgr().DecSession(session.ID, out, true, "reverse")
				}
			}
		}
		var se *sessions.Data
		ng.Mgr().IterateSessions(func(s *sessions.Data) bool {
			if s.ID == x.ID {
				log.D.F("received balance %s for session %s %s was %s",
					x.MilliSatoshi, x.ID, x.ConfID, s.Remaining)
				se = s
				return true
			}
			return false
		})
		if se != nil {
			log.D.F("got %v, expected %v", se.Remaining, x.MilliSatoshi)
			se.Remaining = x.MilliSatoshi
		}
		ng.Pending().ProcessAndDelete(pending.ID, nil, s.GetAll())
	}
	return
}

func (x *Balance) Account(res *sess.Data, sm *sess.Manager,
	s *sessions.Data, last bool) (skip bool, sd *sessions.Data) {
	
	res.ID = x.ID
	return
}