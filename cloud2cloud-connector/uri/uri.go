package uri

// Resource Service URIs.
const (
	API     string = "/api"
	Version string = API + "/v1"

	// GET - retrieve all linked clouds
	// POST - add linked cloud
	LinkedClouds string = Version + "/clouds"
	// DELETE - delete linked cloud
	LinkedCloud string = LinkedClouds + "/{CloudId}"
	// GET - add linked account - params: cloud_id
	LinkedAccounts string = LinkedCloud + "/accounts"
	// DELETE - delete linked account
	LinkedAccount string = LinkedAccounts + "/{AccountId}"
	// POST - new events from target cloud subscriptions
	Events string = Version + "/events"

	// GET
	OAuthCallback string = Version + "/oauthCallback"
)
