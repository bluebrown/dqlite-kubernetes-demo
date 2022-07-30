name: dqlite

x-app:
  &app
  image: dqlite-app:local
  volumes:
    - ../../.certs:/app/certs:ro
    - ../../.data:/app/data:rw,delegated
    - ../../bin:/app/bin:ro
  healthcheck:
    test: /app/bin/httpcheck http://localhost:8080/ready

x-env:
  &env
  K8S_SERVICE_NAME: dqlite-app-headless
  K8S_NAMESPACE: sandbox
  K8S_CLUSTER_DOMAIN: cluster.local

services:
  ingress-controller:
    image: haproxy:lts-alpine
    volumes: [ ./haproxy.cfg:/usr/local/etc/haproxy/haproxy.cfg:ro ]
    ports: [ 8080:8080 ]
{{ range iter 3 }}
  app-{{ . }}:
    <<: *app
    container_name: dqlite-app-{{ . }}
    environment:
      <<: *env
      K8S_POD_NAME: dqlite-app-{{ . }}
      DATA_DIR: /app/data/{{ . }}
    networks:
      default:
        aliases:
          - dqlite-app-{{ . }}.dqlite-app-headless.sandbox.svc.cluster.local
          - dqlite-app-headless
{{ end -}}
