package service

import (
	"fmt"
	"strings"

	"github.com/plgd-dev/cloud/pkg/net/grpc"
)

func ParseAuth(ownerClaim, auth string) (token, sub string, err error) {
	if strings.HasPrefix(strings.ToLower(auth), "bearer ") {
		rawToken := auth[7:]

		sub, err = grpc.ParseOwnerFromJwtToken(ownerClaim, rawToken)
		if err != nil {
			err = fmt.Errorf("cannot parse owner from bearer: %w", err)
			return
		}
		token = rawToken
		return
	}
	return "", "", fmt.Errorf("cannot parse bearer: prefix 'Bearer ' not found")
}
