package ngin

import (
	"errors"
	"net/netip"
	
	"github.com/gookit/color"
	
	"git-indra.lan/indra-labs/indra/pkg/crypto/key/pub"
	"git-indra.lan/indra-labs/indra/pkg/crypto/nonce"
	"git-indra.lan/indra-labs/indra/pkg/util/slice"
)

type SendData struct {
	B         slice.Bytes
	Sessions  Sessions
	Billable  []nonce.ID
	Ret, Last nonce.ID
	Port      uint16
	PostAcct  []func()
}

// Send a message to a peer via their AddrPort.
func (sm *SessionManager) Send(addr *netip.AddrPort, s *Splice) {
	// first search if we already have the node available with connection open.
	as := addr.String()
	sm.ForEachNode(func(n *Node) bool {
		if as == n.AddrPort.String() {
			log.T.F("%s sending message to %v",
				sm.GetLocalNodeAddressString(), color.Yellow.Sprint(addr))
			n.Transport.Send(s.GetRange(-1, -1))
			return true
		}
		return false
	})
}

// SendWithOneHook is used for onions with only one confirmation hook. Usually
// as returned from PostAcctOnion this is the last, confirmation or response
// layer in an onion.Skins.
func (sm *SessionManager) SendWithOneHook(ap *netip.AddrPort,
	res SendData, responseHook Callback, p *PendingResponses) {
	
	if responseHook == nil {
		responseHook = func(_ nonce.ID, _ *pub.Bytes, _ slice.Bytes) (e error) {
			log.D.Ln("nil response hook")
			return errors.New("nil response hook")
		}
	}
	p.Add(ResponseParams{
		ID:       res.Last,
		SentSize: res.B.Len(),
		S:        res.Sessions,
		Billable: res.Billable,
		Ret:      res.Ret,
		Port:     res.Port,
		Callback: responseHook,
		PostAcct: res.PostAcct},
	)
	log.T.Ln(sm.GetLocalNodeAddressString(), "sending out onion", res.Last,
		"to", color.Yellow.Sprint(ap.String()))
	sm.Send(ap, Load(res.B, slice.NewCursor()))
}
