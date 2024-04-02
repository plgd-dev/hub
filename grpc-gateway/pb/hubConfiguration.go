package pb

func copyScopes(scopes []string) []string {
	if len(scopes) == 0 {
		return nil
	}
	sc := make([]string, len(scopes))
	copy(sc, scopes)
	return sc
}

func (r *WebOAuthClient) Clone() *WebOAuthClient {
	if r == nil {
		return nil
	}
	return &WebOAuthClient{
		ClientId: r.GetClientId(),
		Audience: r.GetAudience(),
		Scopes:   copyScopes(r.GetScopes()),
	}
}

func (r *DeviceOAuthClient) Clone() *DeviceOAuthClient {
	if r == nil {
		return nil
	}
	return &DeviceOAuthClient{
		ProviderName: r.GetProviderName(),
		ClientId:     r.GetClientId(),
		Audience:     r.GetAudience(),
		Scopes:       copyScopes(r.GetScopes()),
	}
}
