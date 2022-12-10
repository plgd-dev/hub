package pb

func (r *WebOAuthClient) Clone() *WebOAuthClient {
	if r == nil {
		return nil
	}
	var scopes []string
	if len(r.Scopes) > 0 {
		scopes = make([]string, len(r.Scopes))
		copy(scopes, r.Scopes)
	}
	return &WebOAuthClient{
		ClientId: r.ClientId,
		Audience: r.Audience,
		Scopes:   scopes,
	}
}

func (r *DeviceOAuthClient) Clone() *DeviceOAuthClient {
	if r == nil {
		return nil
	}
	var scopes []string
	if len(r.Scopes) > 0 {
		scopes = make([]string, len(r.Scopes))
		copy(scopes, r.Scopes)
	}
	return &DeviceOAuthClient{
		ProviderName: r.ProviderName,
		ClientId:     r.ClientId,
		Audience:     r.Audience,
		Scopes:       scopes,
	}
}
