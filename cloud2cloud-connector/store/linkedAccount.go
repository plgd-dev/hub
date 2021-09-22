package store

import (
	"context"

	"github.com/plgd-dev/cloud/pkg/security/oauth2"
)

type LinkedCloudsHandler struct {
	LinkedClouds []LinkedCloud
}

func (h *LinkedCloudsHandler) Handle(ctx context.Context, iter LinkedCloudIter) (err error) {
	for {
		var s LinkedCloud
		if !iter.Next(ctx, &s) {
			break
		}
		h.LinkedClouds = append(h.LinkedClouds, s)
	}
	return iter.Err()
}

type LinkedAccountDataFlags uint8

const (
	// OAuth Access Token of the origin cloud has been obtained
	linkedAccountState_PROVISIONED_ORIGIN_CLOUD LinkedAccountDataFlags = 1 << iota
	// OAuth Access Token of the target cloud has been obtained
	linkedAccountState_PROVISIONED_TARGET_CLOUD_ACCOUNT
)

type LinkedAccountData struct {
	originCloud oauth2.Token
	targetCloud oauth2.Token
	state       LinkedAccountDataFlags
}

// Create linked data with existing origin cloud and target cloud account
func MakeLinkedAccountData(originCloud, targetCloud oauth2.Token) LinkedAccountData {
	return LinkedAccountData{
		originCloud: originCloud,
		targetCloud: targetCloud,
		state:       linkedAccountState_PROVISIONED_ORIGIN_CLOUD | linkedAccountState_PROVISIONED_TARGET_CLOUD_ACCOUNT,
	}
}

func (d LinkedAccountData) HasOrigin() bool {
	return d.state&linkedAccountState_PROVISIONED_ORIGIN_CLOUD != 0
}

func (d LinkedAccountData) SetOrigin(originCloud oauth2.Token) LinkedAccountData {
	d.originCloud = originCloud
	d.state |= linkedAccountState_PROVISIONED_ORIGIN_CLOUD
	return d
}

func (d LinkedAccountData) Origin() oauth2.Token {
	return d.originCloud
}

func (d LinkedAccountData) HasTarget() bool {
	return d.state&linkedAccountState_PROVISIONED_TARGET_CLOUD_ACCOUNT != 0
}

func (d LinkedAccountData) SetTarget(targetCloud oauth2.Token) LinkedAccountData {
	d.targetCloud = targetCloud
	d.state |= linkedAccountState_PROVISIONED_TARGET_CLOUD_ACCOUNT
	return d
}

func (d LinkedAccountData) Target() oauth2.Token {
	return d.targetCloud
}

func (d LinkedAccountData) State() LinkedAccountDataFlags {
	return d.state
}

type LinkedAccount struct {
	ID            string `json:"Id" bson:"_id"`
	LinkedCloudID string `bson:"linkedcloudid"`
	UserID        string
	Data          LinkedAccountData
}
