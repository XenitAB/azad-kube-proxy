package health

import (
	"context"
	"fmt"
	"net/url"

	"github.com/xenitab/azad-kube-proxy/pkg/config"
	"github.com/xenitab/azad-kube-proxy/pkg/util"
	k8sapiauthorization "k8s.io/api/authorization/v1"
	k8sapimachinerymetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8s "k8s.io/client-go/kubernetes"
	k8sclientrest "k8s.io/client-go/rest"
)

// ClientInterface ...
type ClientInterface interface {
	Ready(ctx context.Context) (bool, error)
	Live(ctx context.Context) (bool, error)
}

// Validator ...
type Validator interface {
	Valid(ctx context.Context) bool
}

// Client ...
type Client struct {
	k8sClient         k8s.Interface
	livenessValidator Validator
}

// NewHealthClient ...
func NewHealthClient(ctx context.Context, cfg *config.Config, livenessValidator Validator) (ClientInterface, error) {
	k8sTLSConfig := k8sclientrest.TLSClientConfig{Insecure: true}
	if cfg.KubernetesAPIValidateCert {
		kubernetesRootCAString, err := util.GetStringFromFile(ctx, cfg.KubernetesAPICACertPath)
		if err != nil {
			return nil, err
		}

		k8sTLSConfig = k8sclientrest.TLSClientConfig{
			Insecure: false,
			CAData:   []byte(kubernetesRootCAString),
		}
	}

	kubernetesAPIUrl, err := getKubernetesAPIUrl(cfg.KubernetesAPIHost, cfg.KubernetesAPIPort, cfg.KubernetesAPITLS)
	if err != nil {
		return nil, err
	}

	kubernetesToken, err := util.GetStringFromFile(ctx, cfg.KubernetesAPITokenPath)
	if err != nil {
		return nil, err
	}

	k8sRestConfig := &k8sclientrest.Config{
		Host:            kubernetesAPIUrl.String(),
		BearerToken:     kubernetesToken,
		TLSClientConfig: k8sTLSConfig,
	}

	k8sClient, err := k8s.NewForConfig(k8sRestConfig)
	if err != nil {
		return nil, err
	}

	healthClient := &Client{
		k8sClient:         k8sClient,
		livenessValidator: livenessValidator,
	}

	return healthClient, nil
}

// Ready ...
func (client *Client) Ready(ctx context.Context) (bool, error) {
	ready := false

	selfSubjectRulesReview := &k8sapiauthorization.SelfSubjectRulesReview{Spec: k8sapiauthorization.SelfSubjectRulesReviewSpec{Namespace: "default"}}
	createOptions := k8sapimachinerymetav1.CreateOptions{}
	res, err := client.k8sClient.AuthorizationV1().SelfSubjectRulesReviews().Create(ctx, selfSubjectRulesReview, createOptions)
	if err != nil {
		return false, err
	}

	for _, rule := range res.Status.ResourceRules {
		if util.SliceContains(rule.Verbs, "impersonate") {
			if util.SliceContains(rule.Resources, "users") && util.SliceContains(rule.Resources, "groups") && util.SliceContains(rule.Resources, "serviceaccounts") {
				ready = true
			}
		}
	}

	if !ready {
		err := fmt.Errorf("Impersonate rule not found: %q", res)
		return false, err
	}

	return true, nil
}

// Live ...
func (client *Client) Live(ctx context.Context) (bool, error) {
	valid := client.livenessValidator.Valid(ctx)
	return valid, nil
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
