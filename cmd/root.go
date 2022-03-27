package cmd

import (
	"fmt"
	"os"

	"github.com/isayme/go-httpproxy/httpproxy"
	"github.com/isayme/go-logger"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

var showVersion bool
var logFormat string
var listenPort uint16
var certFile string
var keyFile string
var proxyAddress string

func aliasNormalizeFunc(f *pflag.FlagSet, name string) pflag.NormalizedName {
	switch name {
	case "cert-file", "cert_file":
		name = "certFile"
	case "key-file", "key_file":
		name = "keyFile"
	case "log-format", "log_format":
		name = "logFormat"
	}
	return pflag.NormalizedName(name)
}

func init() {
	rootCmd.Flags().BoolVarP(&showVersion, "version", "v", false, "show version")
	rootCmd.Flags().StringVarP(&logFormat, "logFormat", "", "console", "log format")
	rootCmd.Flags().Uint16VarP(&listenPort, "port", "p", 1087, "listen port")
	rootCmd.Flags().StringVarP(&certFile, "certFile", "", "", "cert file")
	rootCmd.Flags().StringVarP(&keyFile, "keyFile", "", "", "key file")
	rootCmd.Flags().StringVar(&proxyAddress, "proxy", "", "use this proxy")
	rootCmd.Flags().SetNormalizeFunc(aliasNormalizeFunc)
}

var rootCmd = &cobra.Command{
	Use: "httpproxy",
	Run: func(cmd *cobra.Command, args []string) {
		if showVersion {
			httpproxy.ShowVersion()
			os.Exit(0)
		}

		if logFormat != "" {
			logger.SetFormat(logFormat)
		}

		address := fmt.Sprintf(":%d", listenPort)
		server, err := httpproxy.NewServer(address, proxyAddress)
		if err != nil {
			logger.Error(err)
			os.Exit(1)
		}

		if certFile != "" && keyFile != "" {
			logger.Infow("start listen with tls ...", "addr", address)
			logger.Error(server.ListenAndServeTLS(certFile, keyFile))
		} else {
			logger.Infow("start listen ...", "addr", address)
			logger.Error(server.ListenAndServe())
		}
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		logger.Panicf("rootCmd execute fail: %s", err.Error())
		os.Exit(1)
	}
}
