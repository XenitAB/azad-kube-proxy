package config

import (
	"crypto/x509"
	"net/url"

	"github.com/go-playground/validator/v10"
	"github.com/xenitab/azad-kube-proxy/pkg/models"
)

// Config contains the configuration that is used for the application
type Config struct {
	ClientID             string `validate:"uuid"`
	ClientSecret         string
	TenantID             string `validate:"uuid"`
	ListenerAddress      string `validate:"hostname_port"`
	ListenerTLSConfig    ListenerTLSConfig
	CacheEngine          models.CacheEngine
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
