package proxy

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewConfig(t *testing.T) {
	envVarsToClear := []string{
		"AZURE_AD_GROUP_PREFIX",
		"AZURE_AD_MAX_GROUP_COUNT",
		"CLIENT_ID",
		"CLIENT_SECRET",
		"TENANT_ID",
		"CORS_ALLOWED_HEADERS",
		"CORS_ALLOWED_METHODS",
		"CORS_ALLOWED_ORIGINS",
		"CORS_ALLOWED_ORIGINS_DEFAULT_SCHEME",
		"CORS_ENABLED",
		"GROUP_IDENTIFIER",
		"GROUP_SYNC_INTERVAL",
		"KUBERNETES_API_CA_CERT_PATH",
		"KUBERNETES_API_HOST",
		"KUBERNETES_SERVICE_HOST",
		"KUBERNETES_API_PORT",
		"KUBERNETES_SERVICE_PORT",
		"KUBERNETES_API_TLS",
		"KUBERNETES_API_TOKEN_PATH",
		"KUBERNETES_API_VALIDATE_CERT",
		"ADDRESS",
		"PORT",
		"TLS_CERTIFICATE_PATH",
		"TLS_ENABLED",
		"TLS_KEY_PATH",
		"METRICS",
		"METRICS_ADDRESS",
		"METRICS_PORT",
	}

	for _, envVar := range envVarsToClear {
		restore := testTempUnsetEnv(t, envVar)
		defer restore()
	}

	t.Run("binary only", func(t *testing.T) {
		args := []string{
			"/foo/bar/bin",
		}
		_, err := NewConfig(args[1:], "", "", "")
		require.ErrorContains(t, err, "--client-id")
	})

	t.Run("populated", func(t *testing.T) {
		args := []string{
			"/foo/bar/bin",
			"--client-id=ze-client-id",
			"--client-secret=ze-client-secret",
			"--tenant-id=ze-tenant-id",
		}
		cfg, err := NewConfig(args[1:], "", "", "")
		require.NoError(t, err)
		expectedCfg := &config{
			AzureADMaxGroupCount:            50,
			AzureClientID:                   "ze-client-id",
			AzureClientSecret:               "ze-client-secret",
			AzureTenantID:                   "ze-tenant-id",
			CorsAllowedOriginsDefaultScheme: "https",
			CorsEnabled:                     true,
			GroupIdentifier:                 "NAME",
			GroupSyncInterval:               5,
			KubernetesAPICACertPath:         "/var/run/secrets/kubernetes.io/serviceaccount/ca.crt",
			KubernetesAPIHost:               "kubernetes.default",
			KubernetesAPIPort:               443,
			KubernetesAPITLS:                true,
			KubernetesAPITokenPath:          "/var/run/secrets/kubernetes.io/serviceaccount/token",
			KubernetesAPIValidateCert:       true,
			ListenerAddress:                 "0.0.0.0",
			ListenerPort:                    8080,
			Metrics:                         "PROMETHEUS",
			MetricsListenerAddress:          "0.0.0.0",
			MetricsListenerPort:             8081,
		}
		require.Equal(t, expectedCfg, cfg)
	})
}

func testTempUnsetEnv(t *testing.T, key string) func() {
	t.Helper()

	oldEnv := os.Getenv(key)
	os.Unsetenv(key)
	return func() { os.Setenv(key, oldEnv) }
}
