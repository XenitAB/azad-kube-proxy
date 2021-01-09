package app

import (
	"context"
	"fmt"
	"net/url"

	"github.com/urfave/cli/v2"
	"github.com/xenitab/azad-kube-proxy/pkg/config"
	"github.com/xenitab/azad-kube-proxy/pkg/models"
	"github.com/xenitab/azad-kube-proxy/pkg/proxy"
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
			EnvVars:  []string{"PORT"},
			Value:    8080,
		},
		&cli.StringFlag{
			Name:     "tls-certificate-path",
			Usage:    "Path for the TLS Certificate",
			Required: false,
			EnvVars:  []string{"TLS_CERTIFICATE_PATH"},
			Value:    "",
		},
		&cli.StringFlag{
			Name:     "tls-key-path",
			Usage:    "Path for the TLS KEY",
			Required: false,
			EnvVars:  []string{"TLS_KEY_PATH"},
			Value:    "",
		},
		&cli.BoolFlag{
			Name:     "tls-enabled",
			Usage:    "Should TLS be enabled for the listner?",
			Required: false,
			EnvVars:  []string{"TLS_ENABLED"},
			Value:    false,
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
		&cli.IntFlag{
			Name:     "azure-ad-max-group-count",
			Usage:    "The maximum of groups allowed to be passed to the Kubernetes API before the proxy will return unauthorized",
			Required: false,
			EnvVars:  []string{"AZURE_AD_MAX_GROUP_COUNT"},
			Value:    50,
		},
		&cli.StringFlag{
			Name:     "cache-engine",
			Usage:    "What cache engine to use",
			Required: false,
			EnvVars:  []string{"CACHE_ENGINE"},
			Value:    "MEMORY",
		},
		&cli.StringFlag{
			Name:     "redis-uri",
			Usage:    "The redis uri (redis://<user>:<password>@<host>:<port>/<db_number>)",
			Required: false,
			EnvVars:  []string{"REDIS_URI"},
			Value:    "redis://127.0.0.1:6379/0",
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

	cacheEngine, err := models.GetCacheEngine(cli.String("cache-engine"))
	if err != nil {
		return err
	}

	config := config.Config{
		ClientID:        cli.String("client-id"),
		ClientSecret:    cli.String("client-secret"),
		TenantID:        cli.String("tenant-id"),
		ListenerAddress: fmt.Sprintf("%s:%d", cli.String("address"), cli.Int("port")),
		ListenerTLSConfig: config.ListenerTLSConfig{
			Enabled:         cli.Bool("tls-enabled"),
			CertificatePath: cli.String("tls-certificate-path"),
			KeyPath:         cli.String("tls-key-path"),
		},
		CacheEngine:          cacheEngine,
		RedisURI:             cli.String("redis-uri"),
		AzureADGroupPrefix:   cli.String("azure-ad-group-prefix"),
		AzureADMaxGroupCount: cli.Int("azure-ad-max-group-count"),
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

	err = proxy.Start(ctx, config)
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
