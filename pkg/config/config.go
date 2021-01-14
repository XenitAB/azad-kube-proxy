package config

import (
	"context"
	"crypto/x509"
	"fmt"
	"net/url"
	"os"
	"strconv"

	"github.com/go-playground/validator/v10"
	flag "github.com/spf13/pflag"
	"github.com/xenitab/azad-kube-proxy/pkg/models"
	"github.com/xenitab/azad-kube-proxy/pkg/util"
)

// Config contains the configuration that is used for the application
type Config struct {
	ClientID             string `validate:"uuid"`
	ClientSecret         string
	TenantID             string `validate:"uuid"`
	ListenerAddress      string `validate:"hostname_port"`
	ListenerTLSConfig    ListenerTLSConfig
	CacheEngine          models.CacheEngine
	RedisURI             string `validate:"uri"`
	AzureADGroupPrefix   string
	AzureADMaxGroupCount int `validate:"min=1,max=1000"`
	KubernetesConfig     KubernetesConfig
}

// KubernetesConfig contains the Kubernetes specific configuration
type KubernetesConfig struct {
	URL                 *url.URL `validate:"url"`
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
func GetConfig(ctx context.Context) (Config, error) {
	clientID := getConfigString("client-id", "", []string{"CLIENT_ID"}, "Azure AD Application Client ID")
	clientSecret := getConfigString("client-secret", "", []string{"CLIENT_SECRET"}, "Azure AD Application Client Secret")
	tenantID := getConfigString("tenant-id", "", []string{"TENANT_ID"}, "Azure AD Tenant ID")
	address := getConfigString("address", "0.0.0.0", []string{"ADDRESS"}, "Address to listen on")
	port := getConfigInt("port", 8080, []string{"PORT"}, "Port number to listen on")
	tlsCertificatePath := getConfigString("tls-certificate-path", "", []string{"TLS_CERTIFICATE_PATH"}, "Path for the TLS Certificate")
	tlsKeyPath := getConfigString("tls-key-path", "", []string{"TLS_KEY_PATH"}, "Path for the TLS KEY")
	tlsEnabled := getConfigBool("tls-enabled", false, []string{"TLS_ENABLED"}, "Should TLS be enabled for the listner?")
	kubernetesAPIHost := getConfigString("kubernetes-api-host", "kubernetes.default", []string{"KUBERNETES_API_HOST", "KUBERNETES_SERVICE_HOST"}, "The host for the Kubernetes API")
	kubernetesAPIPort := getConfigInt("kubernetes-api-port", 443, []string{"KUBERNETES_API_PORT", "KUBERNETES_SERVICE_PORT"}, "The port for the Kubernetes API")
	kubernetesAPITLS := getConfigBool("kubernetes-api-tls", true, []string{"KUBERNETES_API_TLS"}, "Use TLS to communicate with the Kubernetes API?")
	kubernetesAPIValidateCert := getConfigBool("kubernetes-api-validate-cert", true, []string{"KUBERNETES_API_VALIDATE_CERT"}, "Should the Kubernetes API Certificate be validated?")
	kubernetesAPICACertPath := getConfigString("kubernetes-api-ca-cert-path", "/var/run/secrets/kubernetes.io/serviceaccount/ca.crt", []string{"KUBERNETES_API_CA_CERT_PATH"}, "The ca certificate path for communication to the Kubernetes API")
	kubernetesAPITokenPath := getConfigString("kubernetes-api-token-path", "/var/run/secrets/kubernetes.io/serviceaccount/token", []string{"KUBERNETES_API_TOKEN_PATH"}, "The token for communication to the Kubernetes API")
	azureADGroupPrefix := getConfigString("azure-ad-group-prefix", "", []string{"AZURE_AD_GROUP_PREFIX"}, "The prefix of the Azure AD groups to be passed to the Kubernetes API")
	azureADMaxGroupCount := getConfigInt("azure-ad-max-group-count", 50, []string{"AZURE_AD_MAX_GROUP_COUNT"}, "The maximum of groups allowed to be passed to the Kubernetes API before the proxy will return unauthorized")
	cacheEngine := getConfigString("cache-engine", "MEMORY", []string{"CACHE_ENGINE"}, "What cache engine to use")
	redisURI := getConfigString("redis-uri", "redis://127.0.0.1:6379/0", []string{"REDIS_URI"}, "The redis uri (redis://<user>:<password>@<host>:<port>/<db_number>)")

	kubernetesAPIUrl, err := getKubernetesAPIUrl(kubernetesAPIHost, kubernetesAPIPort, kubernetesAPITLS)
	if err != nil {
		return Config{}, err
	}

	kubernetesRootCA, err := util.GetCertificate(ctx, kubernetesAPICACertPath)
	if err != nil {
		return Config{}, err
	}

	kubernetesToken, err := util.GetStringFromFile(ctx, kubernetesAPITokenPath)
	if err != nil {
		return Config{}, err
	}

	cacheEng, err := models.GetCacheEngine(cacheEngine)
	if err != nil {
		return Config{}, err
	}

	config := Config{
		ClientID:        clientID,
		ClientSecret:    clientSecret,
		TenantID:        tenantID,
		ListenerAddress: fmt.Sprintf("%s:%d", address, port),
		ListenerTLSConfig: ListenerTLSConfig{
			Enabled:         tlsEnabled,
			CertificatePath: tlsCertificatePath,
			KeyPath:         tlsKeyPath,
		},
		CacheEngine:          cacheEng,
		RedisURI:             redisURI,
		AzureADGroupPrefix:   azureADGroupPrefix,
		AzureADMaxGroupCount: azureADMaxGroupCount,
		KubernetesConfig: KubernetesConfig{
			URL:                 kubernetesAPIUrl,
			RootCA:              kubernetesRootCA,
			Token:               kubernetesToken,
			ValidateCertificate: kubernetesAPIValidateCert,
		},
	}

	err = config.Validate()
	if err != nil {
		return Config{}, err
	}

	return config, nil
}

func getConfigString(name string, defaultValue string, envVars []string, description string) string {
	var flagResult string
	flag.StringVar(&flagResult, name, defaultValue, description)
	flag.Parse()
	if flag.Lookup(name).Changed {
		return flagResult
	}

	for _, env := range envVars {
		envResult := os.Getenv(env)
		if envResult != "" {
			return envResult
		}
	}

	return defaultValue
}

func getConfigInt(name string, defaultValue int, envVars []string, description string) int {
	var flagResult int
	flag.IntVar(&flagResult, name, defaultValue, description)
	flag.Parse()
	if flag.Lookup(name).Changed {
		return flagResult
	}

	for _, env := range envVars {
		envResult, err := strconv.Atoi(os.Getenv(env))
		if err == nil {
			return envResult
		}
	}

	return defaultValue
}

func getConfigBool(name string, defaultValue bool, envVars []string, description string) bool {
	var flagResult bool
	flag.BoolVar(&flagResult, name, defaultValue, description)
	flag.Parse()
	if flag.Lookup(name).Changed {
		return flagResult
	}

	for _, env := range envVars {
		envResult, err := strconv.ParseBool(os.Getenv(env))
		if err == nil {
			return envResult
		}
	}

	return defaultValue
}

// Validate validates AppConfig struct
func (config Config) Validate() error {
	validate := validator.New()

	err := validate.Struct(config)
	if err != nil {
		return err
	}

	err = validate.Struct(config.KubernetesConfig)
	if err != nil {
		return err
	}

	err = validate.Struct(config.ListenerTLSConfig)
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
