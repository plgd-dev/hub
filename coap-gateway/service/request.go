package service

import "fmt"

/// Check that all required request fields are set
func checkOAuthRequest(request CoapSignInReq) error {
	if request.UserID == "" {
		return fmt.Errorf("invalid UserId")
	}
	if request.AccessToken == "" {
		return fmt.Errorf("invalid AccessToken")
	}
	return nil
}

/// Update empty values
func (request CoapSignInReq) updateOAUthRequestIfEmpty(deviceID, userID, accessToken string) CoapSignInReq {
	if request.DeviceID == "" {
		request.DeviceID = deviceID
	}
	if request.UserID == "" {
		request.UserID = userID
	}
	if request.AccessToken == "" {
		request.AccessToken = accessToken
	}
	return request
}
