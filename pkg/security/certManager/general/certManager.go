package general

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strings"
	"sync"

	"github.com/plgd-dev/hub/v2/pkg/fsnotify"
	"github.com/plgd-dev/hub/v2/pkg/log"
	"github.com/plgd-dev/kit/v2/security"
	"go.uber.org/atomic"
)

// Config provides configuration of a file based Server Certificate manager
type Config struct {
	CAPool                    string `yaml:"caPool" json:"caPool" description:"file path to the root certificates in PEM format"`
	KeyFile                   string `yaml:"keyFile" json:"keyFile" description:"file name of private key in PEM format"`
	CertFile                  string `yaml:"certFile" json:"certFile" description:"file name of certificate in PEM format"`
	ClientCertificateRequired bool   `yaml:"clientCertificateRequired" json:"clientCertificateRequired" description:"require client certificate"`
	UseSystemCAPool           bool   `yaml:"useSystemCAPool" json:"useSystemCaPool" description:"use system certification pool"`
}

func (c Config) Validate() error {
	if c.CAPool == "" && !c.UseSystemCAPool {
		return fmt.Errorf("caPool('%v')", c.CAPool)
	}
	if c.CertFile == "" {
		return fmt.Errorf("certFile('%v')", c.CertFile)
	}
	if c.KeyFile == "" {
		return fmt.Errorf("keyFile('%v')", c.KeyFile)
	}
	return nil
}

// CertManager holds certificates from filesystem watched for changes
type CertManager struct {
	config Config

	fileWatcher             *fsnotify.Watcher
	verifyClientCertificate tls.ClientAuthType
	logger                  log.Logger
	onFileChangeFunc        func(event fsnotify.Event)
	done                    atomic.Bool

	private struct {
		mutex      sync.Mutex
		tlsKeyPair tls.Certificate
		tlsCAPool  *x509.CertPool
	}
}

// New creates a new certificate manager which watches for certs in a filesystem
func New(config Config, fileWatcher *fsnotify.Watcher, logger log.Logger) (*CertManager, error) {
	verifyClientCertificate := tls.RequireAndVerifyClientCert
	if !config.ClientCertificateRequired {
		verifyClientCertificate = tls.NoClientCert
	}

	c := &CertManager{
		fileWatcher:             fileWatcher,
		config:                  config,
		verifyClientCertificate: verifyClientCertificate,
		logger:                  logger,
	}
	err := c.loadCAs()
	if err != nil {
		return nil, err
	}
	err = c.loadCerts()
	if err != nil {
		return nil, err
	}
	if config.CAPool != "" {
		if err := c.fileWatcher.Add(filepath.Dir(config.CAPool)); err != nil {
			return nil, fmt.Errorf("cannot watch CAPool directory(%v): %w", filepath.Dir(config.CAPool), err)
		}
	}
	if config.CertFile != "" {
		if err := c.fileWatcher.Add(filepath.Dir(config.CertFile)); err != nil {
			return nil, fmt.Errorf("cannot watch CertFile directory(%v): %w", filepath.Dir(config.CertFile), err)
		}
	}
	if config.KeyFile != "" {
		if err := c.fileWatcher.Add(filepath.Dir(config.KeyFile)); err != nil {
			return nil, fmt.Errorf("cannot watch KeyFile directory(%v): %w", filepath.Dir(config.KeyFile), err)
		}
	}
	c.onFileChangeFunc = c.onFileChange
	c.fileWatcher.AddOnEventHandler(&c.onFileChangeFunc)

	return c, nil
}

// GetCertificateAuthorities returns certificates authorities
func (a *CertManager) GetCertificateAuthorities() *x509.CertPool {
	a.private.mutex.Lock()
	defer a.private.mutex.Unlock()
	return a.private.tlsCAPool
}

// GetServerTLSConfig returns tls configuration for servers
func (a *CertManager) GetServerTLSConfig() *tls.Config {
	return &tls.Config{
		ClientCAs:      a.GetCertificateAuthorities(),
		GetCertificate: a.getServerCertificate,
		MinVersion:     tls.VersionTLS12,
		ClientAuth:     a.verifyClientCertificate,
	}
}

// GetClientTLSConfig returns tls configuration for clients
func (a *CertManager) GetClientTLSConfig() *tls.Config {
	return &tls.Config{
		RootCAs:                  a.GetCertificateAuthorities(),
		GetClientCertificate:     a.getClientCertificate,
		PreferServerCipherSuites: true,
		MinVersion:               tls.VersionTLS12,
	}
}

// Close ends watching certificates
func (a *CertManager) Close() {
	if !a.done.CAS(false, true) {
		return
	}
	if a.config.CAPool != "" {
		if err := a.fileWatcher.Remove(filepath.Dir(a.config.CAPool)); err != nil {
			a.logger.Errorf("cannot remove fileWatcher for CAPool directory(%v): %w", filepath.Dir(a.config.CAPool), err)
		}
	}
	if a.config.CertFile != "" {
		if err := a.fileWatcher.Remove(filepath.Dir(a.config.CertFile)); err != nil {
			a.logger.Errorf("cannot remove fileWatcher for CertFile directory(%v): %w", filepath.Dir(a.config.CertFile), err)
		}
	}
	if a.config.KeyFile != "" {
		if err := a.fileWatcher.Remove(filepath.Dir(a.config.KeyFile)); err != nil {
			a.logger.Errorf("cannot remove fileWatcher for KeyFile directory(%v): %w", filepath.Dir(a.config.KeyFile), err)
		}
	}
	a.fileWatcher.RemoveOnEventHandler(&a.onFileChangeFunc)
}

func (a *CertManager) getServerCertificate(hello *tls.ClientHelloInfo) (*tls.Certificate, error) {
	a.private.mutex.Lock()
	defer a.private.mutex.Unlock()
	return &a.private.tlsKeyPair, nil
}

func (a *CertManager) getClientCertificate(*tls.CertificateRequestInfo) (*tls.Certificate, error) {
	a.private.mutex.Lock()
	defer a.private.mutex.Unlock()
	return &a.private.tlsKeyPair, nil
}

func (a *CertManager) loadCerts() error {
	if a.config.KeyFile != "" && a.config.CertFile != "" {
		keyPath := a.config.KeyFile
		tlsKey, err := ioutil.ReadFile(keyPath)
		if err != nil {
			return fmt.Errorf("cannot load certificate key from '%v': %w", keyPath, err)
		}
		certPath := a.config.CertFile
		tlsCert, err := ioutil.ReadFile(certPath)
		if err != nil {
			return fmt.Errorf("cannot load certificate from '%v': %w", certPath, err)
		}
		cert, err := tls.X509KeyPair(tlsCert, tlsKey)
		if err != nil {
			return fmt.Errorf("cannot load certificate pair: %w", err)
		}
		a.setTLSKeyPair(cert)
	}
	return nil
}
func (a *CertManager) loadCAs() error {
	var cas []*x509.Certificate
	if a.config.CAPool != "" {
		certs, err := security.LoadX509(a.config.CAPool)
		if err != nil {
			return fmt.Errorf("cannot load certificate authorities from '%v': %w", a.config.CAPool, err)
		}
		cas = certs
	}
	if a.config.UseSystemCAPool {
		a.setCAPool(security.NewDefaultCertPool(cas))
	} else {
		p := x509.NewCertPool()
		for _, c := range cas {
			p.AddCert(c)
		}
		a.setCAPool(p)
	}
	return nil
}

func (a *CertManager) setTLSKeyPair(cert tls.Certificate) {
	a.private.mutex.Lock()
	defer a.private.mutex.Unlock()
	a.private.tlsKeyPair = cert
}

func (a *CertManager) setCAPool(capool *x509.CertPool) {
	a.private.mutex.Lock()
	defer a.private.mutex.Unlock()
	a.private.tlsCAPool = capool
}

func (a *CertManager) onFileChange(event fsnotify.Event) {
	var updateCert, updateKey, updateCAs bool
	switch event.Op {
	case fsnotify.Create, fsnotify.Write, fsnotify.Rename:
		if strings.Contains(event.Name, a.config.KeyFile) {
			updateKey = true
		}

		if strings.Contains(event.Name, a.config.CertFile) {
			updateCert = true
		}

		if strings.Contains(event.Name, a.config.CAPool) {
			updateCAs = true
		}
	}
	if updateCert && updateKey {
		err := a.loadCerts()
		if err != nil {
			a.logger.Error(err.Error())
		}
	}
	if updateCAs {
		err := a.loadCAs()
		if err != nil {
			a.logger.Error(err.Error())
		}
	}
}
