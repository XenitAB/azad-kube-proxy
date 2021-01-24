package actions

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/go-logr/logr"
	logrTesting "github.com/go-logr/logr/testing"
)

func TestNewTokens(t *testing.T) {
	ctx := logr.NewContext(context.Background(), logrTesting.NullLogger{})

	fakeHomeDir := "/home/test-user"
	restoreHomeDir := tempChangeEnv("HOME", fakeHomeDir)
	defer restoreHomeDir()

	defaultAzureCredentialOptions := &azidentity.DefaultAzureCredentialOptions{
		ExcludeAzureCLICredential:    false,
		ExcludeEnvironmentCredential: false,
		ExcludeMSICredential:         false,
	}

	cases := []struct {
		path                string
		expectedErrContains string
		expectedPath        string
	}{
		{
			path:                "~/test",
			expectedErrContains: "",
			expectedPath:        fmt.Sprintf("%s/test", fakeHomeDir),
		},
		{
			path:                "~/.kube/abc",
			expectedErrContains: "",
			expectedPath:        fmt.Sprintf("%s/.kube/abc", fakeHomeDir),
		},
		{
			path:                "/home/test2/.kube/abc",
			expectedErrContains: "",
			expectedPath:        "/home/test2/.kube/abc",
		},
	}

	for _, c := range cases {
		tokens, err := NewTokens(ctx, c.path, defaultAzureCredentialOptions)
		if err != nil && c.expectedErrContains == "" {
			t.Errorf("Expected err to be nil: %q", err)
		}

		if err == nil && c.expectedErrContains != "" {
			t.Errorf("Expected err to contain '%s' but was nil", c.expectedErrContains)
		}

		if err != nil && c.expectedErrContains != "" {
			if !strings.Contains(err.Error(), c.expectedErrContains) {
				t.Errorf("Expected err to contain '%s' but was: %q", c.expectedErrContains, err)
			}
		}

		if !strings.Contains(tokens.GetPath(), c.expectedPath) {
			t.Errorf("Expected path to be '%s' but was: %s", c.expectedPath, tokens.GetPath())
		}
	}

	curDir, err := os.Getwd()
	if err != nil {
		t.Errorf("Expected err to be nil: %q", err)
	}
	fakeFile := fmt.Sprintf("%s/../../../tmp/test-cached-tokens", curDir)
	fakeToken := make(map[string]Token)
	fakeToken["test"] = Token{
		Token:               "abc123",
		ExpirationTimestamp: time.Now().Add(1 * time.Hour),
		Resource:            "https://fake-resource",
		Name:                "test",
	}

	err = createCacheFile(fakeFile, fakeToken)
	if err != nil {
		t.Errorf("Expected err to be nil: %q", err)
	}
	defer deleteFile(t, fakeFile)

	_, err = NewTokens(ctx, fakeFile, defaultAzureCredentialOptions)
	if err != nil {
		t.Errorf("Expected err to be nil: %q", err)
	}

	fakeFileErr := fmt.Sprintf("%s/../../../tmp/test-cached-tokens-err", curDir)
	fakeTokenErr := make(map[string]struct {
		FakeToken bool `json:"token"`
	})
	fakeTokenErr["test"] = struct {
		FakeToken bool `json:"token"`
	}{
		FakeToken: true,
	}
	err = createCacheFile(fakeFileErr, fakeTokenErr)
	if err != nil {
		t.Errorf("Expected err to be nil: %q", err)
	}
	defer deleteFile(t, fakeFileErr)

	_, err = NewTokens(ctx, fakeFileErr, defaultAzureCredentialOptions)
	if !strings.Contains(err.Error(), "json: cannot unmarshal bool") {
		t.Errorf("Expected err contain 'json: cannot unmarshal bool' but was: %q", err)
	}
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

	ctx := logr.NewContext(context.Background(), logrTesting.NullLogger{})

	fakeHomeDir := "/home/test-user"
	restoreHomeDir := tempChangeEnv("HOME", fakeHomeDir)
	defer restoreHomeDir()

	defaultAzureCredentialOptions := &azidentity.DefaultAzureCredentialOptions{
		ExcludeAzureCLICredential:    true,
		ExcludeEnvironmentCredential: false,
		ExcludeMSICredential:         true,
	}

	defaultAzureCredentialOptionsFalse := &azidentity.DefaultAzureCredentialOptions{
		ExcludeAzureCLICredential:    true,
		ExcludeEnvironmentCredential: true,
		ExcludeMSICredential:         true,
	}

	curDir, err := os.Getwd()
	if err != nil {
		t.Errorf("Expected err to be nil: %q", err)
	}
	fakeFile := fmt.Sprintf("%s/../../../tmp/test-cached-tokens-fake", curDir)
	fakeToken := make(map[string]Token)
	fakeToken["fake-cluster-1"] = Token{
		Token:               "abc123",
		ExpirationTimestamp: time.Now().Add(1 * time.Hour),
		Resource:            "https://fake-resource",
		Name:                "fake-cluster-1",
	}

	err = createCacheFile(fakeFile, fakeToken)
	if err != nil {
		t.Errorf("Expected err to be nil: %q", err)
	}
	defer deleteFile(t, fakeFile)

	fakeTokens, err := NewTokens(ctx, fakeFile, defaultAzureCredentialOptions)
	if err != nil {
		t.Errorf("Expected err to be nil: %q", err)
	}

	realFile := fmt.Sprintf("%s/../../../tmp/test-cached-tokens-real", curDir)
	realTokens, err := NewTokens(ctx, realFile, defaultAzureCredentialOptions)
	if err != nil {
		t.Errorf("Expected err to be nil: %q", err)
	}
	defer deleteFile(t, realFile)

	realFalseFile := fmt.Sprintf("%s/../../../tmp/test-cached-tokens-realfalse", curDir)
	realFalseTokens, err := NewTokens(ctx, realFalseFile, defaultAzureCredentialOptionsFalse)
	if err != nil {
		t.Errorf("Expected err to be nil: %q", err)
	}
	defer deleteFile(t, realFalseFile)

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
			expectedErrContains: "Default Azure Credential: ",
			clusterName:         "realfalse-cluster-1",
			resource:            resource,
		},
	}

	for _, c := range cases {
		token, err := c.tokens.GetToken(ctx, c.clusterName, c.resource)
		if err != nil && c.expectedErrContains == "" {
			t.Errorf("Expected err to be nil: %q", err)
		}

		if err == nil && c.expectedErrContains != "" {
			t.Errorf("Expected err to contain '%s' but was nil", c.expectedErrContains)
		}

		if err != nil && c.expectedErrContains != "" {
			if !strings.Contains(err.Error(), c.expectedErrContains) {
				t.Errorf("Expected err to contain '%s' but was: %q", c.expectedErrContains, err)
			}
		}

		if c.expectedErrContains == "" {
			if token.Name != c.clusterName {
				t.Errorf("Expected token.Name to be '%s' but was: %s", c.clusterName, token.Name)
			}
			if token.Resource != c.resource {
				t.Errorf("Expected token.Resource to be '%s' but was: %s", c.resource, token.Resource)
			}
			if len(token.Token) < 0 {
				t.Errorf("Expected token.Token to be larger than 0 but was: %d", len(token.Token))
			}
		}
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

	err = ioutil.WriteFile(path, fileContents, 0600)
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
