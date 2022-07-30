# dqlite kubernetes demo

This is a demo application for dqlite wich is intended to be deployed in kubernetes. The local setup with compose reflects this intention by using kubernetes like network aliases.

## Local Development

The make default command starts the app locally. You can [review the compose template](./assets/docker/) to learn more.

```bash
make deps
make
make test
```

## Kubernetes

the app can be deployed as kubernetes statefulset. You can [review the manifests](./assets/kube/) to learn more.

```bash
make kube.apply
```
