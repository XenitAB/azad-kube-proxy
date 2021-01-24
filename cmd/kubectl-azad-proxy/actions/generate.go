package actions

import (
	"context"
	"crypto/tls"
	"encoding/pem"
	"fmt"
	"net"
	"net/url"
	"os/user"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/urfave/cli/v2"
	k8sclientcmd "k8s.io/client-go/tools/clientcmd"
	k8sclientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

// GenerateConfig ...
type GenerateConfig struct {
	clusterName                   string
	proxyURL                      url.URL
	resource                      string
	kubeConfig                    string
	tokenCache                    string
	overwrite                     bool
	insecureSkipVerify            bool
	defaultAzureCredentialOptions *azidentity.DefaultAzureCredentialOptions
}

// NewGenerateConfig ...
func NewGenerateConfig(ctx context.Context, c *cli.Context) (GenerateConfig, error) {
	proxyURL, err := url.Parse(c.String("proxy-url"))
	if err != nil {
		return GenerateConfig{}, err
	}
	return GenerateConfig{
		clusterName:        c.String("cluster-name"),
		proxyURL:           *proxyURL,
		resource:           c.String("resource"),
		kubeConfig:         c.String("kubeconfig"),
		tokenCache:         c.String("token-cache"),
		overwrite:          c.Bool("overwrite"),
		insecureSkipVerify: c.Bool("tls-insecure-skip-verify"),
		defaultAzureCredentialOptions: &azidentity.DefaultAzureCredentialOptions{
			ExcludeAzureCLICredential:    c.Bool("exclude-azure-cli-auth"),
			ExcludeEnvironmentCredential: c.Bool("exclude-environment-auth"),
			ExcludeMSICredential:         c.Bool("exclude-msi-auth"),
		},
	}, nil
}

// GenerateFlags ...
func GenerateFlags(ctx context.Context) []cli.Flag {
	usr, _ := user.Current()
	return []cli.Flag{
		&cli.StringFlag{
			Name:     "cluster-name",
			Usage:    "The name of the Kubernetes cluster / context",
			EnvVars:  []string{"CLUSTER_NAME"},
			Required: true,
		},
		&cli.StringFlag{
			Name:     "proxy-url",
			Usage:    "The proxy url for azad-kube-proxy",
			EnvVars:  []string{"PROXY_URL"},
			Required: true,
		},
		&cli.StringFlag{
			Name:     "resource",
			Usage:    "The Azure AD App URI / resource",
			EnvVars:  []string{"RESOURCE"},
			Required: true,
		},
		&cli.StringFlag{
			Name:    "kubeconfig",
			Usage:   "The path of the Kubernetes Config",
			EnvVars: []string{"KUBECONFIG"},
			Value:   fmt.Sprintf("%s/.kube/config", usr.HomeDir),
		},
		&cli.StringFlag{
			Name:    "token-cache",
			Usage:   "The token cache path to cache tokens",
			EnvVars: []string{"TOKEN_CACHE"},
			Value:   "~/.kube/azad-proxy.json",
		},
		&cli.BoolFlag{
			Name:    "overwrite",
			Usage:   "If the cluster already exists in the kubeconfig, should it be overwritten?",
			EnvVars: []string{"OVERWRITE_KUBECONFIG"},
			Value:   false,
		},
		&cli.BoolFlag{
			Name:    "tls-insecure-skip-verify",
			Usage:   "Should the proxy url server certificate validation be skipped?",
			EnvVars: []string{"TLS_INSECURE_SKIP_VERIFY"},
			Value:   false,
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

// Generate ...
func Generate(ctx context.Context, cfg GenerateConfig) error {
	kubeCfg, err := k8sclientcmd.LoadFromFile(cfg.kubeConfig)
	if err != nil && !strings.Contains(err.Error(), "no such file or directory") {
		return err
	}

	if err != nil && strings.Contains(err.Error(), "no such file or directory") {
		kubeCfg = k8sclientcmdapi.NewConfig()
	}

	var found bool
	_, found = kubeCfg.Clusters[cfg.clusterName]
	if found && !cfg.overwrite {
		return fmt.Errorf("Cluster (%s) was found in config (%s) but overwrite is %t", cfg.clusterName, cfg.kubeConfig, cfg.overwrite)
	}

	_, found = kubeCfg.Contexts[cfg.clusterName]
	if found && !cfg.overwrite {
		return fmt.Errorf("Context (%s) was found in config (%s) but overwrite is %t", cfg.clusterName, cfg.kubeConfig, cfg.overwrite)
	}

	_, found = kubeCfg.AuthInfos[cfg.clusterName]
	if found && !cfg.overwrite {
		return fmt.Errorf("User (%s) was found in config (%s) but overwrite is %t", cfg.clusterName, cfg.kubeConfig, cfg.overwrite)
	}

	caCerts, err := getCACertificates(cfg.proxyURL, cfg.insecureSkipVerify)
	if err != nil {
		return err
	}

	serverScheme := cfg.proxyURL.Scheme
	if !strings.EqualFold(serverScheme, "https") {
		serverScheme = "https"
	}

	server := fmt.Sprintf("%s://%s", serverScheme, cfg.proxyURL.Host)

	cluster := &k8sclientcmdapi.Cluster{
		CertificateAuthorityData: caCerts,
		Server:                   server,
	}

	authInfo := &k8sclientcmdapi.AuthInfo{
		Exec: &k8sclientcmdapi.ExecConfig{
			APIVersion: "client.authentication.k8s.io/v1beta1",
			Command:    "kubectl",
			Args: []string{
				"azad-proxy",
				"login",
			},
			Env: []k8sclientcmdapi.ExecEnvVar{
				{
					Name:  "CLUSTER_NAME",
					Value: cfg.clusterName,
				},
				{
					Name:  "RESOURCE",
					Value: cfg.resource,
				},
				{
					Name:  "TOKEN_CACHE",
					Value: cfg.tokenCache,
				},
				{
					Name:  "EXCLUDE_AZURE_CLI_AUTH",
					Value: fmt.Sprintf("%t", cfg.defaultAzureCredentialOptions.ExcludeAzureCLICredential),
				},
				{
					Name:  "EXCLUDE_ENVIRONMENT_AUTH",
					Value: fmt.Sprintf("%t", cfg.defaultAzureCredentialOptions.ExcludeEnvironmentCredential),
				},
				{
					Name:  "EXCLUDE_MSI_AUTH",
					Value: fmt.Sprintf("%t", cfg.defaultAzureCredentialOptions.ExcludeMSICredential),
				},
			},
		},
	}

	context := &k8sclientcmdapi.Context{
		Cluster:  cfg.clusterName,
		AuthInfo: cfg.clusterName,
	}

	kubeCfg.AuthInfos[cfg.clusterName] = authInfo
	kubeCfg.Clusters[cfg.clusterName] = cluster
	kubeCfg.Contexts[cfg.clusterName] = context
	kubeCfg.CurrentContext = cfg.clusterName

	err = k8sclientcmd.WriteToFile(*kubeCfg, cfg.kubeConfig)
	if err != nil {
		return err
	}

	return nil
}

func getCACertificates(url url.URL, insecureSkipVerify bool) ([]byte, error) {
	dialer := &net.Dialer{
		Timeout: 3 * time.Second,
	}

	hostPort := url.Host
	if !strings.Contains(hostPort, ":") {
		hostPort = fmt.Sprintf("%s:%s", hostPort, "443")
	}

	tlsConfig := &tls.Config{
		InsecureSkipVerify: insecureSkipVerify,
	}

	conn, err := tls.DialWithDialer(dialer, "tcp", hostPort, tlsConfig)
	if err != nil {
		return nil, err
	}

	defer conn.Close()

	certs := conn.ConnectionState().PeerCertificates
	var pemCerts []byte
	certCount := len(certs)
	for _, cert := range certs {
		if cert.IsCA || certCount == 1 {
			pemBlock := pem.Block{
				Type:  "CERTIFICATE",
				Bytes: cert.Raw,
			}
			pemCertBytes := pem.EncodeToMemory(&pemBlock)
			pemCerts = append(pemCerts, pemCertBytes...)
		}
	}

	// b64Certs := base64.StdEncoding.EncodeToString(pemCerts)

	return pemCerts, nil
}
