package engine

import (
	"context"
	"github.com/indra-labs/indra/pkg/crypto"
	"github.com/indra-labs/indra/pkg/crypto/nonce"
	"github.com/indra-labs/indra/pkg/engine/node"
	"github.com/indra-labs/indra/pkg/engine/responses"
	"github.com/indra-labs/indra/pkg/engine/sess"
	"github.com/indra-labs/indra/pkg/engine/sessions"
	"github.com/indra-labs/indra/pkg/engine/tpt"
	"github.com/indra-labs/indra/pkg/engine/transport"
	"github.com/indra-labs/indra/pkg/onions/hidden"
	"github.com/indra-labs/indra/pkg/onions/ont"
	"github.com/indra-labs/indra/pkg/onions/reg"
	"github.com/indra-labs/indra/pkg/util/qu"
	"github.com/indra-labs/indra/pkg/util/slice"
	"github.com/indra-labs/indra/pkg/util/splice"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"go.uber.org/atomic"
)

const (
	PubSubTopic = "indra"
)

var _ ont.Ngin = &Engine{}

type (
	// Engine processes onion messages, forwarding the relevant data to other relays
	// and locally accessible servers as indicated by the API function and message
	// parameters.
	Engine struct {
		ctx          context.Context
		cancel       func()
		Responses    *responses.Pending
		Manager      *sess.Manager
		Listener     *transport.Listener
		PubSub       *pubsub.PubSub
		topic        *pubsub.Topic
		sub          *pubsub.Subscription
		h            *hidden.Hidden
		KeySet       *crypto.KeySet
		Load         atomic.Uint32
		Pause        qu.C
		ShuttingDown atomic.Bool
	}
	Params struct {
		Transport tpt.Transport
		Listener  *transport.Listener
		*crypto.Keys
		Node            *node.Node
		Nodes           []*node.Node
		NReturnSessions int
	}
)

// Cleanup closes and flushes any resources the client opened that require sync
// in order to reopen correctly.
func (ng *Engine) Cleanup() {
	// Do cleanup stuff before shutdown.
}

func (ng *Engine) GetHidden() *hidden.Hidden { return ng.h }

func (ng *Engine) GetLoad() byte { return byte(ng.Load.Load()) }

func (ng *Engine) HandleMessage(s *splice.Splice, pr ont.Onion) {
	log.D.F("%s handling received message",
		ng.Manager.GetLocalNodeAddressString())
	s.SetCursor(0)
	s.Segments = s.Segments[:0]
	on := reg.Recognise(s)
	if on != nil {
		log.D.Ln("magic", on.Magic())
		if fails(on.Decode(s)) {
			return
		}
		if pr != nil && on.Magic() != pr.Magic() {
			log.D.S("",s.GetAll().ToBytes())
		}
		m := on.GetOnion()
		if m == nil {
			log.D.Ln("did not get onion")
			return
		}
		if fails(m.(ont.Onion).Handle(s, pr, ng)) {
			log.W.S("unrecognised packet", s.GetAll().ToBytes())
		}
	}
}

func (ng *Engine) Handler() (out bool) {
	log.T.C(func() string {
		return ng.Manager.GetLocalNodeAddressString() + " awaiting message"
	})
	var prev ont.Onion
	select {
	case <-ng.ctx.Done():
		ng.Shutdown()
		out = true
		break
	case c := <-ng.Listener.Accept():
		go func() {
			log.D.Ln("new connection inbound (TODO):", c.Host.Addrs()[0])
			_ = c
		}()
	case b := <-ng.Manager.ReceiveToLocalNode():
		s := splice.Load(b, slice.NewCursor())
		ng.HandleMessage(s, prev)
	case p := <-ng.Manager.GetLocalNode().Chan.Receive():
		log.D.F("incoming payment for %s: %v", p.ID, p.Amount)
		topUp := false
		ng.Manager.IterateSessions(func(s *sessions.Data) bool {
			if s.Preimage == p.Preimage {
				s.IncSats(p.Amount, false, "top-up")
				topUp = true
				log.T.F("topping up %x with %v", s.Header.Bytes, p.Amount)
				return true
			}
			return false
		})
		if !topUp {
			ng.Manager.AddPendingPayment(p)
			log.T.F("awaiting session keys for preimage %s session Keys %s",
				p.Preimage, p.ID)
		}
		// For now if we received this we return true. Later this will wait with
		// a timeout on the lnd node returning the success to trigger this.
		p.ConfirmChan <- true
	case <-ng.Pause:
		log.D.Ln("pausing", ng.Manager.GetLocalNodeAddressString())
		// For testing purposes we need to halt this Handler and discard channel
		// messages.
	out:
		for {
			select {
			case <-ng.Manager.GetLocalNode().Chan.Receive():
				log.D.Ln("discarding payments while in pause")
			case <-ng.Manager.ReceiveToLocalNode():
				log.D.Ln("discarding messages while in pause")
			case <-ng.ctx.Done():
				break out
			case <-ng.Pause:
				// This will then resume to the top level select.
				log.D.Ln("unpausing", ng.Manager.GetLocalNodeAddressString())
				break out
			}
		}
	}
	return
}

func (ng *Engine) Keyset() *crypto.KeySet      { return ng.KeySet }
func (ng *Engine) KillSwitch() <-chan struct{} { return ng.ctx.Done() }
func (ng *Engine) Mgr() *sess.Manager          { return ng.Manager }
func (ng *Engine) Pending() *responses.Pending { return ng.Responses }
func (ng *Engine) SetLoad(load byte)           { ng.Load.Store(uint32(load)) }

// Shutdown triggers the shutdown of the client and the Cleanup before
// finishing.
func (ng *Engine) Shutdown() {
	log.T.Ln("shutting down", ng.Manager.GetLocalNodeAddress().String())
	if ng.ShuttingDown.Load() {
		return
	}
	ng.ShuttingDown.Store(true)
	ng.Cleanup()
	ng.cancel()
}

// Start a single thread of the Engine.
func (ng *Engine) Start() {
	log.T.Ln("starting engine")
	ng.RunAdHandler(ng.HandleAd)
	for {
		if ng.Handler() {
			break
		}
	}
}

func New(p Params) (c *Engine, e error) {
	p.Node.Transport = p.Transport
	p.Node.Identity = p.Keys
	var ks *crypto.KeySet
	if _, ks, e = crypto.NewSigner(); fails(e) {
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	c = &Engine{
		ctx:       ctx,
		cancel:    cancel,
		Responses: &responses.Pending{},
		KeySet:    ks,
		Listener:  p.Listener,
		Manager:   sess.NewSessionManager(),
		h:         hidden.NewHiddenrouting(),
		Pause:     qu.T(),
	}
	if p.Listener.Host != nil {
		if c.PubSub, e = pubsub.NewGossipSub(ctx, p.Listener.Host); fails(e) {
			cancel()
			return
		}
		if c.topic, e = c.PubSub.Join(PubSubTopic); fails(e) {
			return
		}
		if c.sub, e = c.topic.Subscribe(); fails(e) {
			return
		}
		log.D.Ln("subscribed to", PubSubTopic, "topic on gossip network")
	}
	c.Manager.AddNodes(append([]*node.Node{p.Node}, p.Nodes...)...)
	// AddIntro a return session for receiving responses, ideally more of these
	// will be generated during operation and rotated out over time.
	for i := 0; i < p.NReturnSessions; i++ {
		c.Manager.AddSession(sessions.NewSessionData(nonce.NewID(), p.Node, 0,
			nil, nil, 5))
	}
	return
}
