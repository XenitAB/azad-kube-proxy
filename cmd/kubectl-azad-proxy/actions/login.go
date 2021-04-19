package actions

import (
	"context"
	"encoding/json"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/urfave/cli/v2"
	k8smetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sclientauth "k8s.io/client-go/pkg/apis/clientauthentication/v1beta1"
)

// LoginClient ...
type LoginClient struct {
	clusterName                   string
	resource                      string
	tokenCache                    string
	defaultAzureCredentialOptions *azidentity.DefaultAzureCredentialOptions
}

// NewLoginClient ...
func NewLoginClient(ctx context.Context, c *cli.Context) (*LoginClient, error) {
	return &LoginClient{
		clusterName: c.String("cluster-name"),
		resource:    c.String("resource"),
		tokenCache:  c.String("token-cache"),
		defaultAzureCredentialOptions: &azidentity.DefaultAzureCredentialOptions{
			ExcludeAzureCLICredential:    c.Bool("exclude-azure-cli-auth"),
			ExcludeEnvironmentCredential: c.Bool("exclude-environment-auth"),
			ExcludeMSICredential:         c.Bool("exclude-msi-auth"),
		},
	}, nil
}

// LoginFlags ...
func LoginFlags(ctx context.Context) []cli.Flag {
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
			Name:    "token-cache",
			Usage:   "The token cache path to cache tokens",
			EnvVars: []string{"TOKEN_CACHE"},
			Value:   "~/.kube/azad-proxy.json",
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
	}
}

// Login ...
func (client *LoginClient) Login(ctx context.Context) (string, error) {
	tokens, err := NewTokens(ctx, client.tokenCache, client.defaultAzureCredentialOptions)
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
