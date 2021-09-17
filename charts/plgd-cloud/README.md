
## plgd-cloud helm chart

### Install

- (https://cert-manager.io/)[https://artifacthub.io/packages/helm/cert-manager/cert-manager] - install cert-manager helm chart

**Required variables:**

```yaml
resourcedirectory:
  publicConfiguration:
    tokenURL: 
    authorizationURL: 
global:
  domain: 
  cloudId: 
  authority: 
  audience: 
  authorizationServer:
    oauth:
      clientID: 
      clientSecret: 
      scopes: []
      tokenURL: 
  device:
    oauth:
      clientID: 
      clientSecret:
      scopes: []
      tokenURL: 
      redirectURL: 

```