package uri

// Resource Service URIs.
const (
	API     string = "/api"
	Version string = API + "/v1"

	// GET - retrieve all linked clouds
	// POST - add linked cloud
	LinkedClouds string = Version + "/linkedclouds"

	// DELETE - delete linked cloud
	LinkedCloud string = LinkedClouds + "/{{ .LinkedCloudId }}"

	// GET - add linked account - params: target_url, target_linked_cloud_id
	LinkedAccounts string = Version + "/linkedaccounts"
	// GET - retrieve all linked accounts
	RetrieveLinkedAccounts string = Version + "/linkedaccounts/retrieve"

	// DELETE - delete linked account
	LinkedAccount string = LinkedAccounts + "/{{ .LinkedAccountId }}"

	// POST - new events from target cloud subscriptions
	NotifyLinkedAccount string = Version + "/linkedaccountsevents"

	// GET
	//OAuthCallback = is loaded from ENV var
)
