package main

import (
	"context"
	"fmt"
	"os"

	"github.com/go-logr/logr"
	"github.com/go-logr/zapr"
	"github.com/urfave/cli/v2"
	"github.com/xenitab/azad-kube-proxy/pkg/config"
	"github.com/xenitab/azad-kube-proxy/pkg/proxy"
	"go.uber.org/zap"
)

var (
	// Version is the release version and will be set during compile time
	Version = "v0.0.0-dev"

	// Revision is the git sha
	Revision = ""

	// Created is the timestamp for when the application was created
	Created = ""
)

func main() {
	// Initiate the logging
	var log logr.Logger

	zapLog, err := zap.NewProduction()
	if err != nil {
		panic(fmt.Sprintf("who watches the watchmen (%v)?", err))
	}
	log = zapr.NewLogger(zapLog)
	ctx := logr.NewContext(context.Background(), log)

	// Generate config
	cli.VersionPrinter = func(c *cli.Context) {
		fmt.Printf("version=%s revision=%s created=%s\n", c.App.Version, Revision, Created)
	}

	var cfg config.Config

	app := &cli.App{
		Name:    "azad-kube-proxy",
		Usage:   "Azure AD Kubernetes API Proxy",
		Version: Version,
		Flags:   config.Flags(ctx),
		Action: func(c *cli.Context) error {
			var err error
			cfg, err = config.NewConfig(ctx, c)
			if err != nil {
				return err
			}
			return nil
		},
	}

	err = app.Run(os.Args)
	if err != nil {
		log.Error(err, "Unable to generate config")
		os.Exit(1)
	}

	// Start reverse proxy
	server, err := proxy.NewProxyClient(ctx, cfg)
	if err != nil {
		log.Error(err, "Unable to initialize proxy server")
		os.Exit(1)
	}

	err = server.Start(ctx)
	if err != nil {
		log.Error(err, "Proxy server returned error")
		os.Exit(1)
	}

	os.Exit(0)
}
