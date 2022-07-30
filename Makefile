registry=docker.io
repostitory=bluebrown/dqlite-app
certs_dir=.certs
docker_dir=assets/docker
kube_dir=assets/kube
bin_dir=bin
k8s_namespace=sandbox

all: httpcheck cert image up

tpl:
	$(bin_dir)/tpl -f $(docker_dir)/compose.yaml.tpl > $(docker_dir)/compose.yaml

up:
	docker compose --file $(docker_dir)/compose.yaml up --remove-orphans

image:
	docker build -t dqlite-app:local --build-arg uid=$(shell id -u) --file $(docker_dir)/Dockerfile .

push:
	docker build -t $(registry)/$(repostitory) --file $(docker_dir)/Dockerfile .
	docker push $(registry)/$(repostitory)

cert:
	mkdir -p $(certs_dir)
	openssl req -x509 -newkey rsa:4096 -sha256 -days 3650 \
    -nodes -keyout "$(certs_dir)/tls.key" -out "$(certs_dir)/tls.crt" -subj "/CN=dqlite-cluster" \
    -addext "subjectAltName=DNS:dqlite-cluster"

httpcheck:
	go build -o $(bin_dir)/httpcheck ./cmd/httpcheck

wipe:
	rm -rf .data/*

test:
	curl localhost:8080/entries -X POST -d '{"value": "hello"}'
	curl localhost:8080/entries -X POST -d '{"value": "world"}'
	curl localhost:8080/entries

kube.apply: cert
	kubectl create namespace $(k8s_namespace) || true
	kubectl create secret tls dqlite-app-cluster-cert --cert=$(certs_dir)/tls.crt --key=$(certs_dir)/tls.key --save-config --dry-run=client -o yaml \
		| kubectl apply -n $(k8s_namespace)  -f -
	kubectl apply -n $(k8s_namespace) -f $(kube_dir)/

kube.delete:
	kubectl delete -n $(k8s_namespace) -f $(kube_dir)/
	kubectl delete secret/dqlite-app-cluster-cert pvc/data-dqlite-app-0 pvc/data-dqlite-app-1 pvc/data-dqlite-app-2 -n $(k8s_namespace)
	kubectl delete namespace $(k8s_namespace)

deps:
	mkdir -p $(bin_dir)
	curl -fsSL https://github.com/bluebrown/go-template-cli/releases/latest/download/tpl-linux-amd64 >$(bin_dir)/tpl
	chmod 755 $(bin_dir)/tpl
