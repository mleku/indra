package relay

import (
	"fmt"

	"github.com/davecgh/go-spew/spew"

	"git-indra.lan/indra-labs/indra/pkg/onion/layers/session"
	"git-indra.lan/indra-labs/indra/pkg/traffic"
	"git-indra.lan/indra-labs/indra/pkg/types"
	"git-indra.lan/indra-labs/indra/pkg/util/slice"
)

func (eng *Engine) session(on *session.Layer, b slice.Bytes,
	c *slice.Cursor, prev types.Onion) {

	log.T.C(func() string {
		return fmt.Sprint("incoming session",
			spew.Sdump(on.PreimageHash()))
	})
	pi := eng.FindPendingPreimage(on.PreimageHash())
	if pi != nil {
		// We need to delete this first in case somehow two such
		// messages arrive at the same time, and we end up with
		// duplicate sessions.
		eng.DeletePendingPayment(pi.Preimage)
		log.T.F("Adding session %x\n", pi.ID)
		eng.AddSession(traffic.NewSession(pi.ID,
			eng.Node, pi.Amount, on.Header, on.Payload, on.Hop))
		eng.handleMessage(BudgeUp(b, *c), on)
	} else {
		log.T.Ln("dropping session message without payment")
	}
}
