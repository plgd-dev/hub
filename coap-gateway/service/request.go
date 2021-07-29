package service

import "fmt"

/// Check that all required request fields are set
func checkReq(request CoapSignInReq) error {
	if request.UserID == "" {
		return fmt.Errorf("invalid UserId")
	}
	if request.AccessToken == "" {
		return fmt.Errorf("invalid AccessToken")
	}
	return nil
}
