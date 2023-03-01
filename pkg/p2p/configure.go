package p2p

import (
	"git-indra.lan/indra-labs/indra/pkg/cfg"
	"git-indra.lan/indra-labs/indra/pkg/storage"
	"github.com/dgraph-io/badger/v3"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/multiformats/go-multiaddr"
	"github.com/spf13/viper"
)

func configure() {

	log.I.Ln("initializing p2p")

	configureKey()
	configureListeners()
	configureSeeds()
}

func configureKey() {

	log.I.Ln("looking for key in storage")

	var err error
	var item *badger.Item
	var keyBytes []byte = make([]byte, 32)

	err = storage.View(func(txn *badger.Txn) error {

		if item, err = txn.Get([]byte(keyFlag)); err != nil {
			return err
		}

		item.ValueCopy(keyBytes)

		return nil
	})

	if err == badger.ErrKeyNotFound {

		log.I.Ln("key not found, generating a new one")

		if privKey, _, err = crypto.GenerateKeyPair(crypto.Secp256k1, 0); check(err) {
			return
		}

		if keyBytes, err = privKey.Raw(); check(err) {
			return
		}

		err = storage.Update(func(txn *badger.Txn) error {
			err = txn.Set([]byte(keyFlag), keyBytes)
			check(err)
			return nil
		})

		return
	}

	if privKey, err = crypto.UnmarshalSecp256k1PrivateKey(keyBytes); check(err) {
		return
	}

	log.I.Ln("key found")
}

func configureListeners() {

	if len(viper.GetStringSlice(listenFlag)) == 0 {
		log.I.Ln("no listeners found, using defaults")
		return
	}

	for _, listener := range viper.GetStringSlice(listenFlag) {
		listenAddresses = append(listenAddresses, multiaddr.StringCast(listener))
	}
}

func configureSeeds() {

	if len(viper.GetStringSlice(connectFlag)) > 0 {

		log.I.Ln("connect only detected, using only the connect seed addresses")

		for _, connector := range viper.GetStringSlice(connectFlag) {
			seedAddresses = append(seedAddresses, multiaddr.StringCast(connector))
		}

		return
	}

	var err error

	netParams = cfg.SelectNetworkParams(viper.GetString("network"))

	if seedAddresses, err = netParams.ParseSeedMultiAddresses(); err != nil {
		return
	}

	if len(viper.GetStringSlice("seed")) > 0 {

		log.I.Ln("found", len(viper.GetStringSlice("seed")), "additional seeds.")

		for _, seed := range viper.GetStringSlice("seed") {
			seedAddresses = append(seedAddresses, multiaddr.StringCast(seed))
		}
	}

	return
}