package main

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewConfig(t *testing.T) {
	t.Run("no subcommend", func(t *testing.T) {
		args := []string{
			"/foo/bar/bin",
		}
		_, err := newConfig(args[1:])
		require.ErrorContains(t, err, "no valid subcommand provided")
	})
}
