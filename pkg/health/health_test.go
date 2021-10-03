package health

import (
	"context"
	"net/url"
	"strings"
	"testing"

	"github.com/go-logr/logr"
	"github.com/xenitab/azad-kube-proxy/pkg/config"
	k8sapiauthorization "k8s.io/api/authorization/v1"
	"k8s.io/apimachinery/pkg/runtime"
	k8sfake "k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"
)

func TestNewHealthClient(t *testing.T) {
	ctx := logr.NewContext(context.Background(), logr.Discard())

	cases := []struct {
		config              config.Config
		expectedErrContains string
	}{
		{
			config: config.Config{
				KubernetesConfig: config.KubernetesConfig{
					ValidateCertificate: true,
					URL:                 &url.URL{Scheme: "https", Host: "fake-url"},
					Token:               "fake-token",
					RootCAString:        "fake-ca-string",
				},
			},
			expectedErrContains: "unable to load root certificates: unable to parse bytes as PEM block",
		},
		{
			config: config.Config{
				KubernetesConfig: config.KubernetesConfig{
					ValidateCertificate: false,
					URL:                 &url.URL{Scheme: "https", Host: "fake-url"},
					Token:               "fake-token",
				},
			},
			expectedErrContains: "",
		},
	}

	for _, c := range cases {
		validator := &fakeValidator{}
		_, err := NewHealthClient(ctx, c.config, validator)
		if err != nil && c.expectedErrContains == "" {
			t.Errorf("Expected err to be nil: %q", err)
		}

		if err == nil && c.expectedErrContains != "" {
			t.Errorf("Expected err to contain '%s' but was nil", c.expectedErrContains)
		}

		if err != nil && c.expectedErrContains != "" {
			if !strings.Contains(err.Error(), c.expectedErrContains) {
				t.Errorf("Expected err to contain '%s' but was: %q", c.expectedErrContains, err)
			}
		}
	}
}

func TestReady(t *testing.T) {
	ctx := logr.NewContext(context.Background(), logr.Discard())

	fakeClient := &Client{
		k8sClient: k8sfake.NewSimpleClientset(),
	}

	fakeClient.k8sClient.(*k8sfake.Clientset).Fake.PrependReactor("create", "selfsubjectrulesreviews", func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
		object := &k8sapiauthorization.SelfSubjectRulesReview{
			Status: k8sapiauthorization.SubjectRulesReviewStatus{
				ResourceRules: []k8sapiauthorization.ResourceRule{
					{
						Verbs:     []string{"impersonate"},
						Resources: []string{"users", "groups", "serviceaccounts"},
					},
				},
			},
		}
		return true, object, nil
	})

	cases := []struct {
		clientFunc          func(c *Client) ClientInterface
		expectedErrContains string
		expectedReady       bool
	}{
		{
			clientFunc: func(c *Client) ClientInterface {
				return c
			},
			expectedErrContains: "",
			expectedReady:       true,
		},
		{
			clientFunc: func(c *Client) ClientInterface {
				return &Client{
					k8sClient: k8sfake.NewSimpleClientset(),
				}
			},
			expectedErrContains: "Impersonate rule not found",
			expectedReady:       false,
		},
	}

	for _, c := range cases {
		client := c.clientFunc(fakeClient)

		ready, err := client.Ready(ctx)
		if err != nil && c.expectedErrContains == "" {
			t.Errorf("Expected err to be nil: %q", err)
		}

		if err == nil && c.expectedErrContains != "" {
			t.Errorf("Expected err to contain '%s' but was nil", c.expectedErrContains)
		}

		if err != nil && c.expectedErrContains != "" {
			if !strings.Contains(err.Error(), c.expectedErrContains) {
				t.Errorf("Expected err to contain '%s' but was: %q", c.expectedErrContains, err)
			}
		}

		if c.expectedReady != ready {
			t.Errorf("Expected ready to be '%t' but was: %t", c.expectedReady, ready)
		}
	}
}

func TestLive(t *testing.T) {
	ctx := logr.NewContext(context.Background(), logr.Discard())

	validator := &fakeValidator{}
	fakeConfig := config.Config{
		KubernetesConfig: config.KubernetesConfig{
			ValidateCertificate: false,
			URL:                 &url.URL{Scheme: "https", Host: "fake-url"},
			Token:               "fake-token",
			RootCAString:        "fake-ca-string",
		},
	}
	client, err := NewHealthClient(ctx, fakeConfig, validator)
	if err != nil {
		t.Errorf("Expected err to be nil: %q", err)
	}

	live, err := client.Live(ctx)
	if err != nil {
		t.Errorf("Expected err to be nil: %q", err)
	}

	if !live {
		t.Errorf("Expected live to be 'true': %T", live)
	}
}

type fakeValidator struct{}

// Valid ...
func (client *fakeValidator) Valid(ctx context.Context) bool {
	return true
}
