package util

import (
	"context"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"time"

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

// Retry will return error if function doesn't succeed within the timeout
func Retry(attempts int, sleep time.Duration, f func() error) error {
	var err error

	for i := 0; ; i++ {
		err = f()
		if err == nil {
			return err
		}

		if i >= (attempts - 1) {
			break
		}

		time.Sleep(sleep)
	}
	return fmt.Errorf("after %d attempts, last error: %s", attempts, err)
}
