package cmd

import (
	"fmt"
	"os"
	"time"

	"github.com/iancoleman/strcase"
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
var connectTimeout time.Duration

func aliasNormalizeFunc(f *pflag.FlagSet, name string) pflag.NormalizedName {
	name = strcase.ToKebab(name)
	return pflag.NormalizedName(name)
}

func init() {
	rootCmd.Flags().BoolVarP(&showVersion, "version", "v", false, "show version")
	rootCmd.Flags().StringVarP(&logFormat, "log-format", "", "console", "log format")
	rootCmd.Flags().Uint16VarP(&listenPort, "port", "p", 1087, "listen port")
	rootCmd.Flags().StringVarP(&certFile, "cert-file", "", "", "cert file")
	rootCmd.Flags().StringVarP(&keyFile, "key-file", "", "", "key file")
	rootCmd.Flags().StringVar(&proxyAddress, "proxy", "", "use proxy, format: 'socks5://host:port' or 'http://host:port' or 'https://host:port'")
	rootCmd.Flags().DurationVar(&connectTimeout, "connect-timeout", time.Duration(0), "timeout of dial proxy or remote")

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
		
		options := []httpproxy.ServerOption{}
		if proxyAddress != "" {
			options = append(options, httpproxy.WithProxy(proxyAddress))
		}
		if connectTimeout > 0 {
			options = append(options, httpproxy.WithConnectTimeout(connectTimeout))
		}

		server, err := httpproxy.NewServer(address, options...)
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
