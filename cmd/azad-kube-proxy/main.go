package main

import (
	"context"
	"fmt"
	"os"

	"github.com/go-logr/logr"
	"github.com/go-logr/zapr"
	"github.com/xenitab/azad-kube-proxy/pkg/app"
	"go.uber.org/zap"
)

func main() {
	// Logs
	var log logr.Logger

	zapLog, err := zap.NewProduction()
	if err != nil {
		panic(fmt.Sprintf("who watches the watchmen (%v)?", err))
	}
	log = zapr.NewLogger(zapLog)
	ctx := logr.NewContext(context.Background(), log)

	// Run
	err = app.Get(ctx).Run(os.Args)
	if err != nil {
		log.Error(err, "Application returned error")
		os.Exit(1)
	}

	os.Exit(0)
}
