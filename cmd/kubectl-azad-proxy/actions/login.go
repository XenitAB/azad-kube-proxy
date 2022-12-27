package actions

import (
	"context"
	"encoding/json"
	"fmt"
	"os/user"
	"path/filepath"

	"github.com/urfave/cli/v2"
	k8smetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sclientauth "k8s.io/client-go/pkg/apis/clientauthentication/v1beta1"
)

// LoginClient ...
type LoginClient struct {
	clusterName                   string
	resource                      string
	tokenCacheDir                 string
	defaultAzureCredentialOptions defaultAzureCredentialOptions
}

// LoginInterface ...
type LoginInterface interface {
	Login(ctx context.Context) (string, error)
}

// NewLoginClient ...
func NewLoginClient(ctx context.Context, c *cli.Context) (LoginInterface, error) {
	tokenCacheDir := getTokenCacheDirectory(c.String("token-cache-dir"), c.String("kubeconfig"))
	return &LoginClient{
		clusterName:   c.String("cluster-name"),
		resource:      c.String("resource"),
		tokenCacheDir: tokenCacheDir,
		defaultAzureCredentialOptions: defaultAzureCredentialOptions{
			excludeAzureCLICredential:    c.Bool("exclude-azure-cli-auth"),
			excludeEnvironmentCredential: c.Bool("exclude-environment-auth"),
			excludeMSICredential:         c.Bool("exclude-msi-auth"),
		},
	}, nil
}

func getTokenCacheDirectory(tokenCacheDirectory, kubeConfig string) string {
	if tokenCacheDirectory != "" {
		return filepath.Clean(tokenCacheDirectory)
	}

	if kubeConfig != "" {
		return filepath.Dir(kubeConfig)
	}

	userHomeDir := "~"
	usr, err := user.Current()
	if err == nil {
		userHomeDir = usr.HomeDir
	}

	return filepath.Clean(fmt.Sprintf("%s/.kube", userHomeDir))
}

// LoginFlags ...
func LoginFlags(ctx context.Context) ([]cli.Flag, error) {
	usr, err := user.Current()
	if err != nil {
		return nil, err
	}

	return []cli.Flag{
		&cli.StringFlag{
			Name:     "cluster-name",
			Usage:    "The name of the Kubernetes cluster / context",
			EnvVars:  []string{"CLUSTER_NAME"},
			Required: true,
		},
		&cli.StringFlag{
			Name:     "resource",
			Usage:    "The Azure AD App URI / resource",
			EnvVars:  []string{"RESOURCE"},
			Required: true,
		},
		&cli.StringFlag{
			Name:    "token-cache-dir",
			Usage:   "The directory to where the tokens are cached, defaults to the same as KUBECONFIG",
			EnvVars: []string{"TOKEN_CACHE_DIR"},
		},
		&cli.StringFlag{
			Name:     "kubeconfig",
			Usage:    "The path of the Kubernetes Config",
			EnvVars:  []string{"KUBECONFIG"},
			Value:    fmt.Sprintf("%s/.kube/config", usr.HomeDir),
			Required: false,
		},
		&cli.BoolFlag{
			Name:    "exclude-azure-cli-auth",
			Usage:   "Should Azure CLI be excluded from the authentication?",
			EnvVars: []string{"EXCLUDE_AZURE_CLI_AUTH"},
			Value:   false,
		},
		&cli.BoolFlag{
			Name:    "exclude-environment-auth",
			Usage:   "Should environment be excluded from the authentication?",
			EnvVars: []string{"EXCLUDE_ENVIRONMENT_AUTH"},
			Value:   true,
		},
		&cli.BoolFlag{
			Name:    "exclude-msi-auth",
			Usage:   "Should MSI be excluded from the authentication?",
			EnvVars: []string{"EXCLUDE_MSI_AUTH"},
			Value:   true,
		},
	}, nil
}

// Login ...
func (client *LoginClient) Login(ctx context.Context) (string, error) {
	tokens, err := NewTokens(ctx, client.tokenCacheDir, client.defaultAzureCredentialOptions)
	if err != nil {
		return "", err
	}

	token, err := tokens.GetToken(ctx, client.clusterName, client.resource)
	if err != nil {
		return "", err
	}

	execCredential := &k8sclientauth.ExecCredential{
		TypeMeta: k8smetav1.TypeMeta{
			APIVersion: "client.authentication.k8s.io/v1beta1",
			Kind:       "ExecCredential",
		},
		Status: &k8sclientauth.ExecCredentialStatus{
			Token:               token.Token,
			ExpirationTimestamp: &k8smetav1.Time{Time: token.ExpirationTimestamp},
		},
	}

	res, err := json.Marshal(execCredential)
	if err != nil {
		return "", err
	}

	return string(res), nil
}
