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
	"github.com/go-logr/logr"
	"github.com/urfave/cli/v2"
	"github.com/xenitab/azad-kube-proxy/cmd/kubectl-azad-proxy/customerrors"
	k8sclientcmd "k8s.io/client-go/tools/clientcmd"
	k8sclientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

// GenerateClient ...
type GenerateClient struct {
	clusterName                   string
	proxyURL                      url.URL
	resource                      string
	kubeConfig                    string
	tokenCache                    string
	overwrite                     bool
	insecureSkipVerify            bool
	defaultAzureCredentialOptions *azidentity.DefaultAzureCredentialOptions
}

// GenerateInterface ...
type GenerateInterface interface {
	Generate(ctx context.Context) error
	Merge(new GenerateClient)
}

// NewGenerateClient ...
func NewGenerateClient(ctx context.Context, c *cli.Context) (GenerateInterface, error) {
	log := logr.FromContext(ctx)

	proxyURL, err := url.Parse(c.String("proxy-url"))
	if err != nil {
		log.V(1).Info("Unable to parse proxy-url", "error", err.Error())
		return nil, err
	}

	return &GenerateClient{
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
			Name:     "kubeconfig",
			Usage:    "The path of the Kubernetes Config",
			EnvVars:  []string{"KUBECONFIG"},
			Value:    fmt.Sprintf("%s/.kube/config", usr.HomeDir),
			Required: false,
		},
		&cli.StringFlag{
			Name:     "token-cache",
			Usage:    "The token cache path to cache tokens",
			EnvVars:  []string{"TOKEN_CACHE"},
			Value:    "~/.kube/azad-proxy.json",
			Required: false,
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
			Value:   true,
		},
		&cli.BoolFlag{
			Name:    "exclude-msi-auth",
			Usage:   "Should MSI be excluded from the authentication?",
			EnvVars: []string{"EXCLUDE_MSI_AUTH"},
			Value:   true,
		},
	}
}

// Generate ...
func (client *GenerateClient) Generate(ctx context.Context) error {
	log := logr.FromContext(ctx)

	kubeCfg, err := k8sclientcmd.LoadFromFile(client.kubeConfig)
	if err != nil && !strings.Contains(err.Error(), "no such file or directory") {
		log.V(1).Info("Unable to load kubeConfig", "error", err.Error(), "kubeConfig", client.kubeConfig)
		return customerrors.New(customerrors.ErrorTypeKubeConfig, err)
	}

	if err != nil && strings.Contains(err.Error(), "no such file or directory") {
		kubeCfg = k8sclientcmdapi.NewConfig()
	}

	var found bool
	_, found = kubeCfg.Clusters[client.clusterName]
	if found && !client.overwrite {
		err := fmt.Errorf("Cluster (%s) was found in config (%s) but overwrite is %t", client.clusterName, client.kubeConfig, client.overwrite)
		log.V(1).Info("Overwrite is not enabled", "error", err.Error())
		return customerrors.New(customerrors.ErrorTypeOverwriteConfig, err)
	}

	_, found = kubeCfg.Contexts[client.clusterName]
	if found && !client.overwrite {
		err := fmt.Errorf("Context (%s) was found in config (%s) but overwrite is %t", client.clusterName, client.kubeConfig, client.overwrite)
		log.V(1).Info("Overwrite is not enabled", "error", err.Error())
		return customerrors.New(customerrors.ErrorTypeOverwriteConfig, err)
	}

	_, found = kubeCfg.AuthInfos[client.clusterName]
	if found && !client.overwrite {
		err := fmt.Errorf("User (%s) was found in config (%s) but overwrite is %t", client.clusterName, client.kubeConfig, client.overwrite)
		log.V(1).Info("Overwrite is not enabled", "error", err.Error())
		return customerrors.New(customerrors.ErrorTypeOverwriteConfig, err)
	}

	caCerts, err := getCACertificates(client.proxyURL, client.insecureSkipVerify)
	if err != nil {
		log.V(1).Info("Unable to connect get CA certificates", "error", err.Error())
		return customerrors.New(customerrors.ErrorTypeCACertificate, err)
	}

	serverScheme := client.proxyURL.Scheme
	if !strings.EqualFold(serverScheme, "https") {
		serverScheme = "https"
	}

	server := fmt.Sprintf("%s://%s", serverScheme, client.proxyURL.Host)

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
					Value: client.clusterName,
				},
				{
					Name:  "RESOURCE",
					Value: client.resource,
				},
				{
					Name:  "TOKEN_CACHE",
					Value: client.tokenCache,
				},
				{
					Name:  "EXCLUDE_AZURE_CLI_AUTH",
					Value: fmt.Sprintf("%t", client.defaultAzureCredentialOptions.ExcludeAzureCLICredential),
				},
				{
					Name:  "EXCLUDE_ENVIRONMENT_AUTH",
					Value: fmt.Sprintf("%t", client.defaultAzureCredentialOptions.ExcludeEnvironmentCredential),
				},
				{
					Name:  "EXCLUDE_MSI_AUTH",
					Value: fmt.Sprintf("%t", client.defaultAzureCredentialOptions.ExcludeMSICredential),
				},
			},
		},
	}

	context := &k8sclientcmdapi.Context{
		Cluster:  client.clusterName,
		AuthInfo: client.clusterName,
	}

	kubeCfg.AuthInfos[client.clusterName] = authInfo
	kubeCfg.Clusters[client.clusterName] = cluster
	kubeCfg.Contexts[client.clusterName] = context
	kubeCfg.CurrentContext = client.clusterName

	err = k8sclientcmd.WriteToFile(*kubeCfg, client.kubeConfig)
	if err != nil {
		log.V(1).Info("Unable to write to kubeConfig", "error", err.Error())
		return customerrors.New(customerrors.ErrorTypeKubeConfig, err)
	}

	log.V(0).Info("Configuration written", "kubeConfig", client.kubeConfig, "clusterName", client.clusterName)

	return nil
}

func (client *GenerateClient) Merge(new GenerateClient) {
	if new.clusterName != "" {
		client.clusterName = new.clusterName
	}

	if new.proxyURL.String() != "" {
		client.proxyURL = new.proxyURL
	}

	if new.resource != "" {
		client.resource = new.resource
	}

	if new.kubeConfig != "" {
		client.kubeConfig = new.kubeConfig
	}

	if new.tokenCache != "" {
		client.tokenCache = new.tokenCache
	}

	if new.overwrite != client.overwrite {
		client.overwrite = new.overwrite
	}

	if new.insecureSkipVerify != client.insecureSkipVerify {
		client.insecureSkipVerify = new.insecureSkipVerify
	}
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
	} // #nosec

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

	return pemCerts, nil
}
