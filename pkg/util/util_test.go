package util

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"log"
	"math/big"
	"net/http"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/require"
)

func TestGetCertificate(t *testing.T) {
	ctx := logr.NewContext(context.Background(), logr.Discard())

	certPath, err := generateCertificateFile()
	require.NoError(t, err)

	certPool, err := GetCertificate(ctx, certPath)

	err = deleteFile(certPath)
	require.NoError(t, err)

	// nolint:staticcheck
	require.Len(t, certPool.Subjects(), 1)

	_, err = GetCertificate(ctx, fmt.Sprintf("%s-does-not-exist", certPath))
	require.Error(t, err)
}

func TestGetStringFromFile(t *testing.T) {
	ctx := logr.NewContext(context.Background(), logr.Discard())
	filePath, expectedFileString, err := generateRandomFile()
	require.NoError(t, err)

	fileString, err := GetStringFromFile(ctx, filePath)

	err = deleteFile(filePath)
	require.NoError(t, err)
	require.Equal(t, expectedFileString, fileString)

	_, err = GetStringFromFile(ctx, fmt.Sprintf("%s-does-not-exist", filePath))
	require.Error(t, err)
}

func TestGetEncodedHash(t *testing.T) {
	testString := "this is a test string"
	testStringHash := GetEncodedHash(testString)
	expectedHash := "f6774519d1c7a3389ef327e9c04766b999db8cdfb85d1346c471ee86d65885bc"
	require.Equal(t, expectedHash, testStringHash)
}

func TestGetBearerToken(t *testing.T) {
	cases := []struct {
		token               string
		reqFunc             func(token string) *http.Request
		expectedErrContains string
	}{
		{
			token: "token",
			reqFunc: func(token string) *http.Request {
				return &http.Request{
					Header: map[string][]string{
						"Authorization": {fmt.Sprintf("Bearer %s", token)},
					},
				}
			},
			expectedErrContains: "",
		},
		{
			token: "",
			reqFunc: func(token string) *http.Request {
				return &http.Request{}
			},
			expectedErrContains: "No Authorization header present in request",
		},
		{
			token: "token",
			reqFunc: func(token string) *http.Request {
				return &http.Request{
					Header: map[string][]string{
						"Authorization": {token},
					},
				}
			},
			expectedErrContains: "Authorization header does not contain Bearer in request",
		},
		{
			token: "Bearer ",
			reqFunc: func(token string) *http.Request {
				return &http.Request{
					Header: map[string][]string{
						"Authorization": {token},
					},
				}
			},
			expectedErrContains: "Empty token",
		},
		{
			token: "Bearer Bearer Bearer ",
			reqFunc: func(token string) *http.Request {
				return &http.Request{
					Header: map[string][]string{
						"Authorization": {token},
					},
				}
			},
			expectedErrContains: "Authorization split by 'Bearer ' isn't length of 2 (actual length: 4)",
		},
		{
			token: "",
			reqFunc: func(token string) *http.Request {
				return &http.Request{
					Header: map[string][]string{
						"Connection": {"upgrade"},
						"Upgrade":    {"websocket"},
					},
				}
			},
			expectedErrContains: "No Sec-WebSocket-Protocol header present in request",
		},
		{
			token: "",
			reqFunc: func(token string) *http.Request {
				req := &http.Request{
					Header: map[string][]string{
						"Connection": {"upgrade"},
						"Upgrade":    {"websocket"},
					},
				}
				req.Header.Add("Sec-WebSocket-Protocol", "fake")
				return req
			},
			expectedErrContains: "Sec-WebSocket-Protocol header does not contain 'base64url.bearer.authorization.k8s.io.' in request",
		},
		{
			token: "",
			reqFunc: func(token string) *http.Request {
				req := &http.Request{
					Header: map[string][]string{
						"Connection": {"upgrade"},
						"Upgrade":    {"websocket"},
					},
				}
				req.Header.Add("Sec-WebSocket-Protocol", "base64url.bearer.authorization.k8s.io.")
				return req
			},
			expectedErrContains: "Empty token",
		},
		{
			token: "",
			reqFunc: func(token string) *http.Request {
				req := &http.Request{
					Header: map[string][]string{
						"Connection": {"upgrade"},
						"Upgrade":    {"websocket"},
					},
				}
				req.Header.Add("Sec-WebSocket-Protocol", "test,abc,base64url.bearer.authorization.k8s.io.,test,abc")
				return req
			},
			expectedErrContains: "Empty token",
		},
		{
			token: "",
			reqFunc: func(token string) *http.Request {
				req := &http.Request{
					Header: map[string][]string{
						"Connection": {"upgrade"},
						"Upgrade":    {"websocket"},
					},
				}
				req.Header.Add("Sec-WebSocket-Protocol", "base64url.bearer.authorization.k8s.io.a====")
				return req
			},
			expectedErrContains: "Unable to base64 decode string in Sec-WebSocket-Protocol",
		},
		{
			token: "fake-token",
			reqFunc: func(token string) *http.Request {
				req := &http.Request{
					Header: map[string][]string{
						"Connection": {"upgrade"},
						"Upgrade":    {"websocket"},
					},
				}
				req.Header.Add("Sec-WebSocket-Protocol", fmt.Sprintf("base64url.bearer.authorization.k8s.io.%s", base64.RawStdEncoding.EncodeToString([]byte(token))))
				return req
			},
			expectedErrContains: "",
		},
	}

	for _, c := range cases {
		req := c.reqFunc(c.token)

		tokenResponse, err := GetBearerToken(req)
		if c.expectedErrContains != "" {
			require.ErrorContains(t, err, c.expectedErrContains)
			continue
		}

		require.NoError(t, err)
		require.Equal(t, c.token, tokenResponse)
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

func TestStripWebSocketBearer(t *testing.T) {
	cases := []struct {
		input  string
		output string
	}{
		{
			input:  "",
			output: "",
		},
		{
			input:  "fake",
			output: "fake",
		},
		{
			input:  "fake,",
			output: "fake, ",
		},
		{
			input:  ",fake",
			output: ", fake",
		},
		{
			input:  ",fake,",
			output: ", fake, ",
		},
		{
			input:  "base64url.bearer.authorization.k8s.io.",
			output: "",
		},
		{
			input:  "base64url.bearer.authorization.k8s.io.,",
			output: "",
		},
		{
			input:  ",base64url.bearer.authorization.k8s.io.,",
			output: ", ",
		},
		{
			input:  "fake, base64url.bearer.authorization.k8s.io., fake",
			output: "fake, fake",
		},
		{
			input:  "fake, base64url.bearer.authorization.k8s.io.fakeToken, fake",
			output: "fake, fake",
		},
		{
			input:  "fake, base64url.bearer.authorization.k8s.io.fakeToken, fake",
			output: "fake, fake",
		},
		{
			input:  "base64url.bearer.authorization.k8s.io.fakeToken",
			output: "",
		},
	}

	for _, c := range cases {
		r := StripWebSocketBearer(c.input)
		require.Equal(t, c.output, r)
	}
}

func TestSliceContains(t *testing.T) {
	cases := []struct {
		inputSlice     []string
		inputString    string
		expectedResult bool
	}{
		{
			inputSlice:     []string{""},
			inputString:    "",
			expectedResult: true,
		},
		{
			inputSlice:     []string{},
			inputString:    "",
			expectedResult: false,
		},
		{
			inputSlice:     []string{"a"},
			inputString:    "a",
			expectedResult: true,
		},
		{
			inputSlice:     []string{"a"},
			inputString:    "b",
			expectedResult: false,
		},
		{
			inputSlice:     []string{"a", "b", "c"},
			inputString:    "b",
			expectedResult: true,
		},
		{
			inputSlice:     []string{"a", "b", "c", "d", "e", "f"},
			inputString:    "g",
			expectedResult: false,
		},
		{
			inputSlice:     []string{"A", "B", "C", "D", "E", "F"},
			inputString:    "a",
			expectedResult: false,
		},
		{
			inputSlice:     []string{"A", "B", "C", "D", "E", "F"},
			inputString:    "F",
			expectedResult: true,
		},
	}

	for _, c := range cases {
		r := SliceContains(c.inputSlice, c.inputString)
		require.Equal(t, c.expectedResult, r)
	}
}

func generateRandomFile() (string, string, error) {
	timestamp := strconv.FormatInt(time.Now().UTC().UnixNano(), 10)
	filename := fmt.Sprintf("test-random-%s.pem", timestamp)
	content := []byte(timestamp)

	err := os.WriteFile(filename, content, 0600)
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
