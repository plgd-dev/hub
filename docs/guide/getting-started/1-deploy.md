# 1. Deploy plgd Cloud
There are multiple options how to start using / testing the plgd Cloud. If you're just trying to get in touch with this Cloud Native IoT framework, jump right to the [try.plgd.cloud](#Try plgd.cloud) instance and onboard your device right away. In case you want to **get in touch** with the system localy and you have the [Docker installed](https://docs.docker.com/get-docker/), use our [plgd cloud #Bundle](#bundle).
::: tip
Helm chart for deployment in k8s is in progress. Contributions welcome!
:::

## Try plgd.cloud
The plgd team operates their own instance of the plgd cloud for free. This cloud instance is integrated with the plgd mobile application available for both iOS and Android based devices. Together with our IoTivity-Lite sample you're able to onboard and work with your device remotely in couple of seconds. To start right away, follow [try.plgd.cloud](https://try.plgd.cloud). More information about the mobile application is available in the [Onboard]() Getting Started section.

## Bundle
Bundle deployment hosts core plgd Cloud Services with mocked OAuth Server in a single Docker image. All services which hosts the gRPC or HTTP API are proxied through the NGINX with configurable `NGINX_PORT` and `FQDN`. Mobile application documented in the [Onboard]() Getting Started section works also with the Bundle.

::: danger
Bundle version of plgd services should be used only for simple testing and development purposes. Performance evaluations, production environment or other sensitive deployments should deploy plgd services using the plgd HELM chart.
:::
### Run on localhost
To deploy and access plgd cloud on your local PC using bundle, run single command:
`docker run -d --name plgd -p 443:443 -p 5683:5683 -p 5684:5684 plgd/bundle:v2next`

After couple of seconds your plgd cloud should be available in your browser at [https://localhost](https://localhost).
Note that bundle issues it's own self-signed certificate which needs to be accepted in the browser.

### Authorization
The plgd cloud doesn't work without OAuth Server. To not require developers not interested in sharing bundle instance with other users, simple mocked OAuth Server is included in the bundle. Authentication to the plgd is therefore not required and test user is automatically logged in. Same applies also to device connections; in case you're using the bundle, devices connecting to the CoAP Gateway can use random/static onboarding code as it's not verified. Onboarding of devices is therefore much simpler.

### Using external OAuth Server with bundle
As the bundle just start core plgd services as processes in one container, user has still possibility to configure most of the services parameters. **For testing purposes**, the external OAuth Server (e.g. https://auth0.com) can be configured.
To skip internal mocked OAuth Server and switch to your external one, configure following environment variables:
```
    OAUTH_AUDIENCE: https://api.example.com
    OAUTH_ENDPOINT_AUTH_URL: https://auth.example.com/authorize
    OAUTH_ENDPOINT_TOKEN_URL: https://auth.example.com/oauth/token
    OAUTH_ENDPOINT: auth.example.com
    JWKS_URL: https://auth.example.com/.well-known/jwks.json
    OAUTH_CLIENT_ID: ij12OJj2J23K8KJs
    SERVICE_OAUTH_CLIENT_ID: 412dsFf53Sj6$
    SERVICE_OAUTH_CLIENT_SECRET: 235Jgdf65jsd4Shls
```

### Troubleshooting
- By default the plgd cloud bundle hosts the NGINX proxy on port `443`. This port might be already occupied by other process, e.g. Skype. Default port can be changed by environment variable `-e NGINX_PORT=8443`. Please be aware that the port needs to be exposed from the container -> `-p 443:443`needs to be changed to match a new port, e.g. `-p 8443:8443`.

### Pull the image
```bash
docker pull plgd/bundle:vnext
docker run -d --name=plgd --network=host -t plgd/bundle:vnext
```

### Remarks
- OAuth2.0 Authorization Code is not verified during device onboarding
- Cloud2Cloud is not part of the Bundle deployment

## Kubernetes
comming soon...