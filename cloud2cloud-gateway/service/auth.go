package service

import (
	"fmt"
	"strings"

	"github.com/plgd-dev/kit/net/grpc"
)

func parseAuth(ownerClaim, auth string) (token, owner string, err error) {
	if strings.HasPrefix(strings.ToLower(auth), "bearer ") {
		rawToken := auth[7:]
		owner, err = grpc.ParseOwnerFromJwtToken(ownerClaim, rawToken)
		if err != nil {
			err = fmt.Errorf("cannot parse bearer: %w", err)
			return
		}
		token = rawToken
		return
	}
	return "", "", fmt.Errorf("cannot parse bearer: prefix 'Bearer ' not found")
}
