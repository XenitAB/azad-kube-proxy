package util

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"fmt"
	"log"
	"math/big"
	"net/http"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/go-logr/logr"
)

func TestGetCertificate(t *testing.T) {
	ctx := logr.NewContext(context.Background(), logr.DiscardLogger{})

	certPath, err := generateCertificateFile()
	if err != nil {
		t.Errorf("Unable to generate temporary certificate for test: %q", err)
	}

	certPool, err := GetCertificate(ctx, certPath)

	deleteErr := deleteFile(certPath)
	if deleteErr != nil {
		t.Errorf("Unable to delete certificate %s: %q", certPath, err)
	}

	if err != nil {
		t.Errorf("Returned error: %q", err)
	}

	if len(certPool.Subjects()) != 1 {
		t.Error("certPool doesn't contain exactly one subject")
	}

	_, expectedErr := GetCertificate(ctx, fmt.Sprintf("%s-does-not-exist", certPath))
	if expectedErr == nil {
		t.Error("Error wasn't returned for non existing file")
	}
}

func TestGetStringFromFile(t *testing.T) {
	ctx := logr.NewContext(context.Background(), logr.DiscardLogger{})
	filePath, expectedFileString, err := generateRandomFile()
	if err != nil {
		t.Errorf("Unable to generate temporary file for test: %q", err)
	}

	fileString, err := GetStringFromFile(ctx, filePath)

	deleteErr := deleteFile(filePath)
	if deleteErr != nil {
		t.Errorf("Unable to delete file %s: %q", filePath, err)
	}

	if fileString != expectedFileString {
		t.Errorf("fileString (%s) does not match expectedFileString: %s", fileString, expectedFileString)
	}

	_, expectedErr := GetStringFromFile(ctx, fmt.Sprintf("%s-does-not-exist", filePath))
	if expectedErr == nil {
		t.Error("Error wasn't returned for non existing file")
	}
}

func TestGetEncodedHash(t *testing.T) {
	testString := "this is a test string"
	testStringHash := GetEncodedHash(testString)
	expectedHash := "f6774519d1c7a3389ef327e9c04766b999db8cdfb85d1346c471ee86d65885bc"
	if testStringHash != expectedHash {
		t.Errorf("testStringHash (%s) doesn't equal expectedHash: %s", testStringHash, expectedHash)
	}
}

func TestGetBearerToken(t *testing.T) {
	cases := []struct {
		token                  string
		addAuthorizationHeader bool
		headerTemplate         string
		expectedErr            error
	}{
		{
			token:                  "token",
			addAuthorizationHeader: true,
			headerTemplate:         "Bearer %s",
			expectedErr:            nil,
		},
		{
			token:                  "",
			addAuthorizationHeader: false,
			headerTemplate:         "",
			expectedErr:            errors.New("No Authorization header present in request"),
		},
		{
			token:                  "token",
			addAuthorizationHeader: true,
			headerTemplate:         "%s",
			expectedErr:            errors.New("Authorization header does not contain Bearer in request"),
		},
		{
			token:                  "Bearer ",
			addAuthorizationHeader: true,
			headerTemplate:         "%s",
			expectedErr:            errors.New("Empty token"),
		},
		{
			token:                  "Bearer Bearer Bearer ",
			addAuthorizationHeader: true,
			headerTemplate:         "%s",
			expectedErr:            errors.New("Authorization split by 'Bearer ' isn't length of 2 (actual length: 4)"),
		},
	}

	for _, c := range cases {
		req := &http.Request{}
		if c.addAuthorizationHeader {
			authorizationHeader := fmt.Sprintf(c.headerTemplate, c.token)
			req = &http.Request{
				Header: map[string][]string{
					"Authorization": {authorizationHeader},
				},
			}
		}

		tokenResponse, err := GetBearerToken(req)
		if tokenResponse != c.token && c.expectedErr == nil {
			t.Errorf("Expected token (%s) was not returned: %s", c.token, tokenResponse)
		}

		if err != nil && c.expectedErr == nil {
			t.Errorf("Expected err to be nil but it was %q", err)
		}

		if c.expectedErr != nil {
			if err.Error() != c.expectedErr.Error() {
				t.Errorf("Expected err to be %q but it was %q", c.expectedErr, err)
			}
		}
	}
}

func generateCertificateFile() (string, error) {
	timestamp := strconv.FormatInt(time.Now().UTC().UnixNano(), 10)
	filename := fmt.Sprintf("test-cert-%s.pem", timestamp)
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		log.Fatal(err)
	}

	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			Organization: []string{"Testing"},
		},
		NotBefore: time.Now(),
		NotAfter:  time.Now().Add(time.Hour * 24 * 180),

		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}

	certBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &priv.PublicKey, priv)
	if err != nil {
		return "", fmt.Errorf("Failed to create certificate: %v", err)
	}

	certOut, err := os.Create(filename)
	if err != nil {
		return "", fmt.Errorf("Failed to open %s for writing: %v", filename, err)
	}

	err = pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: certBytes})
	if err != nil {
		return "", fmt.Errorf("Failed to write data to %s: %v", filename, err)
	}

	err = certOut.Close()
	if err != nil {
		return "", fmt.Errorf("Error closing %s: %v", filename, err)
	}

	return filename, nil
}

func generateRandomFile() (string, string, error) {
	timestamp := strconv.FormatInt(time.Now().UTC().UnixNano(), 10)
	filename := fmt.Sprintf("test-random-%s.pem", timestamp)
	content := []byte(timestamp)

	err := os.WriteFile(filename, content, 0644)
	if err != nil {
		return "", "", fmt.Errorf("Failed to create %s: %v", filename, err)
	}

	return filename, timestamp, nil
}

func deleteFile(file string) error {
	err := os.Remove(file)
	if err != nil {
		return fmt.Errorf("Unable to delete file %s: %v", file, err)
	}

	return nil
}
