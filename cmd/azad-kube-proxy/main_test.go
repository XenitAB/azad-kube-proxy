package main

import (
	"os"
	"testing"
)

func TestCIRequirements(t *testing.T) {
	ciEnvVar := getEnvOrSkip(t, "CI")
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
		getEnvOrError(t, envVar)
	}

}

func getEnvOrError(t *testing.T, envVar string) string {
	v := os.Getenv(envVar)
	if v == "" {
		t.Errorf("%s environment variable is required by CI.", envVar)
	}

	return v
}

func getEnvOrSkip(t *testing.T, envVar string) string {
	v := os.Getenv(envVar)
	if v == "" {
		t.Skipf("%s environment variable is empty, skipping.", envVar)
	}

	return v
}
