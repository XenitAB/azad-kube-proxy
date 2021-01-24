package main

import (
	"fmt"
	"log"
	"os"

	"github.com/urfave/cli/v2"
	"github.com/xenitab/azad-kube-proxy/cmd/kubectl-azad-proxy/actions"
)

func main() {
	app := &cli.App{
		Name:  "kubectl-azad-proxy",
		Usage: "kubectl plugin for azad-kube-proxy",
		Commands: []*cli.Command{
			{
				Name:    "generate",
				Aliases: []string{"g"},
				Usage:   "Generate kubeconfig",
				Flags:   actions.GenerateFlags(),
				Action: func(c *cli.Context) error {
					cfg, err := actions.NewGenerateConfig(c)
					if err != nil {
						return err
					}
					return actions.Generate(cfg)
				},
			},
			{
				Name:    "login",
				Aliases: []string{"l"},
				Usage:   "Login to Azure AD app and return token",
				Flags:   actions.LoginFlags(),
				Action: func(c *cli.Context) error {
					cfg, err := actions.NewLoginConfig(c)
					if err != nil {
						return err
					}

					output, err := actions.Login(cfg)
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
		log.Fatal(err)
		os.Exit(1)
	}

	os.Exit(0)
}
