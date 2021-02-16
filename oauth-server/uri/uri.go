package uri

const (
	a = `
		https://auth.plgd.cloud/authorize?audience=https://try.plgd.cloud&scope=openid profile email&client_id=LXZ9OhKWWRYqf12W0B5OXduqt02q0zjS&redirect_uri=https://try.plgd.cloud&response_type=code&response_mode=query&state=dUU2MzV+M2Q5SFBtNzB0SEcxNk0ybkRjREYuc2NSak1EQmdNOTlZS0lqfg==&nonce=SHdPQk5EU1FTejlkam9TT25TdmhQeGJ5N0k0SzdDNmhDdlNpUm81OGRaRA==&code_challenge=pc7X8YsGb9_-fuO89UG0auNDO0xFKQzIAPtQN36bNJs&code_challenge_method=S256&auth0Client=eyJuYW1lIjoiYXV0aDAtcmVhY3QiLCJ2ZXJzaW9uIjoiMS4yLjAifQ==
	fetch("https://auth.plgd.cloud/oauth/token", {
  "headers": {
    "accept": "*/*",
    "accept-language": "en-US,en;q=0.9",
    "content-type": "application/json",
    "sec-fetch-dest": "empty",
    "sec-fetch-mode": "cors",
    "sec-fetch-site": "same-site"
  },
  "referrer": "https://try.plgd.cloud/?code=HLtob9_hKSYY5y0G&state=V1g1SG5LcWxHb1pfQnR2NH54Z080VXBYYS1wWUpsMHpORVk0bmwwSWZnZA%3D%3D",
  "referrerPolicy": "no-referrer-when-downgrade",
  "body": "{\"redirect_uri\":\"https://try.plgd.cloud\",\"client_id\":\"LXZ9OhKWWRYqf12W0B5OXduqt02q0zjS\",\"code_verifier\":\"5tapdqpCdzttP8k9sEAgH2aXfUE3yRZT03GIq8t~4C~\",\"grant_type\":\"authorization_code\",\"code\":\"HLtob9_hKSYY5y0G\"}",
  "method": "POST",
  "mode": "cors",
  "credentials": "omit"
});
`
	RedirectURIQueryKey = "redirect_uri"
	StateQueryKey       = "state"
	ClientIDQueryKey    = "client_id"
	NonceQueryKey       = "nonce"
	CodeQueryKey        = "code"

	Token     = "/oauth/token"
	Authorize = "/authorize"
	UserInfo  = Authorize + "/userinfo"
	JWKs      = "/.well-known/jwks.json"
)
