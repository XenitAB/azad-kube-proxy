package main

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCIRequirements(t *testing.T) {
	ciEnvVar := testGetEnvOrSkip(t, "CI")
	if ciEnvVar != "true" {
		t.Skipf("CI environment variable not set to true: %s", ciEnvVar)
	}
	reqEnvVars := []string{
		"CLIENT_ID",
		"CLIENT_SECRET",
		"TENANT_ID",
		"TEST_USER_SP_CLIENT_ID",
		"TEST_USER_SP_CLIENT_SECRET",
		"TEST_USER_SP_RESOURCE",
		"TEST_USER_SP_OBJECT_ID",
		"TEST_USER_OBJECT_ID",
	}

	for _, envVar := range reqEnvVars {
		testGetEnvOrError(t, envVar)
	}

}

func testGetEnvOrError(t *testing.T, envVar string) string {
	t.Helper()

	v := os.Getenv(envVar)
	require.NotEmpty(t, v)

	return v
}

func testGetEnvOrSkip(t *testing.T, envVar string) string {
	t.Helper()

	v := os.Getenv(envVar)
	if v == "" {
		t.Skipf("%s environment variable is empty, skipping.", envVar)
	}

	return v
}
