package app

import (
	"context"
	"fmt"
	"net/url"

	"github.com/urfave/cli/v2"
	"github.com/xenitab/azad-kube-proxy/pkg/config"
	"github.com/xenitab/azad-kube-proxy/pkg/reverseproxy"
	"github.com/xenitab/azad-kube-proxy/pkg/util"
)

// Get return the main app
func Get(ctx context.Context) *cli.App {
	app := &cli.App{
		Name:  "azad-kube-proxy",
		Usage: "Azure AD reverse proxy for Kubernetes API",
		Flags: flags(),
		Action: func(cli *cli.Context) error {
			err := action(ctx, cli)
			if err != nil {
				return err
			}
			return nil
		},
	}

	return app
}

func flags() []cli.Flag {
	flags := []cli.Flag{
		&cli.StringFlag{
			Name:     "client-id",
			Usage:    "Azure AD Application Client ID",
			Required: true,
			EnvVars:  []string{"CLIENT_ID"},
		},
		&cli.StringFlag{
			Name:     "client-secret",
			Usage:    "Azure AD Application Client Secret",
			Required: true,
			EnvVars:  []string{"CLIENT_SECRET"},
		},
		&cli.StringFlag{
			Name:     "tenant-id",
			Usage:    "Azure AD Tenant ID",
			Required: true,
			EnvVars:  []string{"TENANT_ID"},
		},
		&cli.StringFlag{
			Name:     "address",
			Usage:    "Address to listen on",
			Required: false,
			EnvVars:  []string{"ADDRESS"},
			Value:    "0.0.0.0",
		},
		&cli.IntFlag{
			Name:     "port",
			Usage:    "Port number to listen on",
			Required: false,
			EnvVars:  []string{"Port"},
			Value:    8080,
		},
		&cli.BoolFlag{
			Name:     "oidc-validate-cert",
			Usage:    "Should the OpenID Connect CA Certificate be validated?",
			Required: false,
			EnvVars:  []string{"OIDC_VALIDATE_CERT"},
			Value:    true,
		},
		&cli.StringFlag{
			Name:     "kubernetes-api-host",
			Usage:    "The host for the Kubernetes API",
			Required: false,
			EnvVars:  []string{"KUBERNETES_API_HOST", "KUBERNETES_SERVICE_HOST"},
			Value:    "kubernetes.default",
		},
		&cli.IntFlag{
			Name:     "kubernetes-api-port",
			Usage:    "The port for the Kubernetes API",
			Required: false,
			EnvVars:  []string{"KUBERNETES_API_PORT", "KUBERNETES_SERVICE_PORT"},
			Value:    443,
		},
		&cli.BoolFlag{
			Name:     "kubernetes-api-tls",
			Usage:    "Use TLS to communicate with the Kubernetes API?",
			Required: false,
			EnvVars:  []string{"KUBERNETES_API_TLS"},
			Value:    true,
		},
		&cli.BoolFlag{
			Name:     "kubernetes-api-validate-cert",
			Usage:    "Should the Kubernetes API Certificate be validated?",
			Required: false,
			EnvVars:  []string{"KUBERNETES_API_VALIDATE_CERT"},
			Value:    true,
		},
		&cli.StringFlag{
			Name:     "kubernetes-api-ca-cert-path",
			Usage:    "The ca certificate path for communication to the Kubernetes API",
			Required: false,
			EnvVars:  []string{"KUBERNETES_API_CA_CERT_PATH"},
			Value:    "/var/run/secrets/kubernetes.io/serviceaccount/ca.crt",
		},
		&cli.StringFlag{
			Name:     "kubernetes-api-token-path",
			Usage:    "The token for communication to the Kubernetes API",
			Required: false,
			EnvVars:  []string{"KUBERNETES_API_TOKEN_PATH"},
			Value:    "/var/run/secrets/kubernetes.io/serviceaccount/token",
		},
		&cli.StringFlag{
			Name:     "azure-ad-group-prefix",
			Usage:    "The prefix of the Azure AD groups to be passed to the Kubernetes API",
			Required: false,
			EnvVars:  []string{"AZURE_AD_GROUP_PREFIX"},
			Value:    "",
		},
	}

	return flags
}

func action(ctx context.Context, cli *cli.Context) error {
	kubernetesAPIUrl, err := getKubernetesAPIUrl(cli.String("kubernetes-api-host"), cli.Int("kubernetes-api-port"), cli.Bool("kubernetes-api-tls"))
	if err != nil {
		return err
	}

	kubernetesRootCA, err := util.GetCertificate(ctx, cli.String("kubernetes-api-ca-cert-path"))
	if err != nil {
		return err
	}

	kubernetesToken, err := util.GetStringFromFile(ctx, cli.String("kubernetes-api-token-path"))
	if err != nil {
		return err
	}

	config := config.Config{
		ClientID:           cli.String("client-id"),
		ClientSecret:       cli.String("client-secret"),
		TenantID:           cli.String("tenant-id"),
		ListnerAddress:     fmt.Sprintf("%s:%d", cli.String("address"), cli.Int("port")),
		AzureADGroupPrefix: cli.String("azure-ad-group-prefix"),
		KubernetesConfig: config.KubernetesConfig{
			URL:                 kubernetesAPIUrl,
			RootCA:              kubernetesRootCA,
			Token:               kubernetesToken,
			ValidateCertificate: cli.Bool("kubernetes-api-validate-cert"),
		},
	}

	err = config.Validate()
	if err != nil {
		return err
	}

	err = reverseproxy.Start(ctx, config)
	if err != nil {
		return err
	}

	return nil
}

func getKubernetesAPIUrl(host string, port int, tls bool) (*url.URL, error) {
	httpScheme := getHTTPScheme(tls)
	return url.Parse(fmt.Sprintf("%s://%s:%d", httpScheme, host, port))
}

func getHTTPScheme(tls bool) string {
	if tls {
		return "https"
	}

	return "http"
}
