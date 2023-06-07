package transport

import (
	"context"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/p2p/discovery/routing"
	"github.com/multiformats/go-multiaddr"
	"sync"
	"time"
)

func NewDHT(ctx context.Context, host host.Host,
	bootstrapPeers []multiaddr.Multiaddr) (d *dht.IpfsDHT, e error) {

	var options []dht.Option
	if len(bootstrapPeers) == 0 {
		options = append(options, dht.Mode(dht.ModeServer))
	}
	options = append(options,
		dht.ProtocolPrefix(IndraLibP2PID),
	)
	if d, e = dht.New(ctx, host, options...); fails(e) {
		return
	}
	if e = d.Bootstrap(ctx); fails(e) {
		return
	}
	var wg sync.WaitGroup
	for _, peerAddr := range bootstrapPeers {
		var peerinfo *peer.AddrInfo
		if peerinfo, e = peer.AddrInfoFromP2pAddr(peerAddr); fails(e) {
			continue
		}
		wg.Add(1)
		go func() {
			if e := host.Connect(ctx, *peerinfo); fails(e) {
				log.D.F("Error while connecting to node %q: %-v",
					peerinfo, e)
				wg.Done()
				return
			}
			log.D.F(
				"%s: Connection established with bootstrap node: %s",
				blue(GetHostOnlyAddress(host)),
				blue((*peerinfo).Addrs[0]))
			wg.Done()
		}()
	}
	wg.Wait()
	return
}

func Discover(ctx context.Context, h host.Host, dht *dht.IpfsDHT,
	rendezvous string) {

	var disco = routing.NewRoutingDiscovery(dht)
	var e error
	var peers <-chan peer.AddrInfo
	if _, e = disco.Advertise(ctx, rendezvous); e != nil {
	}
	ticker := time.NewTicker(time.Second * 1)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if peers, e = disco.FindPeers(ctx, rendezvous); fails(e) {
				return
			}
			for p := range peers {
				if p.ID == h.ID() {
					continue
				}
				if h.Network().Connectedness(p.ID) !=
					network.Connected {

					if _, e = h.Network().DialPeer(ctx,
						p.ID); fails(e) {

						continue
					}
					log.D.Ln(h.Addrs()[0].String(), "Connected to peer",
						blue(p.Addrs[0]))
				}
			}
		}
	}
}