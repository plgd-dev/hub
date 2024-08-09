package service

import (
	"context"
	"crypto/x509"
	"errors"
	"sync"
	"time"

	"github.com/plgd-dev/hub/v2/device-provisioning-service/pb"
	"github.com/plgd-dev/hub/v2/device-provisioning-service/store"
	"github.com/plgd-dev/hub/v2/device-provisioning-service/store/mongodb"
	"github.com/plgd-dev/hub/v2/pkg/log"
	"go.uber.org/atomic"
)

type EnrollmentGroup struct {
	*pb.EnrollmentGroup
	AttestationMechanismX509CertificateChain []*x509.Certificate
	expireAt                                 atomic.Time
}

type enrollmentGroupsElement = sync.Map // map[EnrollmentGroup.Id]*EnrollmentGroup

type EnrollmentGroupsCache struct {
	ctx        context.Context
	cancel     context.CancelFunc
	cache      sync.Map // map[issuerName]*enrollmentGroupsElement
	cacheByID  sync.Map // map[id]*EnrollmentGroup
	store      *mongodb.Store
	expiration time.Duration
	wg         sync.WaitGroup
}

func NewEnrollmentGroupsCache(ctx context.Context, expiration time.Duration, store *mongodb.Store, logger log.Logger) *EnrollmentGroupsCache {
	ctx, cancel := context.WithCancel(ctx)
	eg := EnrollmentGroupsCache{
		store:      store,
		expiration: expiration,
		ctx:        ctx,
		cancel:     cancel,
	}
	eg.wg.Add(1)
	go func() {
		defer eg.wg.Done()
		err := eg.watch()
		if err != nil {
			logger.Warnf("cannot watch for DB changes in enrollment groups: %v", err)
		}
	}()
	return &eg
}

func (c *EnrollmentGroupsCache) storeEnrollmentGroup(issuerName string, g *EnrollmentGroup) bool {
	v, _ := c.cache.LoadOrStore(issuerName, new(enrollmentGroupsElement))
	e, ok := v.(*enrollmentGroupsElement)
	if !ok {
		return false
	}
	val, ok := e.LoadOrStore(g.GetId(), g)
	c.cacheByID.Store(g.GetId(), val)
	return !ok
}

func (c *EnrollmentGroupsCache) removeByID(id string) {
	// try to remove from cacheByID
	val, ok := c.cacheByID.LoadAndDelete(id)
	if !ok {
		return
	}
	// try to remove from cache
	g, ok := val.(*EnrollmentGroup)
	if !ok {
		return
	}
	certName := g.GetAttestationMechanism().GetX509().GetLeadCertificateName()
	element, ok := c.cache.Load(certName)
	if !ok {
		return
	}
	e, ok := element.(*enrollmentGroupsElement)
	if !ok {
		c.cache.Delete(certName)
		return
	}
	e.Delete(id)
}

func (c *EnrollmentGroupsCache) watch() error {
	iter, err := c.store.WatchEnrollmentGroups(c.ctx)
	if err != nil {
		return err
	}
	for {
		_, id, ok := iter.Next(c.ctx)
		if !ok {
			break
		}
		c.removeByID(id)
	}
	err = iter.Err()
	errClose := iter.Close()
	if err == nil {
		err = errClose
	}
	if errors.Is(err, context.Canceled) {
		return nil
	}
	return err
}

func (c *EnrollmentGroupsCache) Close() {
	c.cancel()
	c.wg.Wait()
}

func (c *EnrollmentGroupsCache) getEnrollmentGroupsFromCache(issuerNames []string, onEnrollmentGroup func(g *EnrollmentGroup) (wantNext bool)) bool {
	for _, issuerName := range issuerNames {
		v, ok := c.cache.Load(issuerName)
		if !ok {
			continue
		}
		e, ok := v.(*enrollmentGroupsElement)
		if !ok {
			continue
		}
		wantNext := true
		e.Range(func(_, value any) bool {
			if g, ok := value.(*EnrollmentGroup); ok {
				g.expireAt.Store(time.Now().Add(c.expiration))
				wantNext = onEnrollmentGroup(g)
				return wantNext
			}
			return true
		})
		if !wantNext {
			return true
		}
	}
	return false
}

func (c *EnrollmentGroupsCache) iterateOverGroups(ctx context.Context, iter store.EnrollmentGroupIter, onEnrollmentGroup func(g *EnrollmentGroup) bool) error {
	for {
		var g pb.EnrollmentGroup
		if !iter.Next(ctx, &g) {
			break
		}
		var attestationMechanismX509CertificateChain []*x509.Certificate
		var err error
		if g.GetAttestationMechanism().GetX509().GetCertificateChain() != "" {
			attestationMechanismX509CertificateChain, err = g.GetAttestationMechanism().GetX509().ResolveCertificateChain()
			if err != nil {
				log.Errorf("cannot parse AttestationMechanism.X509.CertificateChain for enrollment group %v", g.GetId())
				continue
			}
		}
		eg := &EnrollmentGroup{
			EnrollmentGroup:                          &g,
			AttestationMechanismX509CertificateChain: attestationMechanismX509CertificateChain,
		}
		eg.expireAt.Store(time.Now().Add(c.expiration))
		if !c.storeEnrollmentGroup(g.GetAttestationMechanism().GetX509().GetLeadCertificateName(), eg) {
			continue
		}
		if !onEnrollmentGroup(eg) {
			break
		}
	}
	return iter.Err()
}

func (c *EnrollmentGroupsCache) GetEnrollmentGroupsByIssuerNames(ctx context.Context, issuerNames []string, onEnrollmentGroup func(g *EnrollmentGroup) bool) error {
	if c.getEnrollmentGroupsFromCache(issuerNames, onEnrollmentGroup) {
		return nil
	}

	return c.store.LoadEnrollmentGroups(ctx, "", &pb.GetEnrollmentGroupsRequest{
		AttestationMechanismX509CertificateNames: issuerNames,
	}, func(ctx context.Context, iter store.EnrollmentGroupIter) (err error) {
		return c.iterateOverGroups(ctx, iter, onEnrollmentGroup)
	})
}

func compareEnrollmentGroupLeafCertificate(cert *x509.Certificate, g *EnrollmentGroup) bool {
	if len(g.AttestationMechanismX509CertificateChain) == 0 {
		return false
	}

	return cert.CheckSignatureFrom(g.AttestationMechanismX509CertificateChain[0]) == nil
}

func (c *EnrollmentGroupsCache) GetEnrollmentGroup(ctx context.Context, chains [][]*x509.Certificate) (*EnrollmentGroup, bool, error) {
	var eg *EnrollmentGroup
	for _, chain := range chains {
		for _, cert := range chain {
			err := c.GetEnrollmentGroupsByIssuerNames(ctx, []string{cert.Issuer.CommonName}, func(g *EnrollmentGroup) bool {
				if compareEnrollmentGroupLeafCertificate(cert, g) {
					eg = g
					return false
				}
				return true
			})
			if err != nil {
				return nil, false, err
			}
			if eg != nil {
				return eg, true, nil
			}
		}
	}

	return nil, false, nil
}

func (c *EnrollmentGroupsCache) CheckExpirations(t time.Time) {
	c.cache.Range(func(issuerName, element any) bool {
		e, ok := element.(*enrollmentGroupsElement)
		if !ok {
			c.cache.Delete(issuerName)
			return true
		}
		wantToDelete := true
		e.Range(func(id, enrollmentGroup any) bool {
			g, ok := enrollmentGroup.(*EnrollmentGroup)
			if !ok {
				e.Delete(id)
				return true
			}
			if g.expireAt.Load().After(t) {
				e.Delete(id)
				c.cacheByID.Delete(id)
			} else {
				wantToDelete = false
			}
			return true
		})
		if wantToDelete {
			c.cache.Delete(issuerName)
		}
		return true
	})
}
