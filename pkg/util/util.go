package util

import (
	"context"
	"crypto/sha256"
	"crypto/x509"
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

	cert, err := ioutil.ReadFile(path)
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

	byteContent, err := ioutil.ReadFile(path)
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
	h := r.Header.Get("Authorization")
	if h == "" {
		return "", errors.New("No Authorization header present in request")
	}

	if !strings.Contains(h, "Bearer ") {
		return "", errors.New("Authorization header does not contain Bearer in request")
	}

	a := strings.Split(h, "Bearer ")
	if len(a) != 2 {
		return "", fmt.Errorf("Authorization split by 'Bearer ' isn't length of 2 (actual lenght: %d)", len(a))
	}

	token := a[1]

	return token, nil
}
