package config

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"io/ioutil"
	"log"
	"math/big"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/go-logr/logr"
	logrTesting "github.com/go-logr/logr/testing"
)

// ./azad-kube-proxy --test abc --hejsan 123
// [./azad-kube-proxy --test abc --hejsan 123]
func TestGetConfig(t *testing.T) {
	ctx := logr.NewContext(context.Background(), logrTesting.NullLogger{})
	// Fake certificate
	_, err := GetConfig(ctx, []string{"fake-bin"})
	if !strings.Contains(err.Error(), "ca.crt: no such file or directory") {
		t.Errorf("Expected err to contain 'ca.crt: no such file or directory' but it was %q", err)
	}

	certPath, err := generateCertificateFile()
	if err != nil {
		t.Errorf("Unable to generate temporary certificate for test: %q", err)
	}
	defer deleteFile(t, certPath)

	_, err = GetConfig(ctx, []string{"fake-bin", fmt.Sprintf("--kubernetes-api-ca-cert-path=%s", certPath)})
	if !strings.Contains(err.Error(), "token: no such file or directory") {
		t.Errorf("Expected err to contain 'token: no such file or directory' but it was %q", err)
	}

	// Fake token
	tokenPath, _, err := generateRandomFile()
	if err != nil {
		t.Errorf("Unable to generate temporary file for test: %q", err)
	}
	defer deleteFile(t, tokenPath)

	baseArgs := []string{"fake-bin", fmt.Sprintf("--kubernetes-api-ca-cert-path=%s", certPath), fmt.Sprintf("--kubernetes-api-token-path=%s", tokenPath)}
	baseWorkingArgs := append(baseArgs, "--client-id=00000000-0000-0000-0000-000000000000", "--client-secret=supersecret", "--tenant-id=00000000-0000-0000-0000-000000000000")

	cases := []struct {
		osArgs                 []string
		expectedErrContains    []string
		expectedErrNotContains []string
	}{
		{
			osArgs:                 baseArgs,
			expectedErrContains:    []string{"Config.ClientID", "Config.ClientSecret", "Config.TenantID"},
			expectedErrNotContains: []string{},
		},
		{
			osArgs:                 append(baseArgs, "--client-id=00000000-0000-0000-0000-000000000000"),
			expectedErrContains:    []string{"Config.ClientSecret", "Config.TenantID"},
			expectedErrNotContains: []string{"Config.ClientID"},
		},
		{
			osArgs:                 append(baseArgs, "--client-id=00000000-0000-0000-0000-000000000000", "--client-secret=supersecret"),
			expectedErrContains:    []string{"Config.TenantID"},
			expectedErrNotContains: []string{"Config.ClientID", "Config.ClientSecret"},
		},
		{
			osArgs:                 append(baseArgs, "--client-id=00000000-0000-0000-0000-000000000000", "--client-secret=supersecret", "--tenant-id=00000000-0000-0000-0000-000000000000"),
			expectedErrContains:    []string{},
			expectedErrNotContains: []string{"Config.ClientID", "Config.ClientSecret", "Config.TenantID"},
		},
		{
			osArgs:                 append(baseWorkingArgs, "--address=this-shouldnt-work"),
			expectedErrContains:    []string{},
			expectedErrNotContains: []string{"Config.Address"},
		},
		{
			osArgs:                 append(baseWorkingArgs, "--does-not-exist"),
			expectedErrContains:    []string{"unknown flag: --does-not-exist"},
			expectedErrNotContains: []string{},
		},
		{
			osArgs:                 append(baseWorkingArgs, "--kubernetes-api-port=abc"),
			expectedErrContains:    []string{"parsing \"abc\": invalid syntax"},
			expectedErrNotContains: []string{},
		},
		{
			osArgs:                 append(baseWorkingArgs, "--kubernetes-api-host=\"a b c\""),
			expectedErrContains:    []string{"invalid character \" \" in host name"},
			expectedErrNotContains: []string{},
		},
		{
			osArgs:                 append(baseWorkingArgs, "--cache-engine=FAKE"),
			expectedErrContains:    []string{"Unkown cache engine type FAKE."},
			expectedErrNotContains: []string{},
		},
		{
			osArgs:                 append(baseWorkingArgs, "--cache-engine=FAKE"),
			expectedErrContains:    []string{"Unkown cache engine type FAKE."},
			expectedErrNotContains: []string{},
		},
		{
			osArgs:                 append(baseWorkingArgs, "--kubernetes-api-tls=false"),
			expectedErrContains:    []string{},
			expectedErrNotContains: []string{},
		},
		{
			osArgs:                 append(baseWorkingArgs, "--group-identifier=FAKE"),
			expectedErrContains:    []string{"Unkown group identifier FAKE."},
			expectedErrNotContains: []string{},
		},
	}

	for _, c := range cases {
		_, err := GetConfig(ctx, c.osArgs)
		if err != nil && len(c.expectedErrContains) == 0 {
			t.Errorf("Expected err to be nil but it was %q", err)
		}
		if err == nil && len(c.expectedErrContains) > 0 {
			t.Errorf("Expected err but it was nil")
		}
		if err != nil && len(c.expectedErrContains) > 0 {
			for _, s := range c.expectedErrContains {
				if !strings.Contains(err.Error(), s) {
					t.Errorf("Expected err to contain '%s' but it was %q", s, err)
				}
			}
		}
		if err != nil && len(c.expectedErrNotContains) > 0 {
			for _, s := range c.expectedErrNotContains {
				if strings.Contains(err.Error(), s) {
					t.Errorf("Expected err not to contain '%s' but it was %q", s, err)
				}
			}
		}
	}
}

func generateCertificateFile() (string, error) {
	timestamp := strconv.FormatInt(time.Now().UTC().UnixNano(), 10)
	filename := fmt.Sprintf("test-cert-%s.pem", timestamp)
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		log.Fatal(err)
	}

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
	if err != nil {
		return "", fmt.Errorf("Failed to create certificate: %v", err)
	}

	certOut, err := os.Create(filename)
	if err != nil {
		return "", fmt.Errorf("Failed to open %s for writing: %v", filename, err)
	}

	err = pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: certBytes})
	if err != nil {
		return "", fmt.Errorf("Failed to write data to %s: %v", filename, err)
	}

	err = certOut.Close()
	// if err != nil {
	// 	return "", fmt.Errorf("Error closing %s: %v", filename, err)
	// }

	return filename, nil
}

func TestConfigValidate(t *testing.T) {
	randomFile, _, err := generateRandomFile()
	if err != nil {
		t.Errorf("Unable to generate temporary file for test: %q", err)
	}
	defer deleteFile(t, randomFile)

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
		c.config.RedisURI = "redis://127.0.0.1:6379/0"
		c.config.AzureADMaxGroupCount = 50
		err := c.config.Validate()
		if !strings.Contains(err.Error(), c.expectedErrContains) {
			t.Errorf("Expected error to contain '%s' but was: %q", c.expectedErrContains, err)
		}
	}
}

func generateRandomFile() (string, string, error) {
	timestamp := strconv.FormatInt(time.Now().UTC().UnixNano(), 10)
	filename := fmt.Sprintf("test-random-%s.pem", timestamp)
	content := []byte(timestamp)

	err := ioutil.WriteFile(filename, content, 0644)
	if err != nil {
		return "", "", fmt.Errorf("Failed to create %s: %v", filename, err)
	}

	return filename, timestamp, nil
}

func deleteFile(t *testing.T, file string) {
	err := os.Remove(file)
	if err != nil {
		t.Errorf("Unable to delete file: %q", err)
	}
}
