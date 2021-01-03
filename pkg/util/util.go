package util

import (
	"context"
	"crypto/x509"
	"io/ioutil"

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
