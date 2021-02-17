package actions

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/go-logr/logr"
)

// Token contains the struct for a cached token
type Token struct {
	Token               string    `json:"token"`
	ExpirationTimestamp time.Time `json:"expirationTimestamp"`
	Resource            string    `json:"resource"`
	Name                string    `json:"name"`
}

// TokensInterface is the interface for the Tokens struct
type TokensInterface interface {
	GetPath() string
	GetToken(ctx context.Context, name string, resource string) (Token, error)
	SetToken(ctx context.Context, name string, token Token) error
}

// Tokens contains the token struct
type Tokens struct {
	cachedTokens                  map[string]Token
	path                          string
	defaultAzureCredentialOptions *azidentity.DefaultAzureCredentialOptions
}

// NewTokens returns a TokensInterface or error
func NewTokens(ctx context.Context, path string, defaultAzureCredentialOptions *azidentity.DefaultAzureCredentialOptions) (TokensInterface, error) {
	log := logr.FromContext(ctx)

	if strings.HasPrefix(path, "~/") {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			log.Error(err, "Unable to get user home directory")
		}
		path = strings.Replace(path, "~/", fmt.Sprintf("%s/", homeDir), 1)
	}

	t := Tokens{
		cachedTokens:                  make(map[string]Token),
		path:                          path,
		defaultAzureCredentialOptions: defaultAzureCredentialOptions,
	}
	cacheFileExists := fileExists(path)

	if !cacheFileExists {
		return t, nil
	}

	fileContent, err := getFileContent(path)
	if err != nil {
		log.Error(err, "Unable to get file content", "path", path)
		return nil, err
	}

	err = json.Unmarshal(fileContent, &t.cachedTokens)
	if err != nil {
		log.Error(err, "Unable to unmarshal cachedTokens")
		return nil, err
	}

	return t, nil
}

// GetPath returns the path where the cached tokens are stored
func (t Tokens) GetPath() string {
	return t.path
}

// GetToken ...
func (t Tokens) GetToken(ctx context.Context, name string, resource string) (Token, error) {
	log := logr.FromContext(ctx)

	token, found := t.cachedTokens[name]

	generateNewToken := true
	if found {
		if token.ExpirationTimestamp.After(time.Now().Add(-5 * time.Minute)) {
			generateNewToken = false
		}
	}

	if generateNewToken {
		azureToken, err := getAccessToken(ctx, resource, t.defaultAzureCredentialOptions)
		if err != nil {
			log.Error(err, "Unable to get access token", "resource", resource)
			return Token{}, err
		}

		token = Token{
			Token:               azureToken.Token,
			ExpirationTimestamp: azureToken.ExpiresOn,
			Resource:            resource,
			Name:                name,
		}

		err = t.SetToken(ctx, name, token)
		if err != nil {
			return Token{}, err
		}

		return token, nil
	}

	return token, nil
}

// SetToken ...
func (t Tokens) SetToken(ctx context.Context, name string, token Token) error {
	log := logr.FromContext(ctx)

	t.cachedTokens[name] = token

	fileContents, err := json.Marshal(&t.cachedTokens)
	if err != nil {
		log.Error(err, "Unable to marshal cachedTokens", "name", name)
		return err
	}

	err = os.WriteFile(t.path, fileContents, 0600)
	if err != nil {
		log.Error(err, "Unable to write token cache file", "path", t.path)
		return err
	}

	return nil
}

func getAccessToken(ctx context.Context, resource string, defaultAzureCredentialOptions *azidentity.DefaultAzureCredentialOptions) (*azcore.AccessToken, error) {
	scope := fmt.Sprintf("%s/.default", resource)
	cred, err := azidentity.NewDefaultAzureCredential(defaultAzureCredentialOptions)
	if err != nil {
		return nil, err
	}

	token, err := cred.GetToken(ctx, azcore.TokenRequestOptions{Scopes: []string{scope}})
	if err != nil {
		return nil, err
	}

	return token, nil
}

func getFileContent(s string) ([]byte, error) {
	file, err := os.Open(s) // #nosec
	if err != nil {
		return nil, err
	}

	defer file.Close() // #nosec

	bytes, err := io.ReadAll(file)
	if err != nil {
		return nil, err
	}

	return bytes, nil
}

func fileExists(s string) bool {
	_, err := os.Stat(s)
	return err == nil
}
