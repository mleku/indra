package seed

import (
	"context"
	"git-indra.lan/indra-labs/indra"
	"git-indra.lan/indra-labs/indra/pkg/cfg"
	"git-indra.lan/indra-labs/indra/pkg/interrupt"
	log2 "git-indra.lan/indra-labs/indra/pkg/proc/log"
	"git-indra.lan/indra-labs/indra/pkg/seed"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Serves an instance of the seed node",
	Long:  `Serves an instance of the seed node.`,
	Run: func(cmd *cobra.Command, args []string) {

		log.I.Ln("-- ", log2.App, "("+viper.GetString("network")+") -", indra.SemVer, "- Network Freedom. --")

		cfg.SelectNetworkParams(viper.GetString("network"))

		ctx, cancel := context.WithCancel(context.Background())
		interrupt.AddHandler(cancel)

		// Seed //

		go seed.Run(ctx)

		select {
		case <-seed.WhenStartFailed():
			log.I.Ln("stopped")
		case <-seed.WhenShutdown():
			log.I.Ln("shutdown complete")
		}

		log.I.Ln("-- fin --")
	},
}