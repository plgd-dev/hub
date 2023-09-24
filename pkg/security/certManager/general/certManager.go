package general

import (
	"bytes"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/hashicorp/go-multierror"
	"github.com/plgd-dev/hub/v2/pkg/fn"
	"github.com/plgd-dev/hub/v2/pkg/fsnotify"
	"github.com/plgd-dev/hub/v2/pkg/log"
	pkgTime "github.com/plgd-dev/hub/v2/pkg/time"
	"github.com/plgd-dev/kit/v2/security"
	"go.uber.org/atomic"
)

// Config provides configuration of a file based Server Certificate manager
type Config struct {
	CAPool                    []string `yaml:"caPool" json:"caPool" description:"file path to the root certificates in PEM format"`
	KeyFile                   string   `yaml:"keyFile" json:"keyFile" description:"file name of private key in PEM format"`
	CertFile                  string   `yaml:"certFile" json:"certFile" description:"file name of certificate in PEM format"`
	ClientCertificateRequired bool     `yaml:"clientCertificateRequired" json:"clientCertificateRequired" description:"require client certificate"`
	UseSystemCAPool           bool     `yaml:"useSystemCAPool" json:"useSystemCaPool" description:"use system certification pool"`
}

func (c Config) Validate() error {
	if len(c.CAPool) == 0 && !c.UseSystemCAPool {
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
		mutex                sync.Mutex
		tlsKeyPair           *tls.Certificate
		tlsCertNotAfter      time.Time
		tlsCAPool            *x509.CertPool
		tlsCAPoolMinNotAfter time.Time
		tlsCAHash            []byte
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
	_, err := c.loadCAs()
	if err != nil {
		return nil, err
	}
	_, err = c.loadCerts()
	if err != nil {
		return nil, err
	}
	var removeFilesOnError fn.FuncList

	for _, ca := range config.CAPool {
		if err := c.fileWatcher.Add(ca); err != nil {
			removeFilesOnError.Execute()
			return nil, fmt.Errorf("cannot watch CAPool(%v): %w", ca, err)
		}
		caToRemove := ca
		removeFilesOnError.AddFunc(func() {
			_ = c.fileWatcher.Remove(caToRemove)
		})
	}
	if config.CertFile != "" {
		if err := c.fileWatcher.Add(config.CertFile); err != nil {
			removeFilesOnError.Execute()
			return nil, fmt.Errorf("cannot watch CertFile(%v): %w", config.CertFile, err)
		}
		removeFilesOnError.AddFunc(func() {
			_ = c.fileWatcher.Remove(config.CertFile)
		})
	}
	if config.KeyFile != "" {
		if err := c.fileWatcher.Add(config.KeyFile); err != nil {
			removeFilesOnError.Execute()
			return nil, fmt.Errorf("cannot watch KeyFile(%v): %w", config.CertFile, err)
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
	if !a.private.tlsCAPoolMinNotAfter.IsZero() && time.Now().After(a.private.tlsCAPoolMinNotAfter) {
		// current CA is invalid - force reload
		_, err := a.loadCAsLocked()
		if err != nil {
			a.logger.Error(err.Error())
		}
	}
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
	if !a.done.CompareAndSwap(false, true) {
		return
	}
	for _, ca := range a.config.CAPool {
		if err := a.fileWatcher.Remove(ca); err != nil {
			a.logger.Errorf("cannot remove fileWatcher for CAPool(%v): %w", ca, err)
		}
	}
	if a.config.CertFile != "" {
		if err := a.fileWatcher.Remove(a.config.CertFile); err != nil {
			a.logger.Errorf("cannot remove fileWatcher for CertFile(%v): %w", a.config.CertFile, err)
		}
	}
	if a.config.KeyFile != "" {
		if err := a.fileWatcher.Remove(a.config.KeyFile); err != nil {
			a.logger.Errorf("cannot remove fileWatcher for KeyFile(%v): %w", a.config.KeyFile, err)
		}
	}
	a.fileWatcher.RemoveOnEventHandler(&a.onFileChangeFunc)
}

func (a *CertManager) getTLSKeyPair() (*tls.Certificate, error) {
	a.private.mutex.Lock()
	defer a.private.mutex.Unlock()
	if !a.private.tlsCertNotAfter.IsZero() && time.Now().After(a.private.tlsCertNotAfter) {
		// current certificate is invalid - force reload
		_, err := a.loadCertsLocked()
		if err != nil {
			return nil, err
		}
	}
	if a.private.tlsKeyPair == nil {
		return nil, fmt.Errorf("certificate is not loaded")
	}

	return a.private.tlsKeyPair, nil
}

func (a *CertManager) getServerCertificate(*tls.ClientHelloInfo) (*tls.Certificate, error) {
	return a.getTLSKeyPair()
}

func (a *CertManager) getClientCertificate(*tls.CertificateRequestInfo) (*tls.Certificate, error) {
	return a.getTLSKeyPair()
}

func (a *CertManager) loadCertsLocked() (bool, error) {
	if a.config.KeyFile == "" || a.config.CertFile == "" {
		return false, nil
	}
	keyPath := a.config.KeyFile
	tlsKey, err := os.ReadFile(keyPath)
	if err != nil {
		return false, fmt.Errorf("cannot load certificate key from '%v': %w", keyPath, err)
	}
	certPath := a.config.CertFile
	tlsCert, err := os.ReadFile(certPath)
	if err != nil {
		return false, fmt.Errorf("cannot load certificate from '%v': %w", certPath, err)
	}
	cert, err := tls.X509KeyPair(tlsCert, tlsKey)
	if err != nil {
		return false, fmt.Errorf("cannot load certificate pair [%s,%s]: %w", keyPath, certPath, err)
	}
	var tlsCertNotAfter time.Time
	certs, err := x509.ParseCertificates(cert.Certificate[0])
	if err != nil {
		return false, fmt.Errorf("cannot load certificate pair [%s,%s]: %w", keyPath, certPath, err)
	}
	if len(certs) == 0 {
		return false, fmt.Errorf("cannot load certificate pair [%s,%s]: no certificates found", keyPath, certPath)
	}
	tlsCertNotAfter = pkgTime.MaxTime
	for _, c := range certs {
		if tlsCertNotAfter.After(c.NotAfter) {
			tlsCertNotAfter = c.NotAfter
		}
	}
	if time.Now().After(tlsCertNotAfter) {
		return false, fmt.Errorf("cannot load certificate pair [%s,%s]: certificate is expired", keyPath, certPath)
	}
	return a.setTLSKeyPairLocked(cert, tlsCertNotAfter), nil
}

func loadCAs(caPool []string) (cas []*x509.Certificate, tlsCAPoolMinNotAfter time.Time, hash []byte, err error) {
	var errors *multierror.Error
	tlsCAPoolMinNotAfter = pkgTime.MaxTime
	now := time.Now()
	h := sha256.New()
	for _, ca := range caPool {
		certs, err := security.LoadX509(ca)
		if err != nil {
			errors = multierror.Append(errors, fmt.Errorf("cannot load CA from '%v': %w", ca, err))
			continue
		}
		for _, c := range certs {
			if now.After(c.NotAfter) {
				// skip expired certificates
				continue
			}
			if tlsCAPoolMinNotAfter.After(c.NotAfter) {
				tlsCAPoolMinNotAfter = c.NotAfter
			}
			_, _ = h.Write(c.Raw)
			cas = append(cas, c)
		}
	}
	if errors.ErrorOrNil() != nil {
		return nil, time.Time{}, nil, errors.ErrorOrNil()
	}
	return cas, tlsCAPoolMinNotAfter, h.Sum(nil), nil
}

func (a *CertManager) loadCAsLocked() (bool, error) {
	cas, tlsCAPoolMinNotAfter, hash, err := loadCAs(a.config.CAPool)
	if err != nil {
		return false, err
	}
	if a.config.UseSystemCAPool {
		return a.setCAPoolLocked(security.NewDefaultCertPool(cas), tlsCAPoolMinNotAfter, hash), nil
	}
	p := x509.NewCertPool()
	for _, c := range cas {
		p.AddCert(c)
	}
	return a.setCAPoolLocked(p, tlsCAPoolMinNotAfter, hash), nil
}

func (a *CertManager) setTLSKeyPairLocked(cert tls.Certificate, tlsCertNotAfter time.Time) bool {
	if a.private.tlsKeyPair != nil && bytes.Equal(a.private.tlsKeyPair.Certificate[0], cert.Certificate[0]) {
		return false
	}

	a.private.tlsKeyPair = &cert
	a.private.tlsCertNotAfter = tlsCertNotAfter
	return true
}

func (a *CertManager) setCAPoolLocked(capool *x509.CertPool, tlsCAPoolMinNotAfter time.Time, hash []byte) bool {
	if bytes.Equal(a.private.tlsCAHash, hash) {
		return false
	}
	a.private.tlsCAPool = capool
	a.private.tlsCAPoolMinNotAfter = tlsCAPoolMinNotAfter
	a.private.tlsCAHash = hash
	return true
}

func (a *CertManager) whatNeedToUpdate(event fsnotify.Event) (updateCert, updateKey, updateCAs bool) {
	if strings.Contains(event.Name, a.config.KeyFile) {
		updateKey = true
	}

	if strings.Contains(event.Name, a.config.CertFile) {
		updateCert = true
	}

	for _, ca := range a.config.CAPool {
		if strings.Contains(event.Name, ca) {
			updateCAs = true
			break
		}
	}
	return
}

func (a *CertManager) loadCerts() (bool, error) {
	a.private.mutex.Lock()
	defer a.private.mutex.Unlock()
	return a.loadCertsLocked()
}

func (a *CertManager) loadCAs() (bool, error) {
	a.private.mutex.Lock()
	defer a.private.mutex.Unlock()
	return a.loadCAsLocked()
}

func (a *CertManager) onFileChange(event fsnotify.Event) {
	updateCert, updateKey, updateCAs := a.whatNeedToUpdate(event)
	if updateCert || updateKey {
		refreshed, err := a.loadCerts()
		if err != nil {
			a.logger.Error(err.Error())
		}
		if refreshed {
			a.logger.Debugf("Refreshing certificates due to modified file(%v) via event %v", event.Name, event.Op)
		}
	}
	if updateCAs {
		refreshed, err := a.loadCAs()
		if err != nil {
			a.logger.Error(err.Error())
		}
		if refreshed {
			a.logger.Debugf("Refreshing certificate authorities due to modified file(%v) via event %v", event.Name, event.Op)
		}
	}
}
