package pb

func copySlice(scopes []string) []string {
	if len(scopes) == 0 {
		return nil
	}
	sc := make([]string, len(scopes))
	copy(sc, scopes)
	return sc
}

func (r *OAuthClient) Clone() *OAuthClient {
	if r == nil {
		return nil
	}
	return &OAuthClient{
		ClientId:         r.GetClientId(),
		Audience:         r.GetAudience(),
		Scopes:           copySlice(r.GetScopes()),
		ProviderName:     r.GetProviderName(),
		GrantTypes:       copySlice(r.GetGrantTypes()),
		UseJwtPrivateKey: r.GetUseJwtPrivateKey(),
		Authority:        r.GetAuthority(),
	}
}
