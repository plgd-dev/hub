package service

import (
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/plgd-dev/hub/v2/pkg/fn"
)

type Service struct {
	services []APIService
	done     chan struct{}
	sigs     chan os.Signal
	closeFn  fn.FuncList
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
	var errors []error
	for {
		select {
		case err := <-errCh:
			if err != nil {
				errors = append(errors, err)
			}
		default:
			switch len(errors) {
			case 0:
				return nil
			case 1:
				return errors[0]
			default:
				return fmt.Errorf("%v", errors)
			}
		}
	}
}

// Shutdown turn off server.
func (s *Service) Close() error {
	select {
	case s.sigs <- syscall.SIGTERM:
	default:
	}
	<-s.done
	return nil
}

// AddCloseFunc adds a function to be called when the server is closed
func (s *Service) AddCloseFunc(f func()) {
	s.closeFn.AddFunc(f)
}
