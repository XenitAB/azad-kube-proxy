package main

import (
	"context"
	"fmt"
	"os"

	"github.com/go-logr/logr"
	"github.com/urfave/cli/v2"
	"github.com/xenitab/azad-kube-proxy/cmd/kubectl-azad-proxy/actions"
	"k8s.io/klog/v2/klogr"
)

func main() {
	// Initiate the logging
	var log logr.Logger

	log = klogr.New().V(0)
	ctx := logr.NewContext(context.Background(), log)

	app := &cli.App{
		Name:  "kubectl-azad-proxy",
		Usage: "kubectl plugin for azad-kube-proxy",
		Commands: []*cli.Command{
			{
				Name:    "generate",
				Aliases: []string{"g"},
				Usage:   "Generate kubeconfig",
				Flags:   actions.GenerateFlags(ctx),
				Action: func(c *cli.Context) error {
					cfg, err := actions.NewGenerateConfig(ctx, c)
					if err != nil {
						return err
					}
					return actions.Generate(ctx, cfg)
				},
			},
			{
				Name:    "login",
				Aliases: []string{"l"},
				Usage:   "Login to Azure AD app and return token",
				Flags:   actions.LoginFlags(ctx),
				Action: func(c *cli.Context) error {
					cfg, err := actions.NewLoginConfig(ctx, c)
					if err != nil {
						return err
					}

					output, err := actions.Login(ctx, cfg)
					if err != nil {
						return err
					}

					fmt.Print(output)
					return nil
				},
			},
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		os.Exit(1)
	}

	os.Exit(0)
}
