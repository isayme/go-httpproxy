package cmd

import (
	"fmt"
	"os"

	"github.com/isayme/go-httpproxy/httpproxy"
	"github.com/isayme/go-logger"
	"github.com/spf13/cobra"
)

var showVersion bool
var listenPort uint16
var proxyAddress string

func init() {
	rootCmd.Flags().BoolVarP(&showVersion, "version", "v", false, "show version")
	rootCmd.Flags().Uint16VarP(&listenPort, "port", "p", 8080, "listen port")
	rootCmd.Flags().StringVar(&proxyAddress, "proxy", "", "use this proxy")
}

var rootCmd = &cobra.Command{
	Use: "httpproxy",
	Run: func(cmd *cobra.Command, args []string) {
		if showVersion {
			httpproxy.ShowVersion()
			os.Exit(0)
		}

		address := fmt.Sprintf(":%d", listenPort)
		server, err := httpproxy.NewServer(address, proxyAddress)
		if err != nil {
			logger.Error(err)
			os.Exit(1)
		}
		logger.Infow("start listen ...", "addr", address)
		logger.Error(server.ListenAndServe())
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		logger.Panicf("rootCmd execute fail: %s", err.Error())
		os.Exit(1)
	}
}
