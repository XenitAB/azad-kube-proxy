package main

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewConfig(t *testing.T) {
	t.Run("no subcommand", func(t *testing.T) {
		args := []string{
			"/foo/bar/bin",
		}
		_, err := newConfig(args[1:])
		require.ErrorContains(t, err, "no valid subcommand provided")
	})

	t.Run("auth config", func(t *testing.T) {
		args := []string{
			"/foo/bar/bin",
			"discover",
		}
		cfg, err := newConfig(args[1:])
		require.NoError(t, err)
		require.NotNil(t, cfg.Discover)
		expectedAuthConifg := authConfig{
			excludeAzureCLIAuth:    false,
			excludeEnvironmentAuth: true,
			excludeMSIAuth:         true,
		}
		require.Equal(t, expectedAuthConifg, cfg.authConfig)
	})

	t.Run("discover", func(t *testing.T) {
		args := []string{
			"/foo/bar/bin",
			"discover",
		}
		cfg, err := newConfig(args[1:])
		require.NoError(t, err)
		require.NotNil(t, cfg.Discover)
		expectedDiscoverConfig := discoverConfig{
			Output:     "TABLE",
			AuthMethod: "CLI",
		}
		require.Equal(t, expectedDiscoverConfig, *cfg.Discover)
	})

	t.Run("generate", func(t *testing.T) {
		args := []string{
			"/foo/bar/bin",
			"generate",
		}
		_, err := newConfig(args[1:])
		require.Error(t, err)

		args = []string{
			"/foo/bar/bin",
			"generate",
			"--cluster-name",
			"ze-cluster",
		}
		_, err = newConfig(args[1:])
		require.Error(t, err)

		args = []string{
			"/foo/bar/bin",
			"generate",
			"--cluster-name",
			"ze-cluster",
			"--proxy-url",
			"ze-proxy-url",
		}
		_, err = newConfig(args[1:])
		require.Error(t, err)

		args = []string{
			"/foo/bar/bin",
			"generate",
			"--cluster-name",
			"ze-cluster",
			"--proxy-url",
			"ze-proxy-url",
			"--resource",
			"ze-resource",
		}
		cfg, err := newConfig(args[1:])
		require.NoError(t, err)
		require.NotNil(t, cfg.Generate)
		expectedGenerateConfig := generateConfig{
			ClusterName: "ze-cluster",
			ProxyURL:    "ze-proxy-url",
			Resource:    "ze-resource",
		}
		require.Equal(t, expectedGenerateConfig, *cfg.Generate)
	})

	t.Run("login", func(t *testing.T) {
		args := []string{
			"/foo/bar/bin",
			"login",
		}
		_, err := newConfig(args[1:])
		require.Error(t, err)

		args = []string{
			"/foo/bar/bin",
			"login",
			"--cluster-name",
			"ze-cluster",
		}
		_, err = newConfig(args[1:])
		require.Error(t, err)

		args = []string{
			"/foo/bar/bin",
			"login",
			"--cluster-name",
			"ze-cluster",
			"--resource",
			"ze-resource",
		}
		cfg, err := newConfig(args[1:])
		require.NoError(t, err)
		require.NotNil(t, cfg.Login)
		expectedLoginConfig := loginConfig{
			ClusterName: "ze-cluster",
			Resource:    "ze-resource",
		}
		require.Equal(t, expectedLoginConfig, *cfg.Login)
	})

	t.Run("menu", func(t *testing.T) {
		args := []string{
			"/foo/bar/bin",
			"menu",
		}
		cfg, err := newConfig(args[1:])
		require.NoError(t, err)
		require.NotNil(t, cfg.Menu)
		expectedMenuConfig := menuConfig{
			Output:     "TABLE",
			AuthMethod: "CLI",
		}
		require.Equal(t, expectedMenuConfig, *cfg.Menu)
	})
}
