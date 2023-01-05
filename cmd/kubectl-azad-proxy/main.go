package main

import (
	"context"
	"fmt"
	"os"

	"github.com/bombsimon/logrusr/v2"
	"github.com/go-logr/logr"
	"github.com/sirupsen/logrus"
	"github.com/xenitab/azad-kube-proxy/pkg/util"
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
	logrusLog := logrus.New()
	if util.SliceContains(os.Args, "--debug") || util.SliceContains(os.Args, "-debug") {
		logrusLog.Level = 10
	}
	log := logrusr.New(logrusLog)
	ctx := logr.NewContext(context.Background(), log)

	err := run(ctx)
	if err != nil {
		customErr := toCustomError(err)
		log.Error(customErr, "Application returned error", "errorType", customErr.errorType)
		os.Exit(1)
	}
}

func run(ctx context.Context) error {
	cfg, err := newConfig(os.Args[1:])
	if err != nil {
		return err
	}

	switch {
	case cfg.Discover != nil:
		return runDiscover(ctx, os.Stdout, *cfg.Discover, cfg.authConfig)
	case cfg.Generate != nil:
		return runGenerate(ctx, *cfg.Generate, cfg.authConfig)
	case cfg.Login != nil:
		return runLogin(ctx, os.Stdout, *cfg.Login, cfg.authConfig)
	case cfg.Menu != nil:
		return runMenu(ctx, *cfg.Menu, cfg.authConfig)
	}

	return fmt.Errorf("unknown error")
}
