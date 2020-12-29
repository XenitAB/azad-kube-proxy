package config

import (
	"net/url"

	"github.com/go-playground/validator/v10"
)

// Config contains the configuration that is used for the application
type Config struct {
	ClientID                      string   `validate:"uuid"`
	TenantID                      string   `validate:"uuid"`
	ListnerAddress                string   `validate:"hostname_port"`
	KubernetesAPIUrl              *url.URL `validate:"url"`
	ValidateKubernetesCertificate bool
}

// Validate validates AppConfig struct
func (config Config) Validate() error {
	validate := validator.New()

	err := validate.Struct(config)
	if err != nil {
		return err
	}

	return nil
}
