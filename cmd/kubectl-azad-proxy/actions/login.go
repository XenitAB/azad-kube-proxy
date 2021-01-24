package actions

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/urfave/cli/v2"
	k8smetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sclientauth "k8s.io/client-go/pkg/apis/clientauthentication/v1beta1"
)

// LoginConfig ...
type LoginConfig struct {
	resource                      string
	defaultAzureCredentialOptions *azidentity.DefaultAzureCredentialOptions
}

// NewLoginConfig ...
func NewLoginConfig(c *cli.Context) (LoginConfig, error) {
	return LoginConfig{
		resource: c.String("resource"),
		defaultAzureCredentialOptions: &azidentity.DefaultAzureCredentialOptions{
			ExcludeAzureCLICredential:    c.Bool("exclude-azure-cli-auth"),
			ExcludeEnvironmentCredential: c.Bool("exclude-environment-auth"),
			ExcludeMSICredential:         c.Bool("exclude-msi-auth"),
		},
	}, nil
}

// LoginFlags ...
func LoginFlags() []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{
			Name:     "resource",
			Usage:    "The Azure AD App URI / resource",
			EnvVars:  []string{"RESOURCE"},
			Required: true,
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
			Value:   false,
		},
		&cli.BoolFlag{
			Name:    "exclude-msi-auth",
			Usage:   "Should MSI be excluded from the authentication?",
			EnvVars: []string{"EXCLUDE_MSI_AUTH"},
			Value:   false,
		},
	}
}

// Login ...
func Login(cfg LoginConfig) (string, error) {
	ctx := context.Background()

	token, err := getAccessToken(ctx, cfg.resource, cfg.defaultAzureCredentialOptions)
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
			ExpirationTimestamp: &k8smetav1.Time{Time: token.ExpiresOn},
		},
	}

	res, err := json.Marshal(execCredential)
	if err != nil {
		return "", err
	}

	return string(res), nil
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
