package config

import (
	"crypto/x509"
	"net/url"

	"github.com/go-playground/validator/v10"
)

// Config contains the configuration that is used for the application
type Config struct {
	ClientID           string `validate:"uuid"`
	TenantID           string `validate:"uuid"`
	ListnerAddress     string `validate:"hostname_port"`
	AzureADGroupPrefix string
	KubernetesConfig   KubernetesConfig
}

// KubernetesConfig contains the Kubernetes specific configuration
type KubernetesConfig struct {
	URL                 *url.URL `validate:"url"`
	RootCA              *x509.CertPool
	Token               string
	ValidateCertificate bool
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

	return nil
}
