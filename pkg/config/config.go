package config

import (
	"context"
	"crypto/x509"
	"fmt"
	"net"
	"net/url"
	"os"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/urfave/cli/v2"
	"github.com/xenitab/azad-kube-proxy/pkg/models"
	"github.com/xenitab/azad-kube-proxy/pkg/util"
)

// Config contains the configuration that is used for the application
type Config struct {
	ClientID               string `validate:"required,uuid"`
	ClientSecret           string `validate:"required,min=1"`
	TenantID               string `validate:"required,uuid"`
	ListenerAddress        string `validate:"hostname_port"`
	MetricsListenerAddress string `validate:"hostname_port"`
	ListenerTLSConfig      ListenerTLSConfig
	CacheEngine            models.CacheEngine
	RedisURI               string `validate:"uri"`
	AzureADGroupPrefix     string
	AzureADMaxGroupCount   int `validate:"min=1,max=1000"`
	GroupSyncInterval      time.Duration
	GroupIdentifier        models.GroupIdentifier
	KubernetesConfig       KubernetesConfig
	Dashboard              models.Dashboard
	Metrics                models.Metrics
	K8dashConfig           K8dashConfig
	CORSConfig             CORSConfig
}

// KubernetesConfig contains the Kubernetes specific configuration
type KubernetesConfig struct {
	URL                 *url.URL
	RootCA              *x509.CertPool
	RootCAString        string
	Token               string
	ValidateCertificate bool
}

// ListenerTLSConfig contains the TLS configuration for the listener
type ListenerTLSConfig struct {
	Enabled         bool
	CertificatePath string
	KeyPath         string
}

// K8dashConfig contains the configuration for the Dashboard k8dash
type K8dashConfig struct {
	ClientID     string
	ClientSecret string
	Scope        string
}

// CORSConfig contains the CORS configuration for the proxy
type CORSConfig struct {
	Enabled                     bool
	AllowedOriginsDefaultScheme string
	AllowedOrigins              []string
	AllowedHeaders              []string
	AllowedMethods              []string
}

// Flags returns a flag array
func Flags(ctx context.Context) []cli.Flag {
	return []cli.Flag{
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
		&cli.IntFlag{
			Name:     "metrics-port",
			Usage:    "Port number for metrics and health checks to listen on",
			Required: false,
			EnvVars:  []string{"METRICS_PORT"},
			Value:    8081,
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
		&cli.IntFlag{
			Name:     "group-sync-interval",
			Usage:    "The interval groups will be synchronized (in minutes)",
			Required: false,
			EnvVars:  []string{"GROUP_SYNC_INTERVAL"},
			Value:    5,
		},
		&cli.StringFlag{
			Name:     "group-identifier",
			Usage:    "What group identifier to use",
			Required: false,
			EnvVars:  []string{"GROUP_IDENTIFIER"},
			Value:    "NAME",
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
		&cli.StringFlag{
			Name:     "dashboard",
			Usage:    "What Kubernetes dashboard to use",
			Required: false,
			EnvVars:  []string{"DASHBOARD"},
			Value:    "NONE",
		},
		&cli.StringFlag{
			Name:     "metrics",
			Usage:    "What metrics library to use",
			Required: false,
			EnvVars:  []string{"METRICS"},
			Value:    "PROMETHEUS",
		},
		&cli.StringFlag{
			Name:     "k8dash-client-id",
			Usage:    "What Client ID to use with k8dash",
			Required: false,
			EnvVars:  []string{"K8DASH_CLIENT_ID"},
			Value:    "",
		},
		&cli.StringFlag{
			Name:     "k8dash-client-secret",
			Usage:    "What Client Secret to use with k8dash",
			Required: false,
			EnvVars:  []string{"K8DASH_CLIENT_SECRET"},
			Value:    "",
		},
		&cli.StringFlag{
			Name:     "k8dash-scope",
			Usage:    "What scope to use with k8dash",
			Required: false,
			EnvVars:  []string{"K8DASH_SCOPE"},
			Value:    "",
		},
		&cli.BoolFlag{
			Name:     "cors-enabled",
			Usage:    "Should CORS be enabled for the proxy?",
			Required: false,
			EnvVars:  []string{"CORS_ENABLED"},
			Value:    true,
		},
		&cli.StringSliceFlag{
			Name:     "cors-allowed-origins",
			Usage:    "The allowed origins for CORS (Access-Control-Allow-Origin). Defaults to the current host (based on host header - https://<host>).",
			Required: false,
			EnvVars:  []string{"CORS_ALLOWED_ORIGINS"},
		},
		&cli.StringFlag{
			Name:     "cors-allowed-origins-default-scheme",
			Usage:    "If cors-allowed-origins is left to default, what scheme should be used? (https for https://<host>)",
			Required: false,
			EnvVars:  []string{"CORS_ALLOWED_ORIGINS_DEFAULT_SCHEME"},
			Value:    "https",
		},
		&cli.StringSliceFlag{
			Name:     "cors-allowed-headers",
			Usage:    "The allowed headers for CORS (Access-Control-Allow-Headers). Defaults to: *",
			Required: false,
			EnvVars:  []string{"CORS_ALLOWED_HEADERS"},
		},
		&cli.StringSliceFlag{
			Name:     "cors-allowed-methods",
			Usage:    "The allowed methods for CORS (Access-Control-Allow-Methods). Defaults to: GET, HEAD, PUT, PATCH, POST, DELETE, OPTIONS",
			Required: false,
			EnvVars:  []string{"CORS_ALLOWED_METHODS"},
		},
	}
}

// NewConfig returns a Config or error
func NewConfig(ctx context.Context, cli *cli.Context) (Config, error) {
	kubernetesAPIUrl, err := getKubernetesAPIUrl(cli.String("kubernetes-api-host"), cli.Int("kubernetes-api-port"), cli.Bool("kubernetes-api-tls"))
	if err != nil {
		return Config{}, err
	}

	kubernetesRootCA, err := util.GetCertificate(ctx, cli.String("kubernetes-api-ca-cert-path"))
	if err != nil {
		return Config{}, err
	}

	kubernetesRootCAString, err := util.GetStringFromFile(ctx, cli.String("kubernetes-api-ca-cert-path"))
	if err != nil {
		return Config{}, err // NOTE FOR TESTS: Can't reach with test since it will always error before on kubernetesRootCA
	}

	kubernetesToken, err := util.GetStringFromFile(ctx, cli.String("kubernetes-api-token-path"))
	if err != nil {
		return Config{}, err
	}

	cacheEngine, err := models.GetCacheEngine(cli.String("cache-engine"))
	if err != nil {
		return Config{}, err
	}

	groupIdentifier, err := models.GetGroupIdentifier(cli.String("group-identifier"))
	if err != nil {
		return Config{}, err
	}

	dashboard, err := models.GetDashboard(cli.String("dashboard"))
	if err != nil {
		return Config{}, err
	}

	metrics, err := models.GetMetrics(cli.String("metrics"))
	if err != nil {
		return Config{}, err
	}

	config := Config{
		ClientID:               cli.String("client-id"),
		ClientSecret:           cli.String("client-secret"),
		TenantID:               cli.String("tenant-id"),
		ListenerAddress:        net.JoinHostPort(cli.String("address"), fmt.Sprintf("%d", cli.Int("port"))),
		MetricsListenerAddress: net.JoinHostPort(cli.String("address"), fmt.Sprintf("%d", cli.Int("metrics-port"))),
		ListenerTLSConfig: ListenerTLSConfig{
			Enabled:         cli.Bool("tls-enabled"),
			CertificatePath: cli.String("tls-certificate-path"),
			KeyPath:         cli.String("tls-key-path"),
		},
		CacheEngine:          cacheEngine,
		RedisURI:             cli.String("redis-uri"),
		AzureADGroupPrefix:   cli.String("azure-ad-group-prefix"),
		AzureADMaxGroupCount: cli.Int("azure-ad-max-group-count"),
		GroupSyncInterval:    time.Duration(cli.Int("group-sync-interval")) * time.Minute,
		GroupIdentifier:      groupIdentifier,
		KubernetesConfig: KubernetesConfig{
			URL:                 kubernetesAPIUrl,
			RootCA:              kubernetesRootCA,
			RootCAString:        kubernetesRootCAString,
			Token:               kubernetesToken,
			ValidateCertificate: cli.Bool("kubernetes-api-validate-cert"),
		},
		Dashboard: dashboard,
		Metrics:   metrics,
		K8dashConfig: K8dashConfig{
			ClientID:     cli.String("k8dash-client-id"),
			ClientSecret: cli.String("k8dash-client-secret"),
			Scope:        cli.String("k8dash-scope"),
		},
		CORSConfig: CORSConfig{
			Enabled:                     cli.Bool("cors-enabled"),
			AllowedOriginsDefaultScheme: cli.String("cors-allowed-origins-default-scheme"),
			AllowedOrigins:              cli.StringSlice("cors-allowed-origins"),
			AllowedHeaders:              cli.StringSlice("cors-allowed-headers"),
			AllowedMethods:              cli.StringSlice("cors-allowed-methods"),
		},
	}

	err = config.Validate()
	if err != nil {
		return Config{}, err
	}

	return config, nil
}

// Validate validates AppConfig struct
func (config Config) Validate() error {
	validate := validator.New()

	err := validate.Struct(config)
	if err != nil {
		return err
	}

	if config.ListenerTLSConfig.Enabled {
		if len(config.ListenerTLSConfig.CertificatePath) == 0 {
			return fmt.Errorf("config.ListenerTLSConfig.CertificatePath is not set")
		}
		if !fileExists(config.ListenerTLSConfig.CertificatePath) {
			return fmt.Errorf("config.ListenerTLSConfig.CertificatePath is not a file: %s", config.ListenerTLSConfig.CertificatePath)
		}
		if len(config.ListenerTLSConfig.KeyPath) == 0 {
			return fmt.Errorf("config.ListenerTLSConfig.KeyPath is not set")
		}
		if !fileExists(config.ListenerTLSConfig.KeyPath) {
			return fmt.Errorf("config.ListenerTLSConfig.KeyPath is not a file: %s", config.ListenerTLSConfig.KeyPath)
		}
	}

	if config.Dashboard == models.K8dashDashboard {
		if len(config.K8dashConfig.ClientID) == 0 {
			return fmt.Errorf("config.K8dashConfig.ClientID is not set")
		}
		if len(config.K8dashConfig.ClientSecret) == 0 {
			return fmt.Errorf("config.K8dashConfig.ClientSecret is not set")
		}
		if len(config.K8dashConfig.Scope) == 0 {
			return fmt.Errorf("config.K8dashConfig.Scope is not set")
		}
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

func fileExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}
