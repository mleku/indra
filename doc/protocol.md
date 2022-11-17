# Indranet Protocol Specification

Here will be the packet frames, rendezvous, voucher minting and reservation
payments and other specifications for how Indranet works.

## Packet Frame

The packet frame is the layout of individual packets, but it is also used
inside messages. The outer, visible part is laid out as follows:

1. [**16 bytes**] Nonce - the random value used to secure the encryption key
   when combined with the message.
2. [**8 bytes**] Address - A cloaked representation of the public key acquired
   from the correspondent from a previous message or initial handshake. This
   uses 3 bytes of cryptographic random numbers plus the hash of the full
   public key concatenated at the end of the nonce value, hashed and sliced
   to 8 bytes length. Only the sender will be able to identify this to
   associate a batch of message segments together.
3. Payload - The encrypted contents of the message, Includes a header and a
   return public key in the footer.
	1. [**2 bytes**] Message Sequence number - the sequence position of this
	   segment in the message.
	2. [**4 bytes**] Message total size.
	3. [**1 byte**] Parity - a value between 0 and 255 that represents a
	   ratio against 256 for added redundancy for the Reed Solomon
	   Forward Error Correction used to avoid retransmission.
	4. [**33 bytes**] Receiver public key - this key the sender has cached
	   and will be able to recognise the Address header matches a
	   cloaked version of this key for sending a reply message.
	5. **Remainder of available from 1472 bytes**  The data

4. [**4 bytes**] Checksum - truncated SHA256 hash of the rest of Payload
5. [**65 bytes**] Signature - compact ECDSA signature which validates with the
   message part before the checksum to provide the public key needed in
   combination with the receiver's private key.

## Addresses

Addresses consist of a 3 byte prefix nonce and 5 bytes being the product of
concatenating the prefix against the public key and then hashing it. Only the
recipient will know the public key and can thus quickly validate and match
this by using the prefix against its known keys.

The purpose of this scheme is to enable the recognition of segments of a
larger message, without revealing to attackers that the messages are related.

## Signer

Standard key generation can be quite slow and consume a substantial amount
of system entropy. As such, for a multi-segment message instead the sender
generates two keys, and each subsequent message is signed using a key that
is derived by adding the second key's value to the first and then previously
generated keys in order that no message contains a visibly repeating public key.

This signature provides the public key to combine with the private key
indicated by the address, which is combined via ECDH (Elliptic Curve Diffie
Hellman) key exchange, to generate the message cipher, which then allows the
decryption of the Payload.

This scheme is not as computationally cheap as standard message stream
cipher key exchange, but it is as efficient as possible while completely
obscuring relationships between packets.

By ensuring that intermediaries cannot derive any other information about
the relationship between messages, the only remaining data for surveillance
is message timing. For this purpose, message ordering is intentionally
shuffled randomly, and small, random delays are deliberately inserted
between packets, and between relaying messages are very frequent gossip
messages that are for coordinating information between peers. Additionally, the
packet dispatch queue will pick off pieces from the waiting queues randomly.

Shuffling message order also helps ensure that for longer transmissions over 256
packets in length, that any lost packets are distributed across the message
delivery timeline, protecting against longer bursts of disruption breaking a
single section of a message and causing transmission failure.

## Clients, Routers, and the Messaging Model

Only first and last hop layers of the onions contain internet routing 
information relating to clients.

Because clients will often be stuck behind NAT routers that do not allow inbound connection, clients establish outbound connections to both the first hop and the final return hop from connections paths they construct in onions.

> **Inbound Connection Proxy Service** – Because of the common problem of ISPs not providing stable, or inbound routeable connections, one of the services that Indra Labs will offer in addition to producing small embedded hardware with pre-loaded chain data and server installations, is inbound addresses that routers will be able to advertise as their reachable addresses, and the router will create an outbound connection to the inbound port.

This only applies to routers – clients will simply open persistent connections to any node they expect a return connection to come from, in addition to their outbound connections, first and last hop in the 5 hop circuit. Most high capacity home connections have inbound routing capability, and one of the router hardware offerings will enable the user to place the router on the WAN connection and have it forward other traffic to the main house router.

In initial development, the client/router model will be implemented using in-process channels, and when network capability is implemented, it will use these channels to pass messages between the client threads and the network dispatch threads.
