package engine

import (
	"errors"
	"math/rand"
	"testing"
	
	"git-indra.lan/indra-labs/indra/pkg/crypto/key/cloak"
	"git-indra.lan/indra-labs/indra/pkg/crypto/key/prv"
	"git-indra.lan/indra-labs/indra/pkg/crypto/key/pub"
	"git-indra.lan/indra-labs/indra/pkg/crypto/nonce"
	"git-indra.lan/indra-labs/indra/pkg/crypto/sha256"
	log2 "git-indra.lan/indra-labs/indra/pkg/proc/log"
	"git-indra.lan/indra-labs/indra/pkg/util/tests"
)

func TestSplitJoin(t *testing.T) {
	log2.SetLogLevel(log2.Trace)
	msgSize := 1 << 12
	segSize := 1382
	var e error
	var payload []byte
	var pHash sha256.Hash
	if payload, pHash, e = tests.GenMessage(msgSize, ""); fails(e) {
		t.FailNow()
	}
	var sp, rp *prv.Key
	var rP *pub.Key
	if sp, rp, _, rP, e = tests.GenerateTestKeyPairs(); fails(e) {
		t.FailNow()
	}
	addr := rP
	params := &PacketParams{
		To:     addr,
		From:   sp,
		Length: len(payload),
		Data:   payload,
		Parity: 128,
	}
	var splitted [][]byte
	if _, splitted, e = SplitToPackets(params, segSize); fails(e) {
		t.Error(e)
	}
	// log.D.S("splittid", splitted)
	var pkts Packets
	var keys []*pub.Key
	for i := range splitted {
		var pkt *Packet
		var from *pub.Key
		var to cloak.PubKey
		_ = to
		var iv nonce.IV
		if from, to, iv, e = GetKeysFromPacket(splitted[i]); fails(e) {
			log.I.Ln(i)
			continue
		}
		if pkt, e = DecodePacket(splitted[i], from, rp, iv); fails(e) {
			t.Error(e)
		}
		pkts = append(pkts, pkt)
		keys = append(keys, from)
	}
	prev := keys[0]
	// fails all keys are the same
	for _, k := range keys[1:] {
		if !prev.Equals(k) {
			t.Error(e)
		}
		prev = k
	}
	var msg []byte
	if pkts, msg, e = JoinPackets(pkts); fails(e) {
		t.Error(e)
	}
	rHash := sha256.Single(msg)
	if pHash != rHash {
		t.Error(errors.New("message did not decode correctly"))
	}
}

func BenchmarkSplit(b *testing.B) {
	msgSize := 1 << 16
	segSize := 1382
	var e error
	var payload []byte
	if payload, _, e = tests.GenMessage(msgSize, ""); fails(e) {
		b.Error(e)
	}
	var sp *prv.Key
	var rP *pub.Key
	if sp, _, _, rP, e = tests.GenerateTestKeyPairs(); fails(e) {
		b.FailNow()
	}
	addr := rP
	for n := 0; n < b.N; n++ {
		params := &PacketParams{
			To:     addr,
			From:   sp,
			Parity: 64,
			Data:   payload,
		}
		
		var splitted [][]byte
		if _, splitted, e = SplitToPackets(params, segSize); fails(e) {
			b.Error(e)
		}
		_ = splitted
	}
	
	// Example benchmark results show about 10Mb/s/thread throughput
	// handling 64Kb messages.
	//
	// goos: linux
	// goarch: amd64
	// pkg: git-indra.lan/indra-labs/indra/pkg/packet
	// cpu: AMD Ryzen 7 5800H with Radeon Graphics
	// BenchmarkSplit
	// BenchmarkSplit-16    	     157	   7670080 ns/op
	// PASS
}

func TestRemovePacket(t *testing.T) {
	packets := make(Packets, 10)
	for i := range packets {
		packets[i] = &Packet{Seq: uint16(i)}
	}
	var seqs []uint16
	for i := range packets {
		seqs = append(seqs, packets[i].Seq)
	}
	discard := []int{1, 5, 6}
	for i := range discard {
		// Subtracting the iterator accounts for the backwards shift of
		// the shortened slice.
		packets = RemovePacket(packets, discard[i]-i)
	}
	var seqs2 []uint16
	for i := range packets {
		seqs2 = append(seqs2, packets[i].Seq)
	}
}

func TestSplitJoinFEC(t *testing.T) {
	log2.SetLogLevel(log2.Trace)
	msgSize := 2 << 15
	segSize := 1382
	var e error
	var sp, rp, Rp *prv.Key
	var sP, rP, RP *pub.Key
	if sp, rp, sP, rP, e = tests.GenerateTestKeyPairs(); fails(e) {
		t.FailNow()
	}
	_, _, _, _ = sP, Rp, RP, rp
	var parity []int
	for i := 1; i < 255; i *= 2 {
		parity = append(parity, i)
	}
	for i := range parity {
		var payload []byte
		var pHash sha256.Hash
		if payload, pHash, e = tests.GenMessage(msgSize, "b0rk"); fails(e) {
			t.FailNow()
		}
		var punctures []int
		// Generate a set of numbers of punctures starting from equal to
		// parity in a halving sequence to reduce the number but see it
		// function.
		for punc := parity[i]; punc > 0; punc /= 2 {
			punctures = append(punctures, punc)
		}
		// Reverse the ordering just because.
		for p := 0; p < len(punctures)/2; p++ {
			punctures[p], punctures[len(punctures)-p-1] =
				punctures[len(punctures)-p-1], punctures[p]
		}
		addr := rP
		for p := range punctures {
			var splitted [][]byte
			ep := &PacketParams{
				To:     addr,
				From:   sp,
				Parity: parity[i],
				Length: len(payload),
				Data:   payload,
			}
			if _, splitted, e = SplitToPackets(ep, segSize); fails(e) {
				t.Error(e)
				t.FailNow()
			}
			overhead := ep.GetOverhead()
			segMap := NewPacketSegments(len(ep.Data), segSize, overhead,
				ep.Parity)
			for segs := range segMap {
				start := segMap[segs].DStart
				end := segMap[segs].PEnd
				cnt := end - start
				par := segMap[segs].PEnd - segMap[segs].DEnd
				a := make([][]byte, cnt)
				for ss := range a {
					a[ss] = splitted[start:end][ss]
				}
				rand.Seed(int64(punctures[p]))
				rand.Shuffle(cnt,
					func(i, j int) {
						a[i], a[j] = a[j], a[i]
					})
				puncture := punctures[p]
				if puncture > par {
					puncture = par
				}
				for n := 0; n < puncture; n++ {
					copy(a[n][:100], make([]byte, 10))
				}
			}
			var pkts Packets
			var keys []*pub.Key
			for s := range splitted {
				var pkt *Packet
				var from *pub.Key
				var to cloak.PubKey
				_ = to
				var iv nonce.IV
				if from, to, iv, e = GetKeysFromPacket(
					splitted[s]); e != nil {
					// we are puncturing, they some will
					// fail to decode
					continue
				}
				if pkt, e = DecodePacket(splitted[s],
					from, rp, iv); fails(e) {
					continue
				}
				pkts = append(pkts, pkt)
				keys = append(keys, from)
			}
			var msg []byte
			if pkts, msg, e = JoinPackets(pkts); fails(e) {
				t.FailNow()
			}
			rHash := sha256.Single(msg)
			if pHash != rHash {
				t.Error(errors.New("message did not decode" +
					" correctly"))
			}
		}
	}
}