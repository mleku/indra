package wire

import (
	"github.com/Indra-Labs/indra/pkg/key/address"
	"github.com/Indra-Labs/indra/pkg/key/ecdh"
	"github.com/Indra-Labs/indra/pkg/key/prv"
	"github.com/Indra-Labs/indra/pkg/key/signer"
	"github.com/Indra-Labs/indra/pkg/node"
	"github.com/Indra-Labs/indra/pkg/nonce"
	"github.com/Indra-Labs/indra/pkg/sha256"
	"github.com/Indra-Labs/indra/pkg/slice"
)

// Ping is a message which checks the liveness of relays by ensuring they are
// correctly relaying messages. Pending pings are stored in a table with the
// last hop as the key to narrow the number of elements to search through to
// find the matching cipher and reveal the contained ID inside it.
//
// The pending ping records keep the identifiers of the three nodes that were in
// a ping onion and when the Confirmation is correctly received these nodes get
// an increment of their liveness score. By using this scheme, when nodes are
// offline their scores will fall to zero after a time whereas live nodes will
// have steadily increasing scores from successful pings.
func Ping(id nonce.ID, client node.Node, hop [3]node.Node,
	set signer.KeySet) Onion {

	return OnionSkins{}.
		Header(address.FromPubKey(hop[0].HeaderKey), set.Next()).
		Forward(hop[1].IP).
		Header(address.FromPubKey(hop[1].HeaderKey), set.Next()).
		Forward(hop[2].IP).
		Header(address.FromPubKey(hop[2].HeaderKey), set.Next()).
		Forward(client.IP).
		Header(address.FromPubKey(client.HeaderKey), set.Next()).
		Confirmation(id).
		Assemble()
}

// SendKeys provides a pair of private keys that will be used to generate the
// Purchase header bytes and to generate the ciphers provided in the Purchase
// message to encrypt the Session that is returned.
//
// The Header key, its cloaked public key counterpart used in the To field of
// the Purchase message preformed header bytes, but the Ciphers provided in the
// Purchase message, for encrypting the Session to be returned, uses the Payload
// key, along with the public key found in the encrypted layer of the header for
// the Return relay.
//
// This message's last layer is a Confirmation, which allows the client to know
// that the key was successfully delivered to the Return relays that will be
// used in the Purchase.
func SendKeys(id nonce.ID, hdr, pld *prv.Key,
	client node.Node, hop [5]node.Node, set signer.KeySet) Onion {

	return OnionSkins{}.
		Header(address.FromPubKey(hop[0].HeaderKey), set.Next()).
		Forward(hop[1].IP).
		Header(address.FromPubKey(hop[1].HeaderKey), set.Next()).
		Forward(hop[2].IP).
		Header(address.FromPubKey(hop[2].HeaderKey), set.Next()).
		Cipher(hdr, pld).
		Forward(hop[3].IP).
		Header(address.FromPubKey(hop[3].HeaderKey), set.Next()).
		Forward(hop[4].IP).
		Header(address.FromPubKey(hop[4].HeaderKey), set.Next()).
		Forward(client.IP).
		Header(address.FromPubKey(client.HeaderKey), set.Next()).
		Confirmation(id).
		Assemble()
}

// SendPurchase delivers a request for keys for a relaying session with a given
// router (in this case, hop 2). It is almost identical to an Exit except the
// payload is always just a 64-bit unsigned integer.
//
// The response, which will be two public keys that identify the session and
// form the basis of the cloaked "To" keys, is encrypted with the given layers,
// the ciphers are already given in reverse order, so they are decoded in given
// order to create the correct payload encryption to match the PayloadKey
// combined with the header's given public From key.
//
// The header remains a constant size and each node in the Return trims off
// their section at the top, moves the next layer header to the top and pads the
// remainder with noise, so it always looks like the first hop,
// indistinguishable.
func SendPurchase(nBytes uint64, client node.Node,
	hop [5]node.Node, set signer.KeySet) Onion {

	var rtns [3]*prv.Key
	for i := range rtns {
		rtns[i] = set.Next()
	}

	// The ciphers represent the combination of the same From key and the
	// payload keys combined, which the receiver knows means the header uses
	// HeaderKey and the message underneath use a different cipher in place
	// of the HeaderKey, the PayloadKey it corresponds to.
	ciphers := [3]sha256.Hash{
		ecdh.Compute(rtns[2], client.PayloadKey),
		ecdh.Compute(rtns[1], hop[4].PayloadKey),
		ecdh.Compute(rtns[0], hop[3].PayloadKey),
	}

	return OnionSkins{}.
		Header(address.FromPubKey(hop[0].HeaderKey), set.Next()).
		Forward(hop[1].IP).
		Header(address.FromPubKey(hop[1].HeaderKey), set.Next()).
		Forward(hop[2].IP).
		Header(address.FromPubKey(hop[2].HeaderKey), set.Next()).
		Purchase(nBytes, ciphers).
		Return(hop[3].IP).
		Header(address.FromPubKey(hop[3].HeaderKey), rtns[0]).
		Return(hop[4].IP).
		Header(address.FromPubKey(hop[4].HeaderKey), rtns[1]).
		Return(client.IP).
		Header(address.FromPubKey(client.HeaderKey), rtns[2]).
		Assemble()
}

// SendExit constructs a message containing an arbitrary payload to a node (3rd
// hop) with a set of 3 ciphers derived from the hidden PayloadKey of the return
// hops that are layered progressively after the Exit message.
//
// The Exit node forwards the packet it receives to the local port specified in
// the Exit message, and then uses the ciphers to encrypt the reply with the
// three ciphers provided, which don't enable it to decrypt the header, only to
// encrypt the payload.
//
// The response is encrypted with the given layers, the ciphers are already
// given in reverse order, so they are decoded in given order to create the
// correct payload encryption to match the PayloadKey combined with the header's
// given public From key.
//
// The header remains a constant size and each node in the Return trims off
// their section at the top, moves the next layer header to the top and pads the
// remainder with noise, so it always looks like the first hop,
// indistinguishable.
func SendExit(payload slice.Bytes, port uint16, client node.Node,
	hop [5]node.Node, set signer.KeySet) Onion {

	var rtns [3]*prv.Key
	for i := range rtns {
		rtns[i] = set.Next()
	}

	// The ciphers represent the combination of the same From key and the
	// payload keys combined, which the receiver knows means the header uses
	// HeaderKey and the message underneath use a different cipher in place
	// of the HeaderKey, the PayloadKey it corresponds to.
	ciphers := [3]sha256.Hash{
		ecdh.Compute(rtns[2], client.PayloadKey),
		ecdh.Compute(rtns[1], hop[4].PayloadKey),
		ecdh.Compute(rtns[0], hop[3].PayloadKey),
	}

	return OnionSkins{}.
		Header(address.FromPubKey(hop[0].HeaderKey), set.Next()).
		Forward(hop[1].IP).
		Header(address.FromPubKey(hop[1].HeaderKey), set.Next()).
		Forward(hop[2].IP).
		Header(address.FromPubKey(hop[2].HeaderKey), set.Next()).
		Exit(port, ciphers, payload).
		Return(hop[3].IP).
		Header(address.FromPubKey(hop[3].HeaderKey), rtns[0]).
		Return(hop[4].IP).
		Header(address.FromPubKey(hop[4].HeaderKey), rtns[1]).
		Return(client.IP).
		Header(address.FromPubKey(client.HeaderKey), rtns[2]).
		Assemble()
}