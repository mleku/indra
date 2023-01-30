package relay

import (
	"git-indra.lan/indra-labs/lnd/lnd/lnwire"

	"git-indra.lan/indra-labs/indra/pkg/crypto/ciph"
	"git-indra.lan/indra-labs/indra/pkg/onion"
	"git-indra.lan/indra-labs/indra/pkg/onion/layers/crypt"
	"git-indra.lan/indra-labs/indra/pkg/onion/layers/reverse"
	"git-indra.lan/indra-labs/indra/pkg/types"
	"git-indra.lan/indra-labs/indra/pkg/util/slice"
)

func (en *Engine) reverse(on *reverse.Layer, b slice.Bytes,
	c *slice.Cursor, prev types.Onion) {

	var e error
	var on2 types.Onion
	if on.AddrPort.String() == en.Node.AddrPort.String() {
		if on2, e = onion.Peel(b, c); check(e) {
			return
		}
		switch on1 := on2.(type) {
		case *crypt.Layer:
			start := *c - crypt.ReverseLayerLen
			first := *c
			second := first + crypt.ReverseLayerLen
			last := second + crypt.ReverseLayerLen
			log.T.Ln("searching for reverse crypt keys")
			hdr, pld, _, _ := en.FindCloaked(on1.Cloak)
			if hdr == nil || pld == nil {
				log.E.F("failed to find key for %s",
					en.Node.AddrPort.String())
				return
			}
			// We need to find the PayloadPub to match.
			on1.ToPriv = hdr
			blk := ciph.GetBlock(on1.ToPriv, on1.FromPub)
			// Decrypt using the Payload key and header nonce.
			ciph.Encipher(blk, on1.Nonce,
				b[*c:c.Inc(2*crypt.ReverseLayerLen)])
			blk = ciph.GetBlock(pld, on1.FromPub)
			ciph.Encipher(blk, on1.Nonce, b[*c:])
			// shift the header segment upwards and pad the
			// remainder.
			copy(b[start:first], b[first:second])
			copy(b[first:second], b[second:last])
			copy(b[second:last], slice.NoisePad(crypt.ReverseLayerLen))
			if b[start:start+2].String() != reverse.MagicString {
				// It's for us!
				log.T.Ln("handling response")
				en.handleMessage(BudgeUp(b, last), on1)
				break
			}
			sess := en.FindSessionByHeader(hdr)
			if sess != nil {
				en.DecSession(sess.ID,
					en.RelayRate*lnwire.
						MilliSatoshi(len(b))/1024/1024, false, "reverse")
				en.handleMessage(BudgeUp(b, start), on1)
			}
		default:
			// If a reverse is not followed by an onion crypt the
			// message is incorrectly formed, just drop it.
			return
		}
	} else {
		// we need to forward this message onion.
		log.T.Ln("forwarding reverse")
		en.Send(on.AddrPort, b)
	}

}