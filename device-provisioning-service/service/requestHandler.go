package service

import (
	"context"

	"github.com/plgd-dev/go-coap/v3/message/pool"
	"github.com/plgd-dev/go-coap/v3/mux"
)

type RequestHandler interface {
	DefaultHandler(ctx context.Context, req *mux.Message, session *Session, linkedHub []*LinkedHub, group *EnrollmentGroup) (*pool.Message, error)
	ProcessOwnership(ctx context.Context, req *mux.Message, session *Session, linkedHub []*LinkedHub, group *EnrollmentGroup) (*pool.Message, error)
	ProcessCredentials(ctx context.Context, req *mux.Message, session *Session, linkedHub []*LinkedHub, group *EnrollmentGroup) (*pool.Message, error)
	ProcessACLs(ctx context.Context, req *mux.Message, session *Session, linkedHub []*LinkedHub, group *EnrollmentGroup) (*pool.Message, error)
	ProcessCloudConfiguration(ctx context.Context, req *mux.Message, session *Session, linkedHub []*LinkedHub, group *EnrollmentGroup) (*pool.Message, error)
	ProcessPlgdTime(ctx context.Context, req *mux.Message, session *Session, linkedHub []*LinkedHub, group *EnrollmentGroup) (*pool.Message, error)
}

type RequestHandle struct{}
