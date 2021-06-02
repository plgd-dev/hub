package manager

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"sync"
	"time"

	"go.uber.org/zap"
	"golang.org/x/oauth2/clientcredentials"

	"github.com/plgd-dev/cloud/pkg/log"
	"github.com/plgd-dev/cloud/pkg/net/http/client"
	"golang.org/x/oauth2"
)

// Manager holds certificates from filesystem watched for changes
type Manager struct {
	mutex                       sync.Mutex
	config                      clientcredentials.Config
	requestTimeout              time.Duration
	verifyServiceTokenFrequency time.Duration
	nextTokenRenewalTime        time.Time
	token                       *oauth2.Token
	httpClient                  *http.Client
	tokenErr                    error
	doneWg                      sync.WaitGroup
	done                        chan struct{}

	http *client.Client
}

// NewManagerFromConfiguration creates a new oauth manager which refreshing token.
func NewManagerFromConfiguration(config Config, tlsCfg *tls.Config) (*Manager, error) {
	cfg := config.ToClientCrendtials()
	t := http.DefaultTransport.(*http.Transport).Clone()
	t.MaxIdleConns = 100
	t.MaxConnsPerHost = 100
	t.MaxIdleConnsPerHost = 1
	t.IdleConnTimeout = time.Second * 30
	t.TLSClientConfig = tlsCfg
	m, err := new(cfg, &http.Client{
		Transport: t,
		Timeout:   config.RequestTimeout,
	}, config.RequestTimeout, config.TickFrequency)
	if err != nil {
		return nil, err
	}

	return m, nil
}

func new(cfg clientcredentials.Config, httpClient *http.Client, requestTimeout, verifyServiceTokenFrequency time.Duration) (*Manager, error) {
	token, nextTokenRenewalTime, err := getToken(cfg, httpClient, requestTimeout)
	if err != nil {
		return nil, err
	}
	log.Infof("client credential token is refreshed, the next refresh token occurs after %v", nextTokenRenewalTime)

	mgr := &Manager{
		config:                      cfg,
		token:                       token,
		nextTokenRenewalTime:        nextTokenRenewalTime,
		requestTimeout:              requestTimeout,
		httpClient:                  httpClient,
		verifyServiceTokenFrequency: verifyServiceTokenFrequency,

		done: make(chan struct{}),
	}
	mgr.doneWg.Add(1)

	go mgr.watchToken()

	return mgr, nil
}

func New(config ConfigV2, logger *zap.Logger) (*Manager, error) {
	http, err := client.New(config.HTTP, logger)
	if err != nil {
		return nil, fmt.Errorf("cannot create http client: %w", err)
	}
	m, err := new(config.ToClientCrendtials(), http.HTTP(), config.HTTP.Timeout, config.VerifyServiceTokenFrequency)
	if err != nil {
		return nil, err
	}
	m.http = http
	return m, nil
}

// GetToken returns token for clients
func (a *Manager) GetToken(ctx context.Context) (*oauth2.Token, error) {
	a.mutex.Lock()
	defer a.mutex.Unlock()
	return a.token, a.tokenErr
}

// Close ends watching token
func (a *Manager) Close() {
	if a.done != nil {
		close(a.done)
		a.doneWg.Wait()
		if a.http != nil {
			a.http.Close()
		}
	}
}

func (a *Manager) shouldRefresh() bool {
	/*
		We cannot use time.Now().After(a.nextTokenRenewalTime ) because
		golang using monotonic clock for comparision.

		So if we have 2 times:
		// update time on PC to future eg: `date MMDDHHMM`
		t1 := time.Now() eg (2021-06-15T12:00:00)
		// return back time on PC: `date MMDDHHMM`
		t2 := time.Now() eg (2021-06-01T12:00:00)
		and then you call t2.After(t1) - it's return true :)

		more info: https://github.com/golang/go/blob/master/src/time/time.go

		the issue can occurs when pc hibernates.
	*/

	return time.Now().UnixNano() > a.nextTokenRenewalTime.UnixNano()
}

func getToken(cfg clientcredentials.Config, httpClient *http.Client, requestTimeout time.Duration) (*oauth2.Token, time.Time, error) {
	ctx, cancel := context.WithTimeout(context.Background(), requestTimeout)
	defer cancel()

	ctx = context.WithValue(ctx, oauth2.HTTPClient, httpClient)

	token, err := cfg.Token(ctx)
	var nextTokenRenewalTime time.Time
	if err == nil {
		now := time.Now()
		nextTokenRenewalTime = now.Add(token.Expiry.Sub(now) * 2 / 3)
	}
	return token, nextTokenRenewalTime, err
}

func (a *Manager) refreshToken() {
	token, nextTokenRenewalTime, err := getToken(a.config, a.httpClient, a.requestTimeout)
	if err != nil {
		log.Errorf("cannot refresh token: %v", err)
	} else {
		log.Infof("client credential token is refreshed, the next refresh token occurs after %v", nextTokenRenewalTime)
	}
	a.mutex.Lock()
	defer a.mutex.Unlock()
	a.token = token
	a.tokenErr = err
	a.nextTokenRenewalTime = nextTokenRenewalTime
}

func (a *Manager) watchToken() {
	defer a.doneWg.Done()
	t := time.NewTicker(a.verifyServiceTokenFrequency)
	defer t.Stop()

	for {
		select {
		case <-a.done:
			return
		case <-t.C:
			if a.shouldRefresh() {
				a.refreshToken()
			}
		}
	}
}
