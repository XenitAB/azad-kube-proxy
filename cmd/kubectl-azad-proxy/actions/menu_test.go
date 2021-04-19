package actions

import (
	"bytes"
	"context"
	"testing"

	"github.com/go-logr/logr"
	"github.com/urfave/cli/v2"
)

func TestNewMenuConfig(t *testing.T) {
	ctx := logr.NewContext(context.Background(), logr.DiscardLogger{})

	restoreAzureCLIAuth := tempChangeEnv("EXCLUDE_AZURE_CLI_AUTH", "true")
	defer restoreAzureCLIAuth()

	app := &cli.App{
		Name:  "test",
		Usage: "test",
		Commands: []*cli.Command{
			{
				Name:    "test",
				Aliases: []string{"t"},
				Usage:   "test",
				Flags:   MenuFlags(ctx),
				Action: func(c *cli.Context) error {
					_, err := NewMenuConfig(ctx, c)
					if err != nil {
						return err
					}
					return nil
				},
			},
		},
	}

	app.Writer = &bytes.Buffer{}
	app.ErrWriter = &bytes.Buffer{}
	err := app.Run([]string{"fake-binary", "test"})
	if err != nil {
		t.Errorf("Expected err to be nil: %q", err)
	}
}

func TestMergeFlags(t *testing.T) {
	cases := []struct {
		a              []cli.Flag
		b              []cli.Flag
		expectedLength int
	}{
		{
			a:              []cli.Flag{},
			b:              []cli.Flag{},
			expectedLength: 0,
		},
		{
			a: []cli.Flag{
				&cli.StringFlag{
					Name: "flag1",
				},
			},
			b: []cli.Flag{
				&cli.StringFlag{
					Name: "flag2",
				},
			},
			expectedLength: 2,
		},
		{
			a: []cli.Flag{
				&cli.StringFlag{
					Name: "flag1",
				},
			},
			b: []cli.Flag{
				&cli.StringFlag{
					Name: "flag1",
				},
			},
			expectedLength: 1,
		},
		{
			a: []cli.Flag{
				&cli.StringFlag{
					Name: "flag1",
				},
				&cli.StringFlag{
					Name: "flag2",
				},
			},
			b: []cli.Flag{
				&cli.StringFlag{
					Name: "flag1",
				},
				&cli.StringFlag{
					Name: "flag2",
				},
			},
			expectedLength: 2,
		},
		{
			a: []cli.Flag{
				&cli.StringFlag{
					Name: "flag1",
				},
				&cli.StringFlag{
					Name: "flag2",
				},
			},
			b: []cli.Flag{
				&cli.StringFlag{
					Name: "flag1",
				},
				&cli.StringFlag{
					Name: "flag2",
				},
				&cli.StringFlag{
					Name: "flag3",
				},
			},
			expectedLength: 3,
		},
		{
			a: []cli.Flag{
				&cli.StringFlag{
					Name: "flag1",
				},
				&cli.StringFlag{
					Name: "flag2",
				},
				&cli.StringFlag{
					Name: "flag3",
				},
			},
			b: []cli.Flag{
				&cli.StringFlag{
					Name: "flag1",
				},
				&cli.StringFlag{
					Name: "flag2",
				},
			},
			expectedLength: 3,
		},
	}

	for _, c := range cases {
		flags := mergeFlags(c.a, c.b)
		if len(flags) != c.expectedLength {
			t.Errorf("Expected flags length to be '%d' but was: %d", c.expectedLength, len(flags))
		}
	}
}

func TestUnrequireFlags(t *testing.T) {
	cases := []struct {
		a []cli.Flag
	}{
		{
			a: []cli.Flag{},
		},
		{
			a: []cli.Flag{
				&cli.StringFlag{
					Name:     "flag1",
					Required: true,
				},
			},
		},
		{
			a: []cli.Flag{
				&cli.StringFlag{
					Name:     "flag1",
					Required: true,
				},
				&cli.StringFlag{
					Name:     "flag2",
					Required: true,
				},
			},
		},
		{
			a: []cli.Flag{
				&cli.StringFlag{
					Name:     "flag1",
					Required: true,
				},
				&cli.StringFlag{
					Name:     "flag2",
					Required: false,
				},
			},
		},
		{
			a: []cli.Flag{
				&cli.StringFlag{
					Name:     "flag1",
					Required: false,
				},
				&cli.StringFlag{
					Name:     "flag2",
					Required: false,
				},
				&cli.StringFlag{
					Name:     "flag3",
					Required: true,
				},
			},
		},
	}

	for _, c := range cases {
		flags := unrequireFlags(c.a)
		for _, flag := range flags {
			if flag.(*cli.StringFlag).Required {
				t.Errorf("Expected flag to be 'false' but was 'true'")
			}
		}
	}
}
