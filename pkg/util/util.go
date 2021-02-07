package util

import (
	"context"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/go-logr/logr"
)

// GetCertificate returns a *x509.CertPool or error
func GetCertificate(ctx context.Context, path string) (*x509.CertPool, error) {
	log := logr.FromContext(ctx)

	cert, err := ioutil.ReadFile(path) // #nosec
	if err != nil {
		log.Error(err, "Unable to read certificate file", "certificate-file-path", path)
		return nil, err
	}

	certPool := x509.NewCertPool()
	certPool.AppendCertsFromPEM(cert)

	return certPool, nil
}

// GetStringFromFile returns a string or error
func GetStringFromFile(ctx context.Context, path string) (string, error) {
	log := logr.FromContext(ctx)

	byteContent, err := ioutil.ReadFile(path) // #nosec
	if err != nil {
		log.Error(err, "Unable to read string from file", "file-path", path)
		return "", err
	}

	stringContent := string(byteContent)

	return stringContent, nil
}

// GetEncodedHash returns the hex encoded sha256 sum of the input string
func GetEncodedHash(s string) string {
	hash := sha256.Sum256([]byte(s))
	encodedHash := hex.EncodeToString(hash[:])
	return encodedHash
}

// GetBearerToken returns the Bearer token or an error
func GetBearerToken(r *http.Request) (string, error) {
	isWebSocket := false
	if strings.EqualFold(r.Header.Get("Connection"), "upgrade") && r.Header.Get("Upgrade") == "websocket" {
		isWebSocket = true
	}

	if !isWebSocket {
		h := r.Header.Get("Authorization")
		if h == "" {
			return "", errors.New("No Authorization header present in request")
		}

		if !strings.Contains(h, "Bearer ") {
			return "", errors.New("Authorization header does not contain Bearer in request")
		}

		a := strings.Split(h, "Bearer ")
		if len(a) != 2 {
			return "", fmt.Errorf("Authorization split by 'Bearer ' isn't length of 2 (actual length: %d)", len(a))
		}

		token := a[1]
		if token == "" {
			return "", fmt.Errorf("Empty token")
		}

		return token, nil
	}

	h := r.Header.Get("Sec-WebSocket-Protocol")
	if h == "" {
		return "", errors.New("No Sec-WebSocket-Protocol header present in request")
	}

	//
	if !strings.Contains(h, "base64url.bearer.authorization.k8s.io.") {
		return "", errors.New("Sec-WebSocket-Protocol header does not contain 'base64url.bearer.authorization.k8s.io.' in request")
	}

	if !strings.Contains(h, ", base64.binary.k8s.io") {
		return "", errors.New("Sec-WebSocket-Protocol header does not contain ', base64.binary.k8s.io' in request")
	}

	a := strings.TrimPrefix(h, "base64url.bearer.authorization.k8s.io.")
	a = strings.Split(a, ", base64.binary.k8s.io")[0]

	byteToken, err := base64.RawStdEncoding.DecodeString(a)
	if err != nil {
		return "", errors.New("Unable to base64 decode string in Sec-WebSocket-Protocol")
	}

	token := string(byteToken)

	if token == "" {
		return "", fmt.Errorf("Empty token")
	}

	return token, nil
}
