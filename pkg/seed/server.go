package seed

import (
	"context"
	"git-indra.lan/indra-labs/indra/pkg/rpc"
	"strconv"
	"time"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/multiformats/go-multiaddr"

	"git-indra.lan/indra-labs/indra"
	"git-indra.lan/indra-labs/indra/pkg/interrupt"
	"git-indra.lan/indra-labs/indra/pkg/p2p/introducer"
	log2 "git-indra.lan/indra-labs/indra/pkg/proc/log"
	"git-indra.lan/indra-labs/indra/pkg/seed/metrics"
)

var (
	log   = log2.GetLogger(indra.PathBase)
	check = log.E.Chk
)

var (
	userAgent = "/indra:" + indra.SemVer + "/"
)

type Server struct {
	context.Context

	config *Config

	host host.Host

	rpc *rpc.RPC
}

func (srv *Server) Restart() (err error) {

	log.I.Ln("restarting the server.")

	return nil
}

func (srv *Server) Shutdown() (err error) {

	log.I.Ln("shutting down [p2p.host]")

	if srv.host.Close(); check(err) {
		return
	}

	log.I.Ln("shutdown complete")

	return nil
}

func (srv *Server) Serve() (err error) {

	log.I.Ln("starting the server")

	if srv.config.RPCConfig.IsEnabled() {
		srv.rpc.Start()
	}

	// Here we create a context with cancel and add it to the interrupt handler
	var ctx context.Context
	var cancel context.CancelFunc

	ctx, cancel = context.WithCancel(context.Background())

	interrupt.AddHandler(cancel)

	// Introduce your node to the network
	go introducer.Bootstrap(ctx, srv.host, srv.config.SeedAddresses)

	// Get some basic metrics for the host
	// metrics.Init()
	// metrics.Set('indra.host.status.reporting.interval', 30 * time.Second)
	// metrics.Enable('indra.host.status')
	metrics.SetInterval(30 * time.Second)

	go metrics.HostStatus(ctx, srv.host)

	select {

	case <-ctx.Done():

		log.I.Ln("shutting down server")

		srv.Shutdown()
	}

	return nil
}

func New(config *Config) (*Server, error) {

	log.I.Ln("initializing the server")

	var err error
	var s Server

	s.config = config

	if config.RPCConfig.IsEnabled() {

		log.I.Ln("enabling rpc server")

		if s.rpc, err = rpc.New(config.RPCConfig); check(err) {
			return nil, err
		}

		log.I.Ln("rpc public key:")
		log.I.Ln("-", config.RPCConfig.Key.PubKey().Encode())

		log.I.Ln("rpc listeners:")
		log.I.Ln("- [/ip4/0.0.0.0/udp/"+strconv.Itoa(int(config.RPCConfig.ListenPort)), "/ip6/:::/udp/"+strconv.Itoa(int(config.RPCConfig.ListenPort))+"]")
	}

	//var client *rpc.RPCClient
	//
	//if client, err = rpc.NewClient(rpc.DefaultClientConfig); check(err) {
	//	return nil, err
	//}
	//
	//client.Start()

	if s.host, err = libp2p.New(libp2p.Identity(config.PrivKey), libp2p.UserAgent(userAgent), libp2p.ListenAddrs(config.ListenAddresses...)); check(err) {
		return nil, err
	}

	log.I.Ln("host id:")
	log.I.Ln("-", s.host.ID())

	log.I.Ln("p2p listeners:")
	log.I.Ln("-", s.host.Addrs())

	if len(config.ConnectAddresses) > 0 {

		log.I.Ln("connect detected, using only the connect seed addresses")

		config.SeedAddresses = config.ConnectAddresses

		return &s, nil
	}

	var seedAddresses []multiaddr.Multiaddr

	if seedAddresses, err = config.Params.ParseSeedMultiAddresses(); check(err) {
		return nil, err
	}

	config.SeedAddresses = append(config.SeedAddresses, seedAddresses...)

	return &s, err
}
