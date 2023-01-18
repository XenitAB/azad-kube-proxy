package proxy

import (
	"fmt"

	"github.com/alexflint/go-arg"
)

type Config struct {
	AzureADGroupPrefix               string   `arg:"--azure-ad-group-prefix,env:AZURE_AD_GROUP_PREFIX" help:"The prefix of the Azure AD groups to be passed to the Kubernetes API"`
	AzureADMaxGroupCount             int      `arg:"--azure-ad-max-group-count,env:AZURE_AD_MAX_GROUP_COUNT" default:"50" help:"The maximum of groups allowed to be passed to the Kubernetes API before the proxy will return unauthorized"`
	AzureClientID                    string   `arg:"--client-id,env:CLIENT_ID,required" help:"Azure AD Application Client ID"`
	AzureClientSecret                string   `arg:"--client-secret,env:CLIENT_SECRET,required" help:"Azure AD Application Client Secret"`
	AzureTenantID                    string   `arg:"--tenant-id,env:TENANT_ID,required" help:"Azure AD Tenant ID"`
	CorsAllowedHeaders               []string `arg:"--cors-allowed-headers,env:CORS_ALLOWED_HEADERS" help:"The allowed headers for CORS (Access-Control-Allow-Headers). Defaults to: *"`
	CorsAllowedMethods               []string `arg:"--cors-allowed-methods,env:CORS_ALLOWED_METHODS" help:"The allowed methods for CORS (Access-Control-Allow-Methods). Defaults to: GET, HEAD, PUT, PATCH, POST, DELETE, OPTIONS"`
	CorsAllowedOrigins               []string `arg:"--cors-allowed-origins,env:CORS_ALLOWED_ORIGINS" help:"The allowed origins for CORS (Access-Control-Allow-Origin). Defaults to the current host (based on host header - https://<host>)."`
	CorsAllowedOriginsDefaultScheme  string   `arg:"--cors-allowed-origins-default-scheme,env:CORS_ALLOWED_ORIGINS_DEFAULT_SCHEME" default:"https" help:"If cors-allowed-origins is left to default, what scheme should be used? (https for https://<host>)"`
	CorsEnabled                      bool     `arg:"--cors-enabled,env:CORS_ENABLED" default:"true" help:"Should CORS be enabled for the proxy?"`
	GroupIdentifier                  string   `arg:"--group-identifier,env:GROUP_IDENTIFIER" default:"NAME" help:"What group identifier to use"`
	GroupSyncInterval                int      `arg:"--group-sync-interval,env:GROUP_SYNC_INTERVAL" default:"5" help:"The interval groups will be synchronized (in minutes)"`
	KubernetesAPICACertPath          string   `arg:"--kubernetes-api-ca-cert-path,env:KUBERNETES_API_CA_CERT_PATH" default:"/var/run/secrets/kubernetes.io/serviceaccount/ca.crt" help:"The ca certificate path for communication to the Kubernetes API"`
	KubernetesAPIHost                string   `arg:"--kubernetes-api-host,env:KUBERNETES_API_HOST,env:KUBERNETES_SERVICE_HOST" default:"kubernetes.default" help:"The host for the Kubernetes API"`
	KubernetesAPIPort                int      `arg:"--kubernetes-api-port,env:KUBERNETES_API_PORT,env:KUBERNETES_SERVICE_PORT" default:"443" help:"The port for the Kubernetes API"`
	KubernetesAPITLS                 bool     `arg:"--kubernetes-api-tls,env:KUBERNETES_API_TLS" default:"true" help:"Use TLS to communicate with the Kubernetes API?"`
	KubernetesAPITokenPath           string   `arg:"--kubernetes-api-token-path,env:KUBERNETES_API_TOKEN_PATH" default:"/var/run/secrets/kubernetes.io/serviceaccount/token" help:"The token for communication to the Kubernetes API"`
	KubernetesAPIValidateCert        bool     `arg:"--kubernetes-api-validate-cert,env:KUBERNETES_API_VALIDATE_CERT" default:"true" help:"Should the Kubernetes API Certificate be validated?"`
	ListenerAddress                  string   `arg:"--address,env:ADDRESS" default:"0.0.0.0" help:"Address to listen on"`
	ListenerPort                     int      `arg:"--port,env:PORT" default:"8080" help:"Port number to listen on"`
	ListenerTLSConfigCertificatePath string   `arg:"--tls-certificate-path,env:TLS_CERTIFICATE_PATH" help:"Path for the TLS Certificate"`
	ListenerTLSConfigEnabled         bool     `arg:"--tls-enabled,env:TLS_ENABLED" default:"false" help:"Should TLS be enabled for the listner?"`
	ListenerTLSConfigKeyPath         string   `arg:"--tls-key-path,env:TLS_KEY_PATH" help:"Path for the TLS KEY"`
	Metrics                          string   `arg:"--metrics,env:METRICS" default:"PROMETHEUS" help:"What metrics library to use"`
	MetricsListenerAddress           string   `arg:"--metrics-address,env:METRICS_ADDRESS" default:"0.0.0.0" help:"Address to listen on"`
	MetricsListenerPort              int      `arg:"--metrics-port,env:METRICS_PORT" default:"8081" help:"Port number for metrics and health checks to listen on"`

	version  string
	revision string
	created  string
}

func (cfg Config) Version() string {
	return fmt.Sprintf("version=%s revision=%s created=%s\n", cfg.version, cfg.revision, cfg.created)
}

func NewConfig(args []string, version, revision, created string) (*Config, error) {
	cfg := &Config{
		version:  version,
		revision: revision,
		created:  created,
	}
	parser, err := arg.NewParser(arg.Config{
		Program:   "azad-kube-proxy - Azure AD Kubernetes API Proxy",
		IgnoreEnv: false,
	}, cfg)
	if err != nil {
		return &Config{}, err
	}

	err = parser.Parse(args)
	if err != nil {
		return &Config{}, err
	}

	return cfg, err
}
