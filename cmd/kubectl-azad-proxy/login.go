package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"

	k8smetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sclientauth "k8s.io/client-go/pkg/apis/clientauthentication/v1beta1"
)

// LoginClient ...
type LoginClient struct {
	clusterName                   string
	resource                      string
	tokenCacheDir                 string
	defaultAzureCredentialOptions defaultAzureCredentialOptions
}

func runLogin(ctx context.Context, writer io.Writer, cfg loginConfig, authCfg authConfig) error {
	client := newLoginClient(ctx, cfg, authCfg)

	output, err := client.Login(ctx)
	if err != nil {
		return err
	}

	fmt.Fprint(writer, output)

	return nil
}

func newLoginClient(ctx context.Context, cfg loginConfig, authCfg authConfig) *LoginClient {
	tokenCacheDir := getTokenCacheDirectory(cfg.TokenCacheDir, cfg.KubeConfig)
	return &LoginClient{
		clusterName:   cfg.ClusterName,
		resource:      cfg.Resource,
		tokenCacheDir: tokenCacheDir,
		defaultAzureCredentialOptions: defaultAzureCredentialOptions{
			excludeAzureCLICredential:    authCfg.excludeAzureCLIAuth,
			excludeEnvironmentCredential: authCfg.excludeEnvironmentAuth,
			excludeMSICredential:         authCfg.excludeMSIAuth,
		},
	}
}

func getTokenCacheDirectory(tokenCacheDir, kubeConfig string) string {
	if tokenCacheDir != "" {
		return filepath.Clean(tokenCacheDir)
	}

	if kubeConfig != "" {
		return filepath.Dir(kubeConfig)
	}

	userHomeDir := "~"
	osUserHomeDir, err := os.UserHomeDir()
	if err == nil {
		userHomeDir = osUserHomeDir
	}

	return filepath.Clean(fmt.Sprintf("%s/.kube", userHomeDir))
}

func (client *LoginClient) Login(ctx context.Context) (string, error) {
	tokens, err := newTokens(ctx, client.tokenCacheDir, client.defaultAzureCredentialOptions)
	if err != nil {
		return "", err
	}

	token, err := tokens.GetToken(ctx, client.clusterName, client.resource)
	if err != nil {
		return "", err
	}

	execCredential := &k8sclientauth.ExecCredential{
		TypeMeta: k8smetav1.TypeMeta{
			APIVersion: "client.authentication.k8s.io/v1beta1",
			Kind:       "ExecCredential",
		},
		Status: &k8sclientauth.ExecCredentialStatus{
			Token:               token.Token,
			ExpirationTimestamp: &k8smetav1.Time{Time: token.ExpirationTimestamp},
		},
	}

	res, err := json.Marshal(execCredential)
	if err != nil {
		return "", err
	}

	return string(res), nil
}
