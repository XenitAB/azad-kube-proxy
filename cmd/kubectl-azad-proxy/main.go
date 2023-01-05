package main

import (
	"context"
	"fmt"
	"os"

	"github.com/bombsimon/logrusr/v2"
	"github.com/go-logr/logr"
	"github.com/sirupsen/logrus"
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
	cfg, err := newConfig(os.Args[1:])
	if err != nil {
		os.Exit(1)
	}

	logrusLog := logrus.New()
	if cfg.Debug {
		logrusLog.Level = 10
	}
	log := logrusr.New(logrusLog)
	ctx := logr.NewContext(context.Background(), log)

	err = run(ctx, cfg)
	if err != nil {
		customErr := toCustomError(err)
		log.Error(customErr, "Application returned error", "errorType", customErr.errorType)
		os.Exit(1)
	}
}

func run(ctx context.Context, cfg config) error {
	switch {
	case cfg.Discover != nil:
		return runDiscover(ctx, os.Stdout, *cfg.Discover, cfg.authConfig)
	case cfg.Generate != nil:
		return runGenerate(ctx, *cfg.Generate, cfg.authConfig)
	case cfg.Login != nil:
		return runLogin(ctx, os.Stdout, *cfg.Login, cfg.authConfig)
	case cfg.Menu != nil:
		return runMenu(ctx, *cfg.Menu, cfg.authConfig, newPromptClient())
	}

	return fmt.Errorf("unknown error, no subcommand executed")
}
