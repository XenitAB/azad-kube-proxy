package user_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestUser(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "User Suite")
}
