package util

import (
	"context"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/go-logr/logr"
)

// GetCertificate returns a *x509.CertPool or error
func GetCertificate(ctx context.Context, path string) (*x509.CertPool, error) {
	log := logr.FromContext(ctx)

	cert, err := os.ReadFile(path) // #nosec
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

	byteContent, err := os.ReadFile(path) // #nosec
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

	if !strings.Contains(strings.ToLower(h), "base64url.bearer.authorization.k8s.io.") {
		return "", errors.New("Sec-WebSocket-Protocol header does not contain 'base64url.bearer.authorization.k8s.io.' in request")
	}

	var bearer string
	if !strings.Contains(h, ",") {
		bearer = strings.TrimPrefix(h, "base64url.bearer.authorization.k8s.io.")
	}

	if strings.Contains(h, ",") {
		a := strings.Split(h, ",")
		for _, s := range a {
			if strings.Contains(strings.ToLower(s), "base64url.bearer.authorization.k8s.io.") {
				bearer = strings.TrimPrefix(s, "base64url.bearer.authorization.k8s.io.")
			}
		}
	}

	byteToken, err := base64.RawStdEncoding.DecodeString(bearer)
	if err != nil {
		return "", errors.New("Unable to base64 decode string in Sec-WebSocket-Protocol")
	}

	token := string(byteToken)

	if token == "" {
		return "", errors.New("Empty token")
	}

	return token, nil
}

// StripWebSocketBearer takes the string from the header Sec-WebSocket-Protocol (r.Header.Get("Sec-WebSocket-Protocol")) and removes any bearer (base64url.bearer.authorization.k8s.io.<bearer>)
func StripWebSocketBearer(old string) string {
	wsProtoArray := []string{}
	if strings.Contains(old, ",") {
		a := strings.Split(old, ",")
		for _, s := range a {
			if !strings.Contains(strings.ToLower(s), "base64url.bearer.authorization.k8s.io.") {
				wsProtoArray = append(wsProtoArray, strings.TrimSpace(s))
			}
		}
	}

	if !strings.Contains(old, ",") {
		if !strings.Contains(strings.ToLower(old), "base64url.bearer.authorization.k8s.io.") {
			wsProtoArray = append(wsProtoArray, strings.TrimSpace(old))
		}
	}

	return strings.Join(wsProtoArray, ", ")
}

// SliceContains checks if a slice contains a string
func SliceContains(s []string, str string) bool {
	for _, v := range s {
		if v == str {
			return true
		}
	}
	return false
}
