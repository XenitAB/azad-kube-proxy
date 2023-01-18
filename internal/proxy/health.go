package proxy

import (
	"context"
	"fmt"

	"github.com/xenitab/azad-kube-proxy/internal/config"
	k8sapiauthorization "k8s.io/api/authorization/v1"
	k8sapimachinerymetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8s "k8s.io/client-go/kubernetes"
	k8sclientrest "k8s.io/client-go/rest"
)

type Health interface {
	Ready(ctx context.Context) (bool, error)
	Live(ctx context.Context) (bool, error)
}

type HealthValidator interface {
	Valid(ctx context.Context) bool
}

type health struct {
	k8sClient         k8s.Interface
	livenessValidator HealthValidator
}

func newHealthClient(ctx context.Context, cfg *config.Config, livenessValidator HealthValidator) (*health, error) {
	k8sTLSConfig := k8sclientrest.TLSClientConfig{Insecure: true}
	if cfg.KubernetesAPIValidateCert {
		kubernetesRootCAString, err := getStringFromFile(ctx, cfg.KubernetesAPICACertPath)
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

	kubernetesToken, err := getStringFromFile(ctx, cfg.KubernetesAPITokenPath)
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

	healthClient := &health{
		k8sClient:         k8sClient,
		livenessValidator: livenessValidator,
	}

	return healthClient, nil
}

func (h *health) Ready(ctx context.Context) (bool, error) {
	ready := false

	selfSubjectRulesReview := &k8sapiauthorization.SelfSubjectRulesReview{Spec: k8sapiauthorization.SelfSubjectRulesReviewSpec{Namespace: "default"}}
	createOptions := k8sapimachinerymetav1.CreateOptions{}
	res, err := h.k8sClient.AuthorizationV1().SelfSubjectRulesReviews().Create(ctx, selfSubjectRulesReview, createOptions)
	if err != nil {
		return false, err
	}

	for _, rule := range res.Status.ResourceRules {
		if sliceContains(rule.Verbs, "impersonate") {
			if sliceContains(rule.Resources, "users") && sliceContains(rule.Resources, "groups") && sliceContains(rule.Resources, "serviceaccounts") {
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

func (h *health) Live(ctx context.Context) (bool, error) {
	valid := h.livenessValidator.Valid(ctx)
	return valid, nil
}
