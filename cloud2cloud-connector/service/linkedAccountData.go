package service

import "github.com/go-ocf/cloud/cloud2cloud-connector/store"

type LinkedAccountData struct {
	originCloud   store.LinkedCloud
	LinkedAccount store.LinkedAccount
	State         LinkedAccountState
}

type LinkedAccountState uint8

const (
	LinkedAccountState_START LinkedAccountState = iota
	// OAuth Access Token of the origin cloud has been obtained
	LinkedAccountState_PROVISIONED_ORIGIN_CLOUD
	// OAuth Access Token of the target cloud has been obtained
	LinkedAccountState_PROVISIONED_TARGET_CLOUD
)
