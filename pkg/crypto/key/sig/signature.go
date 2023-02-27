// Package sig provides functions to Sign hashes of messages, generating a
// standard compact 65 byte Signature and recover the 33 byte pub.Key embedded
// in it. This is used as a MAC for Indra packets to associate messages with
// Indra peers' sessions.
package sig

import (
	"github.com/decred/dcrd/dcrec/secp256k1/v4"
	"github.com/decred/dcrd/dcrec/secp256k1/v4/ecdsa"
	
	"git-indra.lan/indra-labs/indra"
	"git-indra.lan/indra-labs/indra/pkg/crypto/key/prv"
	"git-indra.lan/indra-labs/indra/pkg/crypto/key/pub"
	"git-indra.lan/indra-labs/indra/pkg/crypto/sha256"
	log2 "git-indra.lan/indra-labs/indra/pkg/proc/log"
)

var (
	log   = log2.GetLogger(indra.PathBase)
	check = log.E.Chk
)

// Len is the length of the signatures used in Indra, compact keys that can have
// the public key extracted from them.
const Len = 65

// Bytes is an ECDSA BIP62 formatted compact signature which allows the recovery
// of the public key from the signature.
type Bytes [Len]byte

// Sign produces an ECDSA BIP62 compact signature.
func Sign(prv *prv.Key, hash sha256.Hash) (sig Bytes, e error) {
	copy(sig[:],
		ecdsa.SignCompact((*secp256k1.PrivateKey)(prv), hash[:], true))
	return
}

// Recover the public key corresponding to the signing private key used to
// create a signature on the hash of a message.
func (sig Bytes) Recover(hash sha256.Hash) (p *pub.Key, e error) {
	var pk *secp256k1.PublicKey
	// We are only using compressed keys, so we can ignore the compressed
	// bool.
	if pk, _, e = ecdsa.RecoverCompact(sig[:], hash[:]); !check(e) {
		p = (*pub.Key)(pk)
	}
	return
}
