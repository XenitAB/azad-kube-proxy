package config

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/require"
	"github.com/urfave/cli/v2"
)

func TestNewConfig(t *testing.T) {
	ctx := logr.NewContext(context.Background(), logr.Discard())
	cfg := Config{}

	envVarsToClear := []string{
		"CLIENT_ID",
		"CLIENT_SECRET",
		"TENANT_ID",
		"KUBERNETES_API_CA_CERT_PATH",
		"KUBERNETES_API_TOKEN_PATH",
		"TLS_CERTIFICATE_PATH",
		"TLS_KEY_PATH",
		"TLS_ENABLED",
		"METRICS",
	}

	for _, envVar := range envVarsToClear {
		restore := testTempUnsetEnv(t, envVar)
		defer restore()
	}

	app := &cli.App{
		Name:  "test",
		Usage: "test",
		Flags: Flags(ctx),
		Action: func(c *cli.Context) error {
			var err error
			cfg, err = NewConfig(ctx, c)
			if err != nil {
				return err
			}
			return nil
		},
	}

	// Fake certificate
	certPath := testGenerateCertificateFile(t)
	defer testDeleteFile(t, certPath)

	// Fake token
	tokenPath, _, err := testGenerateRandomFile(t)
	require.NoError(t, err)
	defer testDeleteFile(t, tokenPath)

	baseArgs := []string{"fake-bin", fmt.Sprintf("--kubernetes-api-ca-cert-path=%s", certPath), fmt.Sprintf("--kubernetes-api-token-path=%s", tokenPath)}
	baseWorkingArgs := append(baseArgs, "--client-id=00000000-0000-0000-0000-000000000000", "--client-secret=supersecret", "--tenant-id=00000000-0000-0000-0000-000000000000")

	cases := []struct {
		cliApp              *cli.App
		args                []string
		expectedConfig      Config
		expectedErrContains string
		outBuffer           bytes.Buffer
		errBuffer           bytes.Buffer
	}{
		{
			cliApp:              app,
			args:                baseArgs,
			expectedConfig:      Config{},
			expectedErrContains: "client-id",
			outBuffer:           bytes.Buffer{},
			errBuffer:           bytes.Buffer{},
		},
		{
			cliApp:              app,
			args:                append(baseArgs, "--client-id=00000000-0000-0000-0000-000000000000"),
			expectedConfig:      Config{},
			expectedErrContains: "client-secret",
			outBuffer:           bytes.Buffer{},
			errBuffer:           bytes.Buffer{},
		},
		{
			cliApp:              app,
			args:                append(baseArgs, "--client-id=00000000-0000-0000-0000-000000000000", "--client-secret=supersecret"),
			expectedConfig:      Config{},
			expectedErrContains: "tenant-id",
			outBuffer:           bytes.Buffer{},
			errBuffer:           bytes.Buffer{},
		},
		{
			cliApp: app,
			args:   append(baseArgs, "--client-id=00000000-0000-0000-0000-000000000000", "--client-secret=supersecret", "--tenant-id=00000000-0000-0000-0000-000000000000"),
			expectedConfig: Config{
				ClientID: "00000000-0000-0000-0000-000000000000",
			},
			expectedErrContains: "",
			outBuffer:           bytes.Buffer{},
			errBuffer:           bytes.Buffer{},
		},
		{
			cliApp: app,
			args:   append(baseWorkingArgs, "--address=this-shouldnt-work"),
			expectedConfig: Config{
				ClientID: "00000000-0000-0000-0000-000000000000",
			},
			expectedErrContains: "",
			outBuffer:           bytes.Buffer{},
			errBuffer:           bytes.Buffer{},
		},
		{
			cliApp:              app,
			args:                append(baseWorkingArgs, "--does-not-exist"),
			expectedConfig:      Config{},
			expectedErrContains: "flag provided but not defined: -does-not-exist",
			outBuffer:           bytes.Buffer{},
			errBuffer:           bytes.Buffer{},
		},
		{
			cliApp:              app,
			args:                append(baseWorkingArgs, "--kubernetes-api-port=abc"),
			expectedConfig:      Config{},
			expectedErrContains: "invalid value \"abc\" for flag -kubernetes-api-port: parse error",
			outBuffer:           bytes.Buffer{},
			errBuffer:           bytes.Buffer{},
		},
		{
			cliApp:              app,
			args:                append(baseWorkingArgs, "--kubernetes-api-host=\"a b c\""),
			expectedConfig:      Config{},
			expectedErrContains: "invalid character \" \" in host name",
			outBuffer:           bytes.Buffer{},
			errBuffer:           bytes.Buffer{},
		},
		{
			cliApp: app,
			args:   append(baseWorkingArgs, "--kubernetes-api-tls=false"),
			expectedConfig: Config{
				ClientID: "00000000-0000-0000-0000-000000000000",
			},
			expectedErrContains: "",
			outBuffer:           bytes.Buffer{},
			errBuffer:           bytes.Buffer{},
		},
		{
			cliApp:              app,
			args:                append(baseWorkingArgs, "--group-identifier=FAKE"),
			expectedConfig:      Config{},
			expectedErrContains: "Unknown group identifier 'FAKE'.",
			outBuffer:           bytes.Buffer{},
			errBuffer:           bytes.Buffer{},
		},
		{
			cliApp:              app,
			args:                append(baseWorkingArgs, "--metrics=FAKE"),
			expectedConfig:      Config{},
			expectedErrContains: "Unknown metrics 'FAKE'.",
			outBuffer:           bytes.Buffer{},
			errBuffer:           bytes.Buffer{},
		},
		{
			cliApp:              app,
			args:                append(baseArgs, "--client-id=123", "--client-secret=supersecret", "--tenant-id=00000000-0000-0000-0000-000000000000"),
			expectedConfig:      Config{},
			expectedErrContains: "Key: 'Config.ClientID' Error:Field validation for 'ClientID' failed on the 'uuid' tag",
			outBuffer:           bytes.Buffer{},
			errBuffer:           bytes.Buffer{},
		},
		{
			cliApp:              app,
			args:                append(baseWorkingArgs, "--tls-enabled=TRUE"),
			expectedConfig:      Config{},
			expectedErrContains: "config.ListenerTLSConfig.CertificatePath is not set",
			outBuffer:           bytes.Buffer{},
			errBuffer:           bytes.Buffer{},
		},
	}

	for _, c := range cases {
		cfg = Config{}
		c.cliApp.Writer = &c.outBuffer
		c.cliApp.ErrWriter = &c.errBuffer
		err := c.cliApp.Run(c.args)
		if c.expectedErrContains != "" {
			require.ErrorContains(t, err, c.expectedErrContains)
			continue
		}

		require.NoError(t, err)
		require.Equal(t, c.expectedConfig.ClientID, cfg.ClientID)
	}
}

func TestConfigValidate(t *testing.T) {
	randomFile, _, err := testGenerateRandomFile(t)
	require.NoError(t, err)
	defer testDeleteFile(t, randomFile)

	cases := []struct {
		config              Config
		expectedErrContains string
	}{
		{
			config: Config{
				ListenerTLSConfig: ListenerTLSConfig{
					Enabled:         true,
					CertificatePath: "",
					KeyPath:         "",
				},
			},
			expectedErrContains: "config.ListenerTLSConfig.CertificatePath is not set",
		},
		{
			config: Config{
				ListenerTLSConfig: ListenerTLSConfig{
					Enabled:         true,
					CertificatePath: "/this/should/not/exist",
					KeyPath:         "",
				},
			},
			expectedErrContains: "config.ListenerTLSConfig.CertificatePath is not a file",
		},
		{
			config: Config{
				ListenerTLSConfig: ListenerTLSConfig{
					Enabled:         true,
					CertificatePath: randomFile,
					KeyPath:         "",
				},
			},
			expectedErrContains: "config.ListenerTLSConfig.KeyPath is not set",
		},
		{
			config: Config{
				ListenerTLSConfig: ListenerTLSConfig{
					Enabled:         true,
					CertificatePath: randomFile,
					KeyPath:         "/this/should/not/exist",
				},
			},
			expectedErrContains: "config.ListenerTLSConfig.KeyPath is not a file",
		},
	}

	for _, c := range cases {
		c.config.ClientID = "00000000-0000-0000-0000-000000000000"
		c.config.TenantID = "00000000-0000-0000-0000-000000000000"
		c.config.ClientSecret = "00000000-0000-0000-0000-000000000000"
		c.config.ListenerAddress = "0.0.0.0:8080"
		c.config.MetricsListenerAddress = "0.0.0.0:8081"
		c.config.AzureADMaxGroupCount = 50
		err := c.config.Validate()
		require.ErrorContains(t, err, c.expectedErrContains)
	}
}

func testGenerateCertificateFile(t *testing.T) string {
	t.Helper()

	timestamp := strconv.FormatInt(time.Now().UTC().UnixNano(), 10)
	filename := fmt.Sprintf("test-cert-%s.pem", timestamp)
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			Organization: []string{"Testing"},
		},
		NotBefore: time.Now(),
		NotAfter:  time.Now().Add(time.Hour * 24 * 180),

		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}

	certBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &priv.PublicKey, priv)
	require.NoError(t, err)

	certOut, err := os.Create(filename)
	require.NoError(t, err)

	err = pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: certBytes})
	require.NoError(t, err)

	err = certOut.Close()
	require.NoError(t, err)

	return filename
}

func testGenerateRandomFile(t *testing.T) (string, string, error) {
	t.Helper()

	timestamp := strconv.FormatInt(time.Now().UTC().UnixNano(), 10)
	filename := fmt.Sprintf("test-random-%s.pem", timestamp)
	content := []byte(timestamp)

	err := os.WriteFile(filename, content, 0600)
	require.NoError(t, err)

	return filename, timestamp, nil
}

func testDeleteFile(t *testing.T, file string) {
	t.Helper()

	err := os.Remove(file)
	require.NoError(t, err)
}

func testTempUnsetEnv(t *testing.T, key string) func() {
	t.Helper()

	oldEnv := os.Getenv(key)
	os.Unsetenv(key)
	return func() { os.Setenv(key, oldEnv) }
}
