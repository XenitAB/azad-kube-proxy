package main

import (
	"context"
	"crypto/tls"
	"encoding/pem"
	"fmt"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-logr/logr"
	k8sclientcmd "k8s.io/client-go/tools/clientcmd"
	k8sclientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

type GenerateClient struct {
	clusterName                   string
	proxyURL                      url.URL
	resource                      string
	kubeConfig                    string
	tokenCacheDir                 string
	overwrite                     bool
	insecureSkipVerify            bool
	defaultAzureCredentialOptions defaultAzureCredentialOptions
}

type GenerateInterface interface {
	Generate(ctx context.Context) error
	Merge(new GenerateClient)
}

func runGenerate(ctx context.Context, cfg generateConfig, authCfg authConfig) error {
	client, err := newGenerateClient(ctx, cfg, authCfg)
	if err != nil {
		return err
	}

	return client.Generate(ctx)
}

func newGenerateClient(ctx context.Context, cfg generateConfig, authCfg authConfig) (*GenerateClient, error) {
	log := logr.FromContextOrDiscard(ctx)

	proxyURL, err := url.Parse(cfg.ProxyURL)
	if err != nil {
		log.V(1).Info("Unable to parse proxy-url", "error", err.Error())
		return nil, err
	}

	tokenCacheDir := getTokenCacheDirectory(cfg.TokenCacheDir, cfg.KubeConfig)

	return &GenerateClient{
		clusterName:        cfg.ClusterName,
		proxyURL:           *proxyURL,
		resource:           cfg.Resource,
		kubeConfig:         filepath.Clean(cfg.KubeConfig),
		tokenCacheDir:      tokenCacheDir,
		overwrite:          cfg.Overwrite,
		insecureSkipVerify: cfg.TLSInsecureSkipVerify,
		defaultAzureCredentialOptions: defaultAzureCredentialOptions{
			excludeAzureCLICredential:    authCfg.excludeAzureCLIAuth,
			excludeEnvironmentCredential: authCfg.excludeEnvironmentAuth,
			excludeMSICredential:         authCfg.excludeMSIAuth,
		},
	}, nil
}

func (client *GenerateClient) Generate(ctx context.Context) error {
	log := logr.FromContextOrDiscard(ctx)

	kubeCfg, err := k8sclientcmd.LoadFromFile(client.kubeConfig)
	if err != nil && !os.IsNotExist(err) {
		log.V(1).Info("Unable to load kubeConfig", "error", err.Error(), "kubeConfig", client.kubeConfig)
		return newCustomError(errorTypeKubeConfig, err)
	}

	if err != nil && os.IsNotExist(err) {
		kubeCfg = k8sclientcmdapi.NewConfig()
	}

	var found bool
	_, found = kubeCfg.Clusters[client.clusterName]
	if found && !client.overwrite {
		err := fmt.Errorf("cluster (%s) was found in config (%s) but overwrite is %t", client.clusterName, client.kubeConfig, client.overwrite)
		log.V(1).Info("Overwrite is not enabled", "error", err.Error())
		return newCustomError(errorTypeOverwriteConfig, err)
	}

	_, found = kubeCfg.Contexts[client.clusterName]
	if found && !client.overwrite {
		err := fmt.Errorf("context (%s) was found in config (%s) but overwrite is %t", client.clusterName, client.kubeConfig, client.overwrite)
		log.V(1).Info("Overwrite is not enabled", "error", err.Error())
		return newCustomError(errorTypeOverwriteConfig, err)
	}

	_, found = kubeCfg.AuthInfos[client.clusterName]
	if found && !client.overwrite {
		err := fmt.Errorf("user (%s) was found in config (%s) but overwrite is %t", client.clusterName, client.kubeConfig, client.overwrite)
		log.V(1).Info("Overwrite is not enabled", "error", err.Error())
		return newCustomError(errorTypeOverwriteConfig, err)
	}

	caCerts, err := getCACertificates(client.proxyURL, client.insecureSkipVerify)
	if err != nil {
		log.V(1).Info("Unable to connect get CA certificates", "error", err.Error())
		return newCustomError(errorTypeCACertificate, err)
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
					Name:  "TOKEN_CACHE_DIR",
					Value: client.tokenCacheDir,
				},
				{
					Name:  "EXCLUDE_AZURE_CLI_AUTH",
					Value: fmt.Sprintf("%t", client.defaultAzureCredentialOptions.excludeAzureCLICredential),
				},
				{
					Name:  "EXCLUDE_ENVIRONMENT_AUTH",
					Value: fmt.Sprintf("%t", client.defaultAzureCredentialOptions.excludeEnvironmentCredential),
				},
				{
					Name:  "EXCLUDE_MSI_AUTH",
					Value: fmt.Sprintf("%t", client.defaultAzureCredentialOptions.excludeMSICredential),
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
		return newCustomError(errorTypeKubeConfig, err)
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

	if new.tokenCacheDir != "" {
		client.tokenCacheDir = new.tokenCacheDir
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
