package x509util

import (
	"crypto/x509"
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
)

// ReadCertPool loads a certificate pool from disk. The given path can be a
// file, a directory, or a comma-separated list of files.
func ReadCertPool(path string) (*x509.CertPool, error) {
	info, err := os.Stat(path)
	if err != nil && !os.IsNotExist(err) {
		return nil, errors.Wrap(err, "error reading cert pool")
	}

	var (
		files []string
		pool  = x509.NewCertPool()
	)
	if info != nil && info.IsDir() {
		finfos, err := os.ReadDir(path)
		if err != nil {
			return nil, errors.Wrap(err, "error reading cert pool")
		}
		for _, finfo := range finfos {
			files = append(files, filepath.Join(path, finfo.Name()))
		}
	} else {
		files = strings.Split(path, ",")
		for i := range files {
			files[i] = strings.TrimSpace(files[i])
		}
	}

	var found bool
	for _, f := range files {
		bytes, err := os.ReadFile(f)
		if err != nil {
			return nil, errors.Wrap(err, "error reading cert pool")
		}
		if ok := pool.AppendCertsFromPEM(bytes); ok {
			found = true
		}
	}
	if !found {
		return nil, errors.New("error reading cert pool: not certificates found")
	}
	return pool, nil
}
