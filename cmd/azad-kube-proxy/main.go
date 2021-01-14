package main

import (
	"context"
	"fmt"
	"os"

	"github.com/go-logr/logr"
	"github.com/go-logr/zapr"
	"github.com/xenitab/azad-kube-proxy/pkg/config"
	"github.com/xenitab/azad-kube-proxy/pkg/proxy"
	"go.uber.org/zap"
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

	// Get configuration
	config, err := config.GetConfig(ctx)
	if err != nil {
		log.Error(err, "Unable to generate config")
		os.Exit(1)
	}

	// Start reverse proxy
	server, err := proxy.NewProxyServer(ctx, config)
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
