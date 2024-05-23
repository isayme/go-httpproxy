package cmd

import (
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
var logLevel string
var listenAddress string
var listenPort uint16
var username string
var password string
var certFile string
var keyFile string
var proxyAddress string
var connectTimeout time.Duration
var timeout time.Duration

func aliasNormalizeFunc(f *pflag.FlagSet, name string) pflag.NormalizedName {
	name = strcase.ToKebab(name)
	return pflag.NormalizedName(name)
}

func init() {
	rootCmd.Flags().BoolVarP(&showVersion, "version", "v", false, "show version")
	rootCmd.Flags().StringVarP(&logFormat, "log-format", "", "console", "log format")
	rootCmd.Flags().StringVarP(&logLevel, "log-level", "", "info", "log level")
	rootCmd.Flags().Uint16VarP(&listenPort, "port", "p", 1087, "listen port, omit if --listen-address used")
	rootCmd.Flags().StringVarP(&listenAddress, "listen-address", "", "0.0.0.0:1087", "listen address")
	rootCmd.Flags().StringVarP(&username, "username", "", "", "proxy server auth username")
	rootCmd.Flags().StringVarP(&password, "password", "", "", "proxy server auth password")
	rootCmd.Flags().StringVarP(&certFile, "cert-file", "", "", "cert file")
	rootCmd.Flags().StringVarP(&keyFile, "key-file", "", "", "key file")
	rootCmd.Flags().StringVar(&proxyAddress, "proxy", "", "use proxy, format: 'socks5://host:port' or 'http://host:port' or 'https://host:port'")
	rootCmd.Flags().DurationVar(&connectTimeout, "connect-timeout", time.Second*5, "timeout of dial proxy or remote")
	rootCmd.Flags().DurationVar(&timeout, "timeout", time.Second*30, "timeout of read/write")

	rootCmd.Flags().SetNormalizeFunc(aliasNormalizeFunc)
}

var rootCmd = &cobra.Command{
	Use: "httpproxy",
	Run: func(cmd *cobra.Command, args []string) {
		if showVersion {
			httpproxy.ShowVersion()
			os.Exit(0)
		}

		logger.SetLevel(logLevel)
		logger.SetFormat(logFormat)
		logger.Debugf("set log level: %s", logLevel)
		logger.Debugf("set log format: %s", logFormat)

		options := []httpproxy.ServerOption{
			httpproxy.WithListenPort(listenPort),
			httpproxy.WithListenAddress(listenAddress),
			httpproxy.WithUsername(username),
			httpproxy.WithPassword(password),
			httpproxy.WithConnectTimeout(connectTimeout),
			httpproxy.WithTimeout(timeout),
			httpproxy.WithProxy(proxyAddress),
			httpproxy.WithCertFile(certFile),
			httpproxy.WithKeyFile(keyFile),
		}

		logger.Debugw("option", "listen-port", listenPort)
		logger.Debugw("option", "listen-address", listenAddress)

		maskPassword := password
		if len(maskPassword) > 1 {
			maskPassword = password[:1] + "***" + password[len(password)-1:]
		}
		logger.Debugw("option", "username", username, "password", maskPassword)

		logger.Debugw("option", "connect-timeout", connectTimeout.String())
		logger.Debugw("option", "timeout", timeout.String())

		logger.Debugw("option", "proxy", proxyAddress)
		logger.Debugw("option", "certFile", certFile, "keyFile", keyFile)

		server, err := httpproxy.NewServer(options...)
		if err != nil {
			logger.Error(err)
			os.Exit(1)
		}

		logger.Error(server.ListenAndServe())
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		logger.Panicf("rootCmd execute fail: %s", err.Error())
		os.Exit(1)
	}
}
