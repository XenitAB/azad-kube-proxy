package proxy

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/require"
	k8sapiauthorization "k8s.io/api/authorization/v1"
	"k8s.io/apimachinery/pkg/runtime"
	k8sfake "k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"
)

func TestNewHealthClient(t *testing.T) {
	ctx := logr.NewContext(context.Background(), logr.Discard())

	tmpDir, err := os.MkdirTemp("", "")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)
	tokenPath := filepath.Clean(fmt.Sprintf("%s/kubernetes-token", tmpDir))
	caPath := filepath.Clean(fmt.Sprintf("%s/kubernetes-ca", tmpDir))
	testCreateTemporaryFile(t, tokenPath, "fake-token")
	testCreateTemporaryFile(t, caPath, "fake-ca-string")

	cases := []struct {
		config              *config
		expectedErrContains string
	}{
		{
			config: &config{
				KubernetesAPITLS:          true,
				KubernetesAPIValidateCert: true,
				KubernetesAPIHost:         "fake-url",
				KubernetesAPITokenPath:    tokenPath,
				KubernetesAPICACertPath:   caPath,
			},
			expectedErrContains: "unable to load root certificates: unable to parse bytes as PEM block",
		},
		{
			config: &config{
				KubernetesAPITLS:          true,
				KubernetesAPIValidateCert: false,
				KubernetesAPIHost:         "fake-url",
				KubernetesAPITokenPath:    tokenPath,
			},
			expectedErrContains: "",
		},
	}

	for _, c := range cases {
		validator := &testFakeValidator{t}
		_, err := newHealthClient(ctx, c.config, validator)
		if c.expectedErrContains != "" {
			require.ErrorContains(t, err, c.expectedErrContains)
			continue
		}

		require.NoError(t, err)
	}
}

func TestReady(t *testing.T) {
	ctx := logr.NewContext(context.Background(), logr.Discard())

	fakeClient := &health{
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
		clientFunc          func(h *health) Health
		expectedErrContains string
		expectedReady       bool
	}{
		{
			clientFunc: func(h *health) Health {
				return h
			},
			expectedErrContains: "",
			expectedReady:       true,
		},
		{
			clientFunc: func(h *health) Health {
				return &health{
					k8sClient: k8sfake.NewSimpleClientset(),
				}
			},
			expectedErrContains: "Impersonate rule not found",
			expectedReady:       false,
		},
	}

	for _, c := range cases {
		client := c.clientFunc(fakeClient)

		ready, err := client.ready(ctx)
		if c.expectedErrContains != "" {
			require.ErrorContains(t, err, c.expectedErrContains)
			continue
		}

		require.NoError(t, err)
		require.Equal(t, c.expectedReady, ready)
	}
}

func TestLive(t *testing.T) {
	ctx := logr.NewContext(context.Background(), logr.Discard())

	tmpDir, err := os.MkdirTemp("", "")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)
	tokenPath := filepath.Clean(fmt.Sprintf("%s/kubernetes-token", tmpDir))
	caPath := filepath.Clean(fmt.Sprintf("%s/kubernetes-ca", tmpDir))
	testCreateTemporaryFile(t, tokenPath, "fake-token")
	testCreateTemporaryFile(t, caPath, "fake-ca-string")

	validator := &testFakeValidator{t}
	fakeConfig := &config{
		KubernetesAPIValidateCert: false,
		KubernetesAPITLS:          true,
		KubernetesAPIHost:         "fake-url",
		KubernetesAPITokenPath:    tokenPath,
		KubernetesAPICACertPath:   caPath,
	}
	client, err := newHealthClient(ctx, fakeConfig, validator)
	require.NoError(t, err)

	live, err := client.live(ctx)
	require.NoError(t, err)
	require.True(t, live)
}

type testFakeValidator struct {
	t *testing.T
}

// Valid ...
func (client *testFakeValidator) valid(ctx context.Context) bool {
	client.t.Helper()

	return true
}

func testCreateTemporaryFile(t *testing.T, path string, content string) {
	t.Helper()

	err := os.WriteFile(path, []byte(content), 0600)
	require.NoError(t, err)
}
