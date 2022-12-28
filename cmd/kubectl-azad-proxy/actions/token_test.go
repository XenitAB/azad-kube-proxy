package actions

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/require"
)

func TestNewTokens(t *testing.T) {
	ctx := logr.NewContext(context.Background(), logr.Discard())

	opts := defaultAzureCredentialOptions{
		excludeAzureCLICredential:    true,
		excludeEnvironmentCredential: true,
		excludeMSICredential:         true,
	}

	t.Run("cache file doesn't exist", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "")
		require.NoError(t, err)
		defer os.RemoveAll(tmpDir)

		_, err = NewTokens(ctx, tmpDir, opts)
		require.NoError(t, err)
	})

	t.Run("cache exists but can't read it", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "")
		require.NoError(t, err)
		defer os.RemoveAll(tmpDir)

		tmpFilePath := fmt.Sprintf("%s/%s", tmpDir, tokenCacheFileName)
		_, err = os.Create(tmpFilePath)
		require.NoError(t, err)

		err = os.Chmod(tmpFilePath, 0000)
		require.NoError(t, err)

		_, err = NewTokens(ctx, tmpDir, opts)
		require.ErrorContains(t, err, "Token cache error: ")
	})

	t.Run("cache exists but wrong format", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "")
		require.NoError(t, err)
		defer os.RemoveAll(tmpDir)

		tmpFilePath := fmt.Sprintf("%s/%s", tmpDir, tokenCacheFileName)
		err = os.WriteFile(tmpFilePath, []byte("[]"), 0600)
		require.NoError(t, err)

		_, err = NewTokens(ctx, tmpDir, opts)
		require.ErrorContains(t, err, "Token cache error: ")
	})

	t.Run("cache exists", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "")
		require.NoError(t, err)
		defer os.RemoveAll(tmpDir)

		tmpFilePath := fmt.Sprintf("%s/%s", tmpDir, tokenCacheFileName)
		err = os.WriteFile(tmpFilePath, []byte("{}"), 0600)
		require.NoError(t, err)

		_, err = NewTokens(ctx, tmpDir, opts)
		require.NoError(t, err)
	})
}

func TestGetToken(t *testing.T) {
	tenantID := getEnvOrSkip(t, "TENANT_ID")
	clientID := getEnvOrSkip(t, "TEST_USER_SP_CLIENT_ID")
	clientSecret := getEnvOrSkip(t, "TEST_USER_SP_CLIENT_SECRET")
	resource := getEnvOrSkip(t, "TEST_USER_SP_RESOURCE")

	restoreTenantID := tempChangeEnv("AZURE_TENANT_ID", tenantID)
	defer restoreTenantID()

	restoreClientID := tempChangeEnv("AZURE_CLIENT_ID", clientID)
	defer restoreClientID()

	restoreClientSecret := tempChangeEnv("AZURE_CLIENT_SECRET", clientSecret)
	defer restoreClientSecret()

	ctx := logr.NewContext(context.Background(), logr.Discard())

	fakeHomeDir := "/home/test-user"
	restoreHomeDir := tempChangeEnv("HOME", fakeHomeDir)
	defer restoreHomeDir()

	creds := defaultAzureCredentialOptions{
		excludeAzureCLICredential:    true,
		excludeEnvironmentCredential: false,
		excludeMSICredential:         true,
	}

	credsFalse := defaultAzureCredentialOptions{
		excludeAzureCLICredential:    true,
		excludeEnvironmentCredential: true,
		excludeMSICredential:         true,
	}

	tmpDir, err := os.MkdirTemp("", "")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	fakeDir := fmt.Sprintf("%s/fake", tmpDir)
	err = os.Mkdir(fakeDir, 0700)
	require.NoError(t, err)

	fakeFile := fmt.Sprintf("%s/%s", fakeDir, tokenCacheFileName)
	fakeToken := make(map[string]Token)
	fakeToken["fake-cluster-1"] = Token{
		Token:               "abc123",
		ExpirationTimestamp: time.Now().Add(1 * time.Hour),
		Resource:            "https://fake-resource",
		Name:                "fake-cluster-1",
	}

	err = createCacheFile(fakeFile, fakeToken)
	require.NoError(t, err)

	fakeTokens, err := NewTokens(ctx, fakeDir, creds)
	require.NoError(t, err)

	realDir := fmt.Sprintf("%s/real", tmpDir)
	err = os.Mkdir(realDir, 0700)
	require.NoError(t, err)

	realTokens, err := NewTokens(ctx, realDir, creds)
	require.NoError(t, err)

	realFalseDir := fmt.Sprintf("%s/realfalse", tmpDir)
	err = os.Mkdir(realFalseDir, 0700)
	require.NoError(t, err)

	realFalseTokens, err := NewTokens(ctx, realFalseDir, credsFalse)
	require.NoError(t, err)

	cases := []struct {
		tokens              TokensInterface
		expectedErrContains string
		clusterName         string
		resource            string
	}{
		{
			tokens:              fakeTokens,
			expectedErrContains: "",
			clusterName:         "fake-cluster-1",
			resource:            "https://fake-resource",
		},
		{
			tokens:              realTokens,
			expectedErrContains: "",
			clusterName:         "real-cluster-1",
			resource:            resource,
		},
		{
			tokens:              realFalseTokens,
			expectedErrContains: "Authentication error:",
			clusterName:         "realfalse-cluster-1",
			resource:            resource,
		},
	}

	for _, c := range cases {
		token, err := c.tokens.GetToken(ctx, c.clusterName, c.resource)
		if c.expectedErrContains != "" {
			require.ErrorContains(t, err, c.expectedErrContains)
			continue
		}

		require.NoError(t, err)
		require.Equal(t, c.clusterName, token.Name)
		require.Equal(t, c.resource, token.Resource)
		require.NotEmpty(t, token.Token)
	}
}

func TestTokenExpired(t *testing.T) {
	cases := []struct {
		expiryDelta         time.Duration
		timeNow             time.Time
		expirationTimestamp time.Time
		expired             bool
	}{
		{
			expiryDelta:         10 * time.Second,
			timeNow:             time.Now(),
			expirationTimestamp: time.Now().Add(1 * time.Minute),
			expired:             false,
		},
		{
			expiryDelta:         10 * time.Second,
			timeNow:             time.Now(),
			expirationTimestamp: time.Now().Add(-1 * time.Minute),
			expired:             true,
		},
		{
			expiryDelta:         10 * time.Second,
			timeNow:             time.Now(),
			expirationTimestamp: time.Now().Add(60 * time.Minute),
			expired:             false,
		},
		{
			expiryDelta:         10 * time.Second,
			timeNow:             time.Now(),
			expirationTimestamp: time.Now().Add(-60 * time.Minute),
			expired:             true,
		},
		{
			expiryDelta:         10 * time.Second,
			timeNow:             time.Now(),
			expirationTimestamp: time.Now().Add(10 * time.Second),
			expired:             false,
		},
		{
			expiryDelta:         10 * time.Second,
			timeNow:             time.Now(),
			expirationTimestamp: time.Now().Add(-10 * time.Second),
			expired:             true,
		},
		{
			expiryDelta:         10 * time.Second,
			timeNow:             time.Now(),
			expirationTimestamp: time.Now().Add(9 * time.Second),
			expired:             true,
		},
		{
			expiryDelta:         10 * time.Second,
			timeNow:             time.Now(),
			expirationTimestamp: time.Now().Add(-9 * time.Second),
			expired:             true,
		},
	}

	for _, c := range cases {
		token := Token{
			Token:               "fake-token",
			ExpirationTimestamp: c.expirationTimestamp,
		}

		require.Equal(t, c.expired, token.expired(c.expiryDelta, c.timeNow))
	}
}

func TestTokenValid(t *testing.T) {
	cases := []struct {
		token *Token
		valid bool
	}{
		{
			token: &Token{
				Token:               "fake-token",
				ExpirationTimestamp: time.Now().Add(1 * time.Minute),
			},
			valid: true,
		},
		{
			token: &Token{
				Token:               "fake-token",
				ExpirationTimestamp: time.Now().Add(-1 * time.Minute),
			},
			valid: false,
		},
		{
			token: &Token{
				ExpirationTimestamp: time.Now().Add(1 * time.Minute),
			},
			valid: false,
		},
		{
			token: nil,
			valid: false,
		},
	}

	for _, c := range cases {
		require.Equal(t, c.valid, c.token.Valid())
	}
}

func tempChangeEnv(key, value string) func() {
	oldEnv := os.Getenv(key)
	os.Setenv(key, value)
	return func() { os.Setenv(key, oldEnv) }
}

func createCacheFile(path string, cachedTokens interface{}) error {
	fileContents, err := json.Marshal(&cachedTokens)
	if err != nil {
		return err
	}

	err = os.WriteFile(path, fileContents, 0600)
	if err != nil {
		return err
	}

	return nil
}

func deleteFile(t *testing.T, file string) {
	err := os.Remove(file)
	if err != nil {
		t.Errorf("Unable to delete file: %q", err)
	}
}

func getEnvOrSkip(t *testing.T, envVar string) string {
	v := os.Getenv(envVar)
	if v == "" {
		t.Skipf("%s environment variable is empty, skipping.", envVar)
	}

	return v
}
