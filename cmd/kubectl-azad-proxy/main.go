package main

import (
	"context"
	"fmt"
	"os"

	"github.com/bombsimon/logrusr/v2"
	"github.com/go-logr/logr"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
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
	cli.VersionPrinter = func(c *cli.Context) {
		fmt.Printf("version=%s revision=%s created=%s\n", c.App.Version, Revision, Created)
	}

	globalFlags := []cli.Flag{
		&cli.BoolFlag{
			Name:  "debug",
			Usage: "Should debugging be enabled?",
			Value: false,
		},
	}

	generateFlags, err := generateFlags(ctx)
	if err != nil {
		return err
	}

	loginFlags, err := loginFlags(ctx)
	if err != nil {
		return err
	}

	menuFlags, err := menuFlags(ctx)
	if err != nil {
		return err
	}

	app := &cli.App{
		Name:    "kubectl-azad-proxy",
		Usage:   "kubectl plugin for azad-kube-proxy",
		Version: Version,
		Flags:   globalFlags,
		Commands: []*cli.Command{
			{
				Name:    "generate",
				Aliases: []string{"g"},
				Usage:   "Generate kubeconfig",
				Flags:   append(generateFlags, globalFlags...),
				Action: func(c *cli.Context) error {
					client, err := newGenerateClient(ctx, c)
					if err != nil {
						return err
					}
					return client.Generate(ctx)
				},
			},
			{
				Name:    "login",
				Aliases: []string{"l"},
				Usage:   "Login to Azure AD app and return token",
				Flags:   append(loginFlags, globalFlags...),
				Action: func(c *cli.Context) error {
					client, err := newLoginClient(ctx, c)
					if err != nil {
						return err
					}

					output, err := client.Login(ctx)
					if err != nil {
						return err
					}

					fmt.Print(output)
					return nil
				},
			},
			{
				Name:    "discover",
				Aliases: []string{"d"},
				Usage:   "Discovery for the azad-kube-proxy enabled apps and their configuration",
				Flags:   append(discoverFlags(ctx), globalFlags...),
				Action: func(c *cli.Context) error {
					client, err := newDiscoverClient(ctx, c)
					if err != nil {
						return err
					}

					output, err := client.Discover(ctx)
					if err != nil {
						return err
					}

					fmt.Print(output)
					return nil
				},
			},
			{
				Name:    "menu",
				Aliases: []string{"m"},
				Usage:   "Menu for the azad-kube-proxy configuration",
				Flags:   append(menuFlags, globalFlags...),
				Action: func(c *cli.Context) error {
					client, err := newMenuClient(ctx, c)
					if err != nil {
						return err
					}

					return client.Menu(ctx)
				},
			},
		},
	}

	err = app.Run(os.Args)
	if err != nil {
		return err
	}

	return nil
}
