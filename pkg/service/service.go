package service

import (
	"errors"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/hashicorp/go-multierror"
	"github.com/plgd-dev/hub/v2/pkg/fn"
	"go.uber.org/atomic"
)

type Service struct {
	services []APIService
	done     chan struct{}
	sigs     chan os.Signal
	closeFn  fn.FuncList
	serving  atomic.Bool
}

type APIService interface {
	Serve() error
	Close() error
}

func New(services ...APIService) *Service {
	return &Service{
		sigs:     make(chan os.Signal, 1),
		done:     make(chan struct{}),
		services: services,
	}
}

// Add adds other API services. This needs to be called before Serve.
func (s *Service) Add(services ...APIService) {
	s.services = append(s.services, services...)
}

func (s *Service) Serve() error {
	if !s.serving.CompareAndSwap(false, true) {
		return errors.New("already serving")
	}
	defer close(s.done)
	var wg sync.WaitGroup
	errCh := make(chan error, len(s.services)*2)
	wg.Add(len(s.services))
	for _, apiService := range s.services {
		go func(serve func() error) {
			defer wg.Done()
			err := serve()
			errCh <- err
		}(apiService.Serve)
	}

	signal.Notify(s.sigs,
		syscall.SIGHUP,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT)
	<-s.sigs
	for _, apiService := range s.services {
		err := apiService.Close()
		errCh <- err
	}
	wg.Wait()
	s.closeFn.Execute()
	var errors *multierror.Error
	for {
		select {
		case err := <-errCh:
			if err != nil {
				errors = multierror.Append(errors, err)
			}
		default:
			return errors.ErrorOrNil()
		}
	}
}

func (s *Service) Signal(sig os.Signal) {
	select {
	case s.sigs <- sig:
	default:
	}
}

func (s *Service) SigTerm() {
	s.Signal(syscall.SIGTERM)
}

// Shutdown turn off server.
func (s *Service) Close() error {
	s.SigTerm()
	if !s.serving.Load() {
		s.closeFn.Execute()
		return nil
	}
	<-s.done
	return nil
}

// AddCloseFunc adds a function to be called when the server is closed
func (s *Service) AddCloseFunc(f func()) {
	s.closeFn.AddFunc(f)
}
