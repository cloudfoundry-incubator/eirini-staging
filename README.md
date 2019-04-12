# eirini-staging

`eirini-staging` is a process for building Droplets on Kubernetes. Three conainter images are created:
- `eirini/recipe-downloader`: Downloads app-bits and buildpacks from the `bits-service` 
- `eirini/recipe-executor`: Executes the `buildpackapplifecyle` to build a Droplet
- `eirini/recipe-uploader`: Uploads the Droplet to the `bits-service`

