package claims

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	azpolicy "github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/coreos/go-oidc"
	"github.com/go-logr/logr"
)

func TestNewClaims(t *testing.T) {
	clientID := getEnvOrSkip(t, "CLIENT_ID")
	tenantID := getEnvOrSkip(t, "TENANT_ID")
	spClientID := getEnvOrSkip(t, "TEST_USER_SP_CLIENT_ID")
	spClientSecret := getEnvOrSkip(t, "TEST_USER_SP_CLIENT_SECRET")
	spResource := getEnvOrSkip(t, "TEST_USER_SP_RESOURCE")
	ctx := logr.NewContext(context.Background(), logr.Discard())
	claimsClient := NewClaimsClient()

	verifier, err := claimsClient.GetOIDCVerifier(ctx, tenantID, clientID)
	if err != nil {
		t.Errorf("Expected err to be nil but it was %q", err)
	}

	token, err := getAccessToken(ctx, tenantID, spClientID, spClientSecret, fmt.Sprintf("%s/.default", spResource))
	if err != nil {
		t.Errorf("Expected err to be nil but it was %q", err)
	}

	verifiedToken, err := verifier.Verify(ctx, token.Token)
	if err != nil {
		t.Errorf("Expected err to be nil but it was %q", err)
	}

	claims, err := claimsClient.NewClaims(verifiedToken)
	if err != nil {
		t.Errorf("Expected err to be nil but it was %q", err)
	}

	if claims.ApplicationID != spClientID {
		t.Errorf("Returned ApplicationID wasn't what was expected.\nExpected: %s\nActual:  %s", spClientID, spClientID)
	}

	_, err = claimsClient.NewClaims(nil)
	if !strings.Contains(err.Error(), "Token nil") {
		t.Errorf("Expected err to contain 'Token nil': %q", err)
	}

	_, err = claimsClient.NewClaims(&oidc.IDToken{})
	if !strings.Contains(err.Error(), "oidc: claims not set") {
		t.Errorf("Expected err to contain 'oidc: claims not set': %q", err)
	}
}

func TestGetOIDCVerifier(t *testing.T) {
	clientID := getEnvOrSkip(t, "CLIENT_ID")
	tenantID := getEnvOrSkip(t, "TENANT_ID")
	ctx := logr.NewContext(context.Background(), logr.Discard())
	claimsClient := NewClaimsClient()

	cases := []struct {
		tenantID            string
		clientID            string
		expectedErrContains string
	}{
		{
			tenantID:            tenantID,
			clientID:            clientID,
			expectedErrContains: "",
		},
		{
			tenantID:            "",
			clientID:            "",
			expectedErrContains: "AADSTS90002",
		},
	}

	for _, c := range cases {
		_, err := claimsClient.GetOIDCVerifier(ctx, c.tenantID, c.clientID)
		if err != nil && len(c.expectedErrContains) == 0 {
			t.Errorf("Expected err to be nil but it was %q", err)
		}

		if len(c.expectedErrContains) > 0 {
			if !strings.Contains(err.Error(), c.expectedErrContains) {
				t.Errorf("Expected err to contain %q but it was %q", c.expectedErrContains, err)
			}
		}
	}
}

func getEnvOrSkip(t *testing.T, envVar string) string {
	v := os.Getenv(envVar)
	if v == "" {
		t.Skipf("%s environment variable is empty, skipping.", envVar)
	}

	return v
}

func getAccessToken(ctx context.Context, tenantID, clientID, clientSecret, scope string) (*azcore.AccessToken, error) {
	tokenFilePath := fmt.Sprintf("../../tmp/test-token-file_%s", clientID)
	tokenFileExists := fileExists(tokenFilePath)
	token := &azcore.AccessToken{}

	generateNewToken := true
	if tokenFileExists {
		fileContent, err := getFileContent(tokenFilePath)
		if err != nil {
			return nil, err
		}

		err = json.Unmarshal(fileContent, &token)
		if err != nil {
			return nil, err
		}

		if token.ExpiresOn.After(time.Now().Add(-5 * time.Minute)) {
			generateNewToken = false
		}
	}

	if generateNewToken {
		cred, err := azidentity.NewClientSecretCredential(tenantID, clientID, clientSecret, nil)
		if err != nil {
			return nil, err
		}

		token, err := cred.GetToken(ctx, azpolicy.TokenRequestOptions{Scopes: []string{scope}})
		if err != nil {
			return nil, err
		}

		fileContents, err := json.Marshal(&token)
		if err != nil {
			return nil, err
		}

		err = os.WriteFile(tokenFilePath, fileContents, 0644)
		if err != nil {
			return nil, err
		}

		return token, nil
	}

	return token, nil
}

func fileExists(s string) bool {
	_, err := os.Stat(s)
	return err == nil
}

func getFileContent(s string) ([]byte, error) {
	file, err := os.Open(s)
	if err != nil {
		return nil, err
	}

	defer file.Close()

	bytes, err := io.ReadAll(file)
	if err != nil {
		return nil, err
	}

	return bytes, nil
}
