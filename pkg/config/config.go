package config

import (
	"context"
	"crypto/x509"
	"fmt"
	"net/url"
	"os"

	"github.com/go-playground/validator/v10"
	flag "github.com/spf13/pflag"
	"github.com/xenitab/azad-kube-proxy/pkg/models"
	"github.com/xenitab/azad-kube-proxy/pkg/util"
)

// Config contains the configuration that is used for the application
type Config struct {
	ClientID             string `validate:"required,uuid"`
	ClientSecret         string `validate:"required,min=1"`
	TenantID             string `validate:"required,uuid"`
	ListenerAddress      string `validate:"hostname_port"`
	ListenerTLSConfig    ListenerTLSConfig
	CacheEngine          models.CacheEngine
	RedisURI             string `validate:"uri"`
	AzureADGroupPrefix   string
	AzureADMaxGroupCount int `validate:"min=1,max=1000"`
	GroupIdentifier      models.GroupIdentifier
	KubernetesConfig     KubernetesConfig
}

// KubernetesConfig contains the Kubernetes specific configuration
type KubernetesConfig struct {
	URL                 *url.URL
	RootCA              *x509.CertPool
	Token               string
	ValidateCertificate bool
}

// ListenerTLSConfig contains the TLS configuration for the listener
type ListenerTLSConfig struct {
	Enabled         bool
	CertificatePath string
	KeyPath         string
}

// GetConfig returns the configuration or an error
func GetConfig(ctx context.Context, args []string) (Config, error) {
	fs := flag.NewFlagSet("azad-kube-proxy", flag.ContinueOnError)

	clientID := fs.String("client-id", "", "Azure AD Application Client ID")
	clientSecret := fs.String("client-secret", "", "Azure AD Application Client Secret")
	tenantID := fs.String("tenant-id", "", "Azure AD Tenant ID")
	address := fs.String("address", "0.0.0.0", "Address to listen on")
	port := fs.Int("port", 8080, "Port number to listen on")
	tlsCertificatePath := fs.String("tls-certificate-path", "", "Path for the TLS Certificate")
	tlsKeyPath := fs.String("tls-key-path", "", "Path for the TLS KEY")
	tlsEnabled := fs.Bool("tls-enabled", false, "Should TLS be enabled for the listner?")
	kubernetesAPIHost := fs.String("kubernetes-api-host", "kubernetes.default", "The host for the Kubernetes API")
	kubernetesAPIPort := fs.Int("kubernetes-api-port", 443, "The port for the Kubernetes API")
	kubernetesAPITLS := fs.Bool("kubernetes-api-tls", true, "Use TLS to communicate with the Kubernetes API?")
	kubernetesAPIValidateCert := fs.Bool("kubernetes-api-validate-cert", true, "Should the Kubernetes API Certificate be validated?")
	kubernetesAPICACertPath := fs.String("kubernetes-api-ca-cert-path", "/var/run/secrets/kubernetes.io/serviceaccount/ca.crt", "The ca certificate path for communication to the Kubernetes API")
	kubernetesAPITokenPath := fs.String("kubernetes-api-token-path", "/var/run/secrets/kubernetes.io/serviceaccount/token", "The token for communication to the Kubernetes API")
	azureADGroupPrefix := fs.String("azure-ad-group-prefix", "", "The prefix of the Azure AD groups to be passed to the Kubernetes API")
	azureADMaxGroupCount := fs.Int("azure-ad-max-group-count", 50, "The maximum of groups allowed to be passed to the Kubernetes API before the proxy will return unauthorized")
	groupIdentifier := fs.String("group-identifier", "NAME", "What group identifier to use")
	cacheEngine := fs.String("cache-engine", "MEMORY", "What cache engine to use")
	redisURI := fs.String("redis-uri", "redis://127.0.0.1:6379/0", "The redis uri (redis://<user>:<password>@<host>:<port>/<db_number>)")

	err := fs.Parse(args)
	if err != nil {
		return Config{}, err
	}

	kubernetesAPIUrl, err := getKubernetesAPIUrl(*kubernetesAPIHost, *kubernetesAPIPort, *kubernetesAPITLS)
	if err != nil {
		return Config{}, err
	}

	kubernetesRootCA, err := util.GetCertificate(ctx, *kubernetesAPICACertPath)
	if err != nil {
		return Config{}, err
	}

	kubernetesToken, err := util.GetStringFromFile(ctx, *kubernetesAPITokenPath)
	if err != nil {
		return Config{}, err
	}

	gIdentifier, err := models.GetGroupIdentifier(*groupIdentifier)
	if err != nil {
		return Config{}, err
	}

	cacheEng, err := models.GetCacheEngine(*cacheEngine)
	if err != nil {
		return Config{}, err
	}

	config := Config{
		ClientID:        *clientID,
		ClientSecret:    *clientSecret,
		TenantID:        *tenantID,
		ListenerAddress: fmt.Sprintf("%s:%d", *address, *port),
		ListenerTLSConfig: ListenerTLSConfig{
			Enabled:         *tlsEnabled,
			CertificatePath: *tlsCertificatePath,
			KeyPath:         *tlsKeyPath,
		},
		CacheEngine:          cacheEng,
		RedisURI:             *redisURI,
		AzureADGroupPrefix:   *azureADGroupPrefix,
		AzureADMaxGroupCount: *azureADMaxGroupCount,
		GroupIdentifier:      gIdentifier,
		KubernetesConfig: KubernetesConfig{
			URL:                 kubernetesAPIUrl,
			RootCA:              kubernetesRootCA,
			Token:               kubernetesToken,
			ValidateCertificate: *kubernetesAPIValidateCert,
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
	info, err = os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}
