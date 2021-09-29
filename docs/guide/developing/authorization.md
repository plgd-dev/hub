# Authorization

## Using external OAuth Server with bundle

Even though the bundle start core plgd services as processes in a single container, a user has still a possibility to configure most of the services parameters. **For testing purposes**, the external OAuth Server (e.g. [Auth0](https://auth0.com)) can be set up.
To skip internal mocked OAuth Server and switch to your external one, configure following environment variables:

```yaml
    OAUTH_AUDIENCE: https://api.example.com
    OAUTH_ENDPOINT: auth.example.com
    OAUTH_CLIENT_ID: ij12OJj2J23K8KJs
    OAUTH_CLIENT_SECRET: 654hkja12asd123d
    OWNER_CLAIM: sub
```

### How to configure Auth0

Assuming you have an account in the Auth0 OAuth as a service, you need to create 2 Applications and one API. Follow these steps to successfully configure bundle to run against your Auth0 instance.

1. Create new **API** in the APIs section
    a. Use name of your choice
    b. Set a unique API identifier (e.g. `https://api.example.com`)
    c. After saving open details of newly created api and **Enable Offline Access**
    d. Store the internal Auth0 API Id required for the step 2c
    e. Switch to **Permissions** tab and add `openid` scope to the list
1. Create new **Regular Web Application** in the Application section
    a. Make sure **Token Endpoint Authentication Method** is set to `None`
    b. Add `https://{FQDN}:{NGINX_PORT}` and `https://{FQDN}:{NGINX_PORT}/api/v1/oauth/callback` to **Allowed Callback URLs**
    c. Add `https://{FQDN}:{NGINX_PORT}` to **Allowed Logout URLs**
    d. Add `https://{FQDN}:{NGINX_PORT}` to **Allowed Web Origins**
    e. Open **Advanced Settings**, switch to **OAuth** tab and paste here the API Id from the step 1d
    f. Switch to **Grant Types** and make sure **only** `Implicit`, `Authorization Code` and `Refresh Token` grants are enabled
1. Create new **Machine to Machine Application** in the Application section
    a. Set **Token Endpoint Authentication Method** to `Post`
    b. Add `https://{FQDN}:{NGINX_PORT}` to **Allowed Callback URLs**
    c. Add `https://{FQDN}:{NGINX_PORT}` to **Allowed Web Origins**
    d. Open **Advanced Settings**, switch to **OAuth** tab and paste here the API Id from the step 1d
    e. Switch to **Grant Types** and make sure **only** `Client Credentials` grant is enabled

## Device ownership configuration

Devices are in the authorization service organized by the owner ID retrieved from the JWT token. The plgd API will based on this value identify the user and grant him the permission only to devices he owns. By default, JWT claim `sub` is used as the owner ID. In case you connect the plgd authorization service with the Auth0, each logged-in user can access only his devices. This behaviour can be changed by changing the `OWNER_CLAIM` configuration property and adding custom claim to your Auth0 users.

### How to use custom claim with Auth0

#### Assign claim to user

1. Go to **Users & Roles**
1. Find your user and edit his details
1. Extend the **user_metadata** by a custom claim, e.g.
    ```json
    {
        "tenant": "e3e0102d-a45b-5cb2-a22e-3a0410deb8d6"
    }
    ```

#### Assign wildcard permission to your service client

1. Go to **Applications**
1. Edit your **Machine to Machine** application
1. Open **Advanced Settings**, switch to **Application Metadata** and add entry:
    - `Key`: `tenant`
    - `Value`: `*`

#### Include custom claim to access token

1. Go to **Rules** and crete new one
1. Copy paste the function below which uses custom claim `https://plgd.dev/tenant`
    ```js
    function addTenantToAccessToken(user, context, callback) {
        var tenantClaim = 'https://plgd.dev/tenant';
        var tenant = (user && user.user_metadata && user.user_metadata.tenant) || (context && context.clientMetadata && context.clientMetadata.tenant) || null;
        if (tenant) {
            context.accessToken[tenantClaim] = tenant;
            context.idToken[tenantClaim] = tenant;
        }
        return callback(null, user, context);
    }
    ```

After the rule is created, Auth0 include into every access tokens custom claim `https://plgd.dev/tenant` used to group users and "their" devices. In case the custom `OWNER_CLAIM` is configured, devices are no more owned by a single user, but in this case, by the **tenant**. Each user who is member of the tenant A will be able to access all the devices of this tenant.

::: warning
If the configuration property `OWNER_CLAIM` is changed, each user is required to have this claim present.
:::
