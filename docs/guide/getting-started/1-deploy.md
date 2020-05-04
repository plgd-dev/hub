# 1. Deploy OCF Cloud
There are multiple options how to start using / testing the gOCF Cloud. If you're just trying to get in touch with this IoT solution, jump right to the **free** [pluggedin.cloud instance](#pluggedin.cloud) and onboard your device. In case you want to **test** the system localy and you have the [Docker ready](https://docs.docker.com/get-docker/), use our [Bundle Docker Image](bundle). Last but not least, in case you're already familiar with the gOCF Cloud and you want to deploy production ready system, follow our [Kubernetes deployment](#kubernetes) using Helm Charts(https://helm.sh/).
If you're already familiar with the OCF Cloud and want to deploy full-blown system, go right to the (#some-markdown-heading)

## Try pluggedin.cloud
Simply visit [pluggedin.cloud](https://pluggedin.cloud) and click `Try`.

## Bundle
Bundle option hosts all gOCF Cloud Services and it's dependencies in a single Docker Image. This solution should be used only for **testing purposes** as the authorization servies is not in full operation.

### Pull the image
```bash
docker pull ocfcloud/bundle:vnext
docker run -d --network=host --name=cloud -t ocfcloud/bundle:vnext
```

### Remarks
- OAuth2.0 Authorization Code is not verified during device onboarding
- Cloud2Cloud is not part of the Bundle deployment

## Kubernetes
comming soon...