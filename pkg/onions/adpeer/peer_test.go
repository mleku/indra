package adpeer

import (
	"fmt"
	"github.com/indra-labs/indra"
	"github.com/indra-labs/indra/pkg/crypto"
	"github.com/indra-labs/indra/pkg/crypto/nonce"
	"github.com/indra-labs/indra/pkg/engine/coding"
	"github.com/indra-labs/indra/pkg/onions/reg"
	log2 "github.com/indra-labs/indra/pkg/proc/log"
	"github.com/indra-labs/indra/pkg/util/splice"
	"testing"
)

func TestPeerAd(t *testing.T) {
	if indra.CI != "false" {
		log2.SetLogLevel(log2.Trace)
		fmt.Println("logging")
	}
	var e error
	pr, ks, _ := crypto.NewSigner()
	id := nonce.NewID()
	// in := New(id, pr, time.Now().Add(time.Hour))
	var prvs crypto.Privs
	for i := range prvs {
		prvs[i] = ks.Next()
	}
	var pubs crypto.Pubs
	for i := range pubs {
		pubs[i] = crypto.DerivePub(prvs[i])
	}
	pa := New(id, pr, 20000)
	s := splice.New(pa.Len())
	if e = pa.Encode(s); fails(e) {
		t.Fatalf("did not encode")
	}
	log.D.S(s.GetAll().ToBytes())
	s.SetCursor(0)
	var onc coding.Codec
	if onc = reg.Recognise(s); onc == nil {
		t.Fatalf("did not unwrap")
	}
	if e = onc.Decode(s); fails(e) {
		t.Fatalf("did not decode")
	}
	log.D.S(onc)
	var peer *Ad
	var ok bool
	if peer, ok = onc.(*Ad); !ok {
		t.Fatal("did not unwrap expected type")
	}
	if peer.ID != pa.ID {
		t.Errorf("ID did not decode correctly")
		t.FailNow()
	}
	if !peer.Key.Equals(crypto.DerivePub(pr)) {
		t.Errorf("public key did not decode correctly")
		t.FailNow()
	}
	if !peer.Validate() {
		t.Fatalf("received Ad did not validate")
	}
}