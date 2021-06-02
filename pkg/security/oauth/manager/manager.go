package manager

import (
	"context"
	"crypto/tls"
	"net/http"
	"sync"
	"time"

	"golang.org/x/oauth2/clientcredentials"

	"github.com/plgd-dev/kit/log"
	"golang.org/x/oauth2"
)

// Manager holds certificates from filesystem watched for changes
type Manager struct {
	mutex                sync.Mutex
	config               clientcredentials.Config
	requestTimeout       time.Duration
	tickFrequency        time.Duration
	nextTokenRenewalTime time.Time
	token                *oauth2.Token
	httpClient           *http.Client
	tokenErr             error
	doneWg               sync.WaitGroup
	done                 chan struct{}
}

// NewManagerFromConfiguration creates a new oauth manager which refreshing token.
func NewManagerFromConfiguration(config Config, tlsCfg *tls.Config) (*Manager, error) {
	cfg := config.ToClientCrendtials()
	t := http.DefaultTransport.(*http.Transport).Clone()
	t.MaxIdleConns = 1
	t.MaxConnsPerHost = 1
	t.MaxIdleConnsPerHost = 1
	t.IdleConnTimeout = time.Second * 30
	t.TLSClientConfig = tlsCfg
	httpClient := &http.Client{
		Transport: t,
		Timeout:   config.RequestTimeout,
	}
	token, nextTokenRenewalTime, err := getToken(cfg, httpClient, config.RequestTimeout)
	if err != nil {
		return nil, err
	}
	log.Infof("client credential token is refreshed, the next refresh token occurs after %v", nextTokenRenewalTime)

	mgr := &Manager{
		config:               cfg,
		token:                token,
		nextTokenRenewalTime: nextTokenRenewalTime,
		requestTimeout:       config.RequestTimeout,
		httpClient:           httpClient,
		tickFrequency:        config.TickFrequency,

		done: make(chan struct{}),
	}
	mgr.doneWg.Add(1)

	go mgr.watchToken()

	return mgr, nil
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
	t := time.NewTicker(a.tickFrequency)
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
