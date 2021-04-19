package main

import (
	"context"
	"fmt"
	"os"

	"github.com/bombsimon/logrusr"
	"github.com/go-logr/logr"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
	"github.com/xenitab/azad-kube-proxy/cmd/kubectl-azad-proxy/actions"
	"github.com/xenitab/azad-kube-proxy/cmd/kubectl-azad-proxy/customerrors"
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
	// Initiate the logging
	logrusLog := logrus.New()
	if util.SliceContains(os.Args, "--debug") || util.SliceContains(os.Args, "-debug") {
		logrusLog.Level = 10
	}
	log := logrusr.NewLogger(logrusLog)
	ctx := logr.NewContext(context.Background(), log)

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
				Flags:   append(actions.GenerateFlags(ctx), globalFlags...),
				Action: func(c *cli.Context) error {
					client, err := actions.NewGenerateClient(ctx, c)
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
				Flags:   append(actions.LoginFlags(ctx), globalFlags...),
				Action: func(c *cli.Context) error {
					client, err := actions.NewLoginClient(ctx, c)
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
				Flags:   append(actions.DiscoverFlags(ctx), globalFlags...),
				Action: func(c *cli.Context) error {
					client, err := actions.NewDiscoverClient(ctx, c)
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
				Flags:   append(actions.MenuFlags(ctx), globalFlags...),
				Action: func(c *cli.Context) error {
					client, err := actions.NewMenuClient(ctx, c)
					if err != nil {
						return err
					}

					return client.Menu(ctx)
				},
			},
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		customErr := customerrors.To(err)
		log.Error(customErr, "Application returned error", "ErrorType", customErr.ErrorType)
		os.Exit(1)
	}

	os.Exit(0)
}
