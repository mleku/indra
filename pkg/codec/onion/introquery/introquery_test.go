package introquery

import (
	"github.com/indra-labs/indra"
	"github.com/indra-labs/indra/pkg/codec"
	"github.com/indra-labs/indra/pkg/codec/onion/end"
	"github.com/indra-labs/indra/pkg/codec/onion/exit"
	"github.com/indra-labs/indra/pkg/codec/ont"
	"github.com/indra-labs/indra/pkg/codec/reg"
	"testing"

	"github.com/indra-labs/indra/pkg/crypto"
	"github.com/indra-labs/indra/pkg/crypto/nonce"
	"github.com/indra-labs/indra/pkg/engine/sessions"
	log2 "github.com/indra-labs/indra/pkg/proc/log"
)

func TestOnions_IntroQuery(t *testing.T) {
	if indra.CI == "false" {
		log2.SetLogLevel(log2.Trace)
	}
	log2.App.Store("")
	var e error
	prvs, pubs := crypto.GetCipherSet()
	ciphers := crypto.GenCiphers(prvs, pubs)
	prv1, _ := crypto.GetTwoPrvKeys()
	pub1 := crypto.DerivePub(prv1)
	n3 := crypto.Gen3Nonces()
	ep := &exit.ExitPoint{
		Routing: &exit.Routing{
			Sessions: [3]*sessions.Data{},
			Keys:     prvs,
			Nonces:   n3,
		},
		ReturnPubs: pubs,
	}
	id := nonce.NewID()
	on := ont.Assemble([]ont.Onion{
		NewIntroQuery(id, crypto.DerivePub(prv1), ep),
		end.NewEnd(),
	})
	s := ont.Encode(on)
	s.SetCursor(0)
	var onc codec.Codec
	if onc = reg.Recognise(s); onc == nil {
		t.Error("did not unwrap")
		t.FailNow()
	}
	if e = onc.Decode(s); fails(e) {
		t.Error("did not decode")
		t.FailNow()
	}
	log.D.Ln(s)
	var ex *IntroQuery
	var ok bool
	if ex, ok = onc.(*IntroQuery); !ok {
		t.Error("did not unwrap expected type")
		t.FailNow()
	}
	for i := range ex.Ciphers {
		if ex.Ciphers[i] != ciphers[i] {
			t.Errorf("cipher %d did not unwrap correctly", i)
			t.FailNow()
		}
	}
	for i := range ex.Nonces {
		if ex.Nonces[i] != n3[i] {
			t.Errorf("nonce %d did not unwrap correctly", i)
			t.FailNow()
		}
	}
	if !ex.Key.Equals(pub1) {
		t.Error("HiddenService did not decode correctly")
		t.FailNow()
	}
	if ex.ID != id {
		t.Error("Keys did not decode correctly")
		t.FailNow()
	}
}