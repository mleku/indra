package node

import (
	"github.com/indra-labs/indra/pkg/lnwire"
	"github.com/indra-labs/indra/pkg/nonce"
	"github.com/indra-labs/indra/pkg/sha256"
)

type Payment struct {
	nonce.ID
	Preimage sha256.Hash
	Amount   lnwire.MilliSatoshi
}

type PaymentChan chan *Payment

type PendingPayments []*Payment

func (p PendingPayments) Add(np *Payment) (pp PendingPayments) {
	return append(p, np)
}

func (p PendingPayments) Delete(preimage sha256.Hash) (pp PendingPayments) {
	pp = p
	for i := range p {
		if p[i].Preimage == preimage {
			if i == len(p)-1 {
				pp = p[:i]
			} else {
				pp = append(p[:i], p[i+1:]...)
			}
		}
	}
	return
}

func (p PendingPayments) Find(id nonce.ID) (pp *Payment) {
	for i := range p {
		if p[i].ID == id {
			return p[i]
		}
	}
	return
}
