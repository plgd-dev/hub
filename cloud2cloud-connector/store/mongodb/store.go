package mongodb

import (
	"context"
	"crypto/tls"
	"fmt"

	pkgMongo "github.com/plgd-dev/cloud/pkg/mongodb"
)

type Store struct {
	*pkgMongo.Store
}

func NewStore(ctx context.Context, cfg pkgMongo.Config, tls *tls.Config) (*Store, error) {
	m, err := pkgMongo.NewStore(ctx, cfg, tls)
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
	var errors []error
	if err := s.Collection(resLinkedAccountCName).Drop(ctx); err != nil {
		errors = append(errors, err)
	}
	if err := s.Collection(resLinkedCloudCName).Drop(ctx); err != nil {
		errors = append(errors, err)
	}
	if len(errors) > 0 {
		return fmt.Errorf("cannot clear: %v", errors)
	}
	return nil
}
