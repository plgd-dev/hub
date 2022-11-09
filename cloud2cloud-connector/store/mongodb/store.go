package mongodb

import (
	"context"
	"crypto/tls"

	"github.com/hashicorp/go-multierror"
	pkgMongo "github.com/plgd-dev/hub/v2/pkg/mongodb"
	"go.opentelemetry.io/otel/trace"
)

type Store struct {
	*pkgMongo.Store
}

func NewStore(ctx context.Context, cfg pkgMongo.Config, tls *tls.Config, tracerProvider trace.TracerProvider) (*Store, error) {
	m, err := pkgMongo.NewStore(ctx, cfg, tls, tracerProvider)
	if err != nil {
		return nil, err
	}
	s := Store{m}
	s.SetOnClear(func(c context.Context) error {
		return s.clearDatabases(ctx)
	})
	return &s, nil
}

func (s *Store) clearDatabases(ctx context.Context) error {
	var errors *multierror.Error
	if err := s.Collection(resLinkedAccountCName).Drop(ctx); err != nil {
		errors = multierror.Append(errors, err)
	}
	if err := s.Collection(resLinkedCloudCName).Drop(ctx); err != nil {
		errors = multierror.Append(errors, err)
	}
	return errors.ErrorOrNil()
}
