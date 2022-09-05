registry=docker.io
repostitory=bluebrown/dqlite-app

dir_certs=assets/certs/
dir_scripts=assets/scripts/

namespace=sandbox

build:
	docker build -t dqlite-app:local --file assets/docker/Dockerfile .

push: build
	docker tag dqlite-app:local $(registry)/$(repostitory):latest
	docker push $(registry)/$(repostitory):latest

cert:
	mkdir -p $(dir_certs)
	$(dir_scripts)/gen-cert.sh

deploy: cert
	kubectl create namespace $(namespace) || true
	kubectl create secret tls dqlite-app-cluster-cert --cert=$(dir_certs)/tls.crt --key=$(dir_certs)/tls.key --save-config --dry-run=client -o yaml \
		| kubectl apply -n $(namespace)  -f -
	kubectl apply -n $(namespace) -f kube.yaml

teardown:
	kubectl delete -n $(namespace) -f kube.yaml --ignore-not-found
	kubectl delete secret/dqlite-app-cluster-cert pvc/data-dqlite-app-0 pvc/data-dqlite-app-1 pvc/data-dqlite-app-2 -n $(namespace) --ignore-not-found
	kubectl delete namespace $(namespace) --ignore-not-found

compose: build
	@docker compose up