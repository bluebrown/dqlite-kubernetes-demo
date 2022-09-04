# Dqlite Kubernetes Demo

The command will create a namespace in the current cluster with name `sandbox` and deploy the application there. The image will be pulled from [dockerhub](https://hub.docker.com/r/bluebrown/dqlite-app). You can customize the makefile and build the image yourself to push it to your own registry and deploy to a namespace of your choice.

```bash
# deploy to kubernetes
make deploy 
# clean up
make teardown
```
