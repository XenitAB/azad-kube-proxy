package main

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewConfig(t *testing.T) {
	envVarsToClear := []string{
		"AZURE_CLIENT_ID",
		"AZURE_CLIENT_SECRET",
		"AZURE_TENANT_ID",
		"CLUSTER_NAME",
		"EXCLUDE_AZURE_CLI_AUTH",
		"EXCLUDE_ENVIRONMENT_AUTH",
		"EXCLUDE_MSI_AUTH",
		"KUBECONFIG",
		"OUTPUT",
		"OVERWRITE_KUBECONFIG",
		"PROXY_URL",
		"RESOURCE",
		"TLS_INSECURE_SKIP_VERIFY",
		"TOKEN_CACHE_DIR",
	}

	for _, envVar := range envVarsToClear {
		restore := testTempUnsetEnv(t, envVar)
		defer restore()
	}
	userHomeDir, err := os.UserHomeDir()
	require.NoError(t, err)
	defaultKubeConfig := filepath.Clean(fmt.Sprintf("%s/.kube/config", userHomeDir))

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
			Output: "TABLE",
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
			KubeConfig:  defaultKubeConfig,
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
			KubeConfig:  defaultKubeConfig,
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
			KubeConfig: defaultKubeConfig,
		}
		require.Equal(t, expectedMenuConfig, *cfg.Menu)
	})
}

func testTempUnsetEnv(t *testing.T, key string) func() {
	t.Helper()

	oldEnv := os.Getenv(key)
	os.Unsetenv(key)
	return func() { os.Setenv(key, oldEnv) }
}
