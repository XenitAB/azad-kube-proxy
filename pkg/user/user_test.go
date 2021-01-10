package user_test

import (
	"context"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/xenitab/azad-kube-proxy/pkg/azure"
	"github.com/xenitab/azad-kube-proxy/pkg/config"
	"github.com/xenitab/azad-kube-proxy/pkg/models"
	"github.com/xenitab/azad-kube-proxy/pkg/user"
)

var _ = Describe("User", func() {
	var (
		userClient  *user.Client
		config      config.Config
		azureClient *azure.Client
	)

	BeforeEach(func() {
		userClient = user.NewUserClient(config, azureClient)
	})

	Describe("Get user", func() {
		Context("Service principal", func() {
			It("it should return userType service principal", func() {
				user, err := userClient.GetUser(context.Background(), "", "00000000-0000-0000-0000-000000000000")
				Expect(err).NotTo(HaveOccurred())
				Expect(user.Type).To(Equal(models.ServicePrincipalUserType))
			})
		})
	})
})
