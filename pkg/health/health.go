package health

import (
	"context"
	"fmt"

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
	k8sClient k8s.Interface
	validator Validator
}

// NewHealthClient ...
func NewHealthClient(ctx context.Context, config config.Config, validator Validator) (ClientInterface, error) {
	k8sTLSConfig := k8sclientrest.TLSClientConfig{Insecure: true}
	if config.KubernetesConfig.ValidateCertificate {
		k8sTLSConfig = k8sclientrest.TLSClientConfig{
			Insecure: false,
			CAData:   []byte(config.KubernetesConfig.RootCAString),
		}
	}

	k8sRestConfig := &k8sclientrest.Config{
		Host:            config.KubernetesConfig.URL.String(),
		BearerToken:     config.KubernetesConfig.Token,
		TLSClientConfig: k8sTLSConfig,
	}

	k8sClient, err := k8s.NewForConfig(k8sRestConfig)
	if err != nil {
		return nil, err
	}

	healthClient := &Client{
		k8sClient: k8sClient,
		validator: validator,
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
	valid := client.validator.Valid(ctx)
	return valid, nil
}
