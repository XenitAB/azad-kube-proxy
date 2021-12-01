package actions

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	azpolicy "github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/go-logr/logr"
	"github.com/xenitab/azad-kube-proxy/cmd/kubectl-azad-proxy/customerrors"
)

type defaultAzureCredentialOptions struct {
	excludeAzureCLICredential    bool
	excludeEnvironmentCredential bool
	excludeMSICredential         bool
}

// Token contains the struct for a cached token
type Token struct {
	Token               string    `json:"token"`
	ExpirationTimestamp time.Time `json:"expirationTimestamp"`
	Resource            string    `json:"resource"`
	Name                string    `json:"name"`
}

func (t *Token) expired(expiryDelta time.Duration, timeNow time.Time) bool {
	if t.ExpirationTimestamp.IsZero() {
		return false
	}

	return t.ExpirationTimestamp.Round(0).Add(-expiryDelta).Before(timeNow)
}

// Valid reports whether t is non-nil, has an AccessToken, and is not expired.
func (t *Token) Valid() bool {
	expiryDelta := 10 * time.Second
	timeNow := time.Now

	return t != nil && t.Token != "" && !t.expired(expiryDelta, timeNow())
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
	defaultAzureCredentialOptions defaultAzureCredentialOptions
}

// NewTokens returns a TokensInterface or error
func NewTokens(ctx context.Context, path string, defaultAzureCredentialOptions defaultAzureCredentialOptions) (TokensInterface, error) {
	log := logr.FromContextOrDiscard(ctx)

	if strings.HasPrefix(path, "~/") {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			log.V(1).Info("Unable to get user home directory", "error", err.Error())
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
		log.V(1).Info("Unable to get file content", "error", err.Error(), "path", path)
		return nil, customerrors.New(customerrors.ErrorTypeTokenCache, err)
	}

	err = json.Unmarshal(fileContent, &t.cachedTokens)
	if err != nil {
		log.V(1).Info("Unable to unmarshal cachedTokens", "error", err.Error())
		return nil, customerrors.New(customerrors.ErrorTypeTokenCache, err)
	}

	return t, nil
}

// GetPath returns the path where the cached tokens are stored
func (t Tokens) GetPath() string {
	return t.path
}

// GetToken ...
func (t Tokens) GetToken(ctx context.Context, name string, resource string) (Token, error) {
	log := logr.FromContextOrDiscard(ctx)

	token, found := t.cachedTokens[name]

	generateNewToken := true
	if found {
		log.V(1).Info("Existing token found")

		if token.Valid() {
			log.V(1).Info("Token valid, no need to request new one")
			generateNewToken = false
		}
	}

	if generateNewToken {
		log.V(1).Info("New token will be requested")
		azureToken, err := getAccessToken(ctx, resource, t.defaultAzureCredentialOptions)
		if err != nil {
			log.V(1).Info("Unable to get access token", "error", err.Error(), "resource", resource)
			return Token{}, customerrors.New(customerrors.ErrorTypeAuthentication, err)
		}

		token = Token{
			Token:               azureToken.Token,
			ExpirationTimestamp: azureToken.ExpiresOn,
			Resource:            resource,
			Name:                name,
		}

		err = t.SetToken(ctx, name, token)
		if err != nil {
			return Token{}, customerrors.New(customerrors.ErrorTypeAuthentication, err)
		}

		return token, nil
	}

	return token, nil
}

// SetToken ...
func (t Tokens) SetToken(ctx context.Context, name string, token Token) error {
	log := logr.FromContextOrDiscard(ctx)

	t.cachedTokens[name] = token

	fileContents, err := json.Marshal(&t.cachedTokens)
	if err != nil {
		log.V(1).Info("Unable to marshal cachedTokens", "error", err.Error(), "name", name)
		return customerrors.New(customerrors.ErrorTypeTokenCache, err)
	}

	err = os.WriteFile(t.path, fileContents, 0600)
	if err != nil {
		log.V(1).Info("Unable to write token cache file", "error", err.Error(), "path", t.path)
		return customerrors.New(customerrors.ErrorTypeTokenCache, err)
	}

	return nil
}

func getAccessToken(ctx context.Context, resource string, defaultAzureCredentialOptions defaultAzureCredentialOptions) (*azcore.AccessToken, error) {
	scope := fmt.Sprintf("%s/.default", resource)
	cred, err := newDefaultAzureCredential(defaultAzureCredentialOptions)
	if err != nil {
		return nil, err
	}

	token, err := cred.GetToken(ctx, azpolicy.TokenRequestOptions{Scopes: []string{scope}})
	if err != nil {
		return nil, err
	}

	return token, nil
}

func newDefaultAzureCredential(options defaultAzureCredentialOptions) (*azidentity.ChainedTokenCredential, error) {
	creds := []azcore.TokenCredential{}
	opts := azidentity.DefaultAzureCredentialOptions{}

	var errMsg string

	if !options.excludeEnvironmentCredential {
		envCred, err := azidentity.NewEnvironmentCredential(&azidentity.EnvironmentCredentialOptions{AuthorityHost: opts.AuthorityHost,
			ClientOptions: opts.ClientOptions,
		})
		if err == nil {
			creds = append(creds, envCred)
		} else {
			errMsg += err.Error()
		}
	}

	if !options.excludeMSICredential {
		msiCred, err := azidentity.NewManagedIdentityCredential(&azidentity.ManagedIdentityCredentialOptions{ClientOptions: opts.ClientOptions})
		if err == nil {
			creds = append(creds, msiCred)
		} else {
			errMsg += err.Error()
		}
	}

	if !options.excludeAzureCLICredential {
		cliCred, err := azidentity.NewAzureCLICredential(&azidentity.AzureCLICredentialOptions{TenantID: opts.TenantID})
		if err == nil {
			creds = append(creds, cliCred)
		} else {
			errMsg += err.Error()
		}
	}

	if len(creds) == 0 {
		err := errors.New(errMsg)
		return nil, err
	}

	chain, err := azidentity.NewChainedTokenCredential(creds, nil)
	if err != nil {
		return nil, err
	}

	return chain, nil
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
