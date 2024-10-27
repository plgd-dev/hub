package grpc

import (
	"bytes"
	"fmt"

	"github.com/plgd-dev/hub/v2/certificate-authority/pb"
	"github.com/plgd-dev/hub/v2/certificate-authority/store"
	"github.com/plgd-dev/hub/v2/pkg/fn"
	"github.com/plgd-dev/hub/v2/pkg/fsnotify"
	"github.com/plgd-dev/hub/v2/pkg/log"
	"go.uber.org/atomic"
)

// CertificateAuthorityServer handles incoming requests.
type CertificateAuthorityServer struct {
	pb.UnimplementedCertificateAuthorityServer

	signerConfig     SignerConfig
	logger           log.Logger
	ownerClaim       string
	store            store.Store
	hubID            string
	fileWatcher      *fsnotify.Watcher
	onFileChangeFunc func(event fsnotify.Event)
	crlServerAddress string

	signer atomic.Pointer[Signer]
}

func NewCertificateAuthorityServer(ownerClaim, hubID, crlServerAddress string, signerConfig SignerConfig, store store.Store, fileWatcher *fsnotify.Watcher, logger log.Logger) (*CertificateAuthorityServer, error) {
	if err := signerConfig.Validate(); err != nil {
		return nil, err
	}

	s := &CertificateAuthorityServer{
		signerConfig:     signerConfig,
		logger:           logger,
		ownerClaim:       ownerClaim,
		store:            store,
		hubID:            hubID,
		fileWatcher:      fileWatcher,
		crlServerAddress: crlServerAddress,
	}

	_, err := s.load()
	if err != nil {
		return nil, err
	}

	var removeFilesOnError fn.FuncList
	for _, ca := range signerConfig.caPoolArray {
		if !ca.IsFile() {
			continue
		}
		if err := fileWatcher.Add(ca.FilePath()); err != nil {
			removeFilesOnError.Execute()
			return nil, fmt.Errorf("cannot watch CAPool(%v): %w", ca, err)
		}
		caToRemove := ca.FilePath()
		removeFilesOnError.AddFunc(func() {
			_ = fileWatcher.Remove(caToRemove)
		})
	}
	if signerConfig.CertFile.IsFile() {
		if err := fileWatcher.Add(signerConfig.CertFile.FilePath()); err != nil {
			removeFilesOnError.Execute()
			return nil, fmt.Errorf("cannot watch CertFile(%v): %w", signerConfig.CertFile, err)
		}
		removeFilesOnError.AddFunc(func() {
			_ = fileWatcher.Remove(signerConfig.CertFile.FilePath())
		})
	}

	if signerConfig.KeyFile.IsFile() {
		if err := fileWatcher.Add(signerConfig.KeyFile.FilePath()); err != nil {
			removeFilesOnError.Execute()
			return nil, fmt.Errorf("cannot watch KeyFile(%v): %w", signerConfig.KeyFile, err)
		}
	}
	s.onFileChangeFunc = s.onFileChange
	fileWatcher.AddOnEventHandler(&s.onFileChangeFunc)

	return s, nil
}

func (s *CertificateAuthorityServer) Close() {
	for _, ca := range s.signerConfig.caPoolArray {
		if !ca.IsFile() {
			continue
		}
		if err := s.fileWatcher.Remove(ca.FilePath()); err != nil {
			s.logger.Errorf("cannot remove fileWatcher for CAPool(%v): %w", ca, err)
		}
	}
	if s.signerConfig.CertFile.IsFile() {
		if err := s.fileWatcher.Remove(s.signerConfig.CertFile.FilePath()); err != nil {
			s.logger.Errorf("cannot remove fileWatcher for CertFile(%v): %w", s.signerConfig.CertFile, err)
		}
	}
	if s.signerConfig.KeyFile.IsFile() {
		if err := s.fileWatcher.Remove(s.signerConfig.KeyFile.FilePath()); err != nil {
			s.logger.Errorf("cannot remove fileWatcher for KeyFile(%v): %w", s.signerConfig.KeyFile, err)
		}
	}
}

func (s *CertificateAuthorityServer) load() (bool, error) {
	signer, err := NewSigner(s.ownerClaim, s.hubID, s.crlServerAddress, s.signerConfig)
	if err != nil {
		return false, fmt.Errorf("cannot create signer: %w", err)
	}

	oldSigner := s.signer.Load()
	if oldSigner != nil && len(signer.certificate) == len(oldSigner.certificate) && bytes.Equal(signer.certificate[0].Raw, oldSigner.certificate[0].Raw) {
		return false, nil
	}
	return s.signer.CompareAndSwap(oldSigner, signer), nil
}

func (s *CertificateAuthorityServer) onFileChange(event fsnotify.Event) {
	ok, err := s.load()
	if err != nil {
		s.logger.Errorf("cannot refresh signer: %v", err)
		return
	}
	if ok {
		s.logger.Debugf("Refreshing signer certificates due to modified file(%v) via event %v", event.Name, event.Op)
	}
}

func (s *CertificateAuthorityServer) GetSigner() *Signer {
	return s.signer.Load()
}
