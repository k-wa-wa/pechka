# Makefile for kind development environment

CLUSTER_NAME = pechka-cluster
K8S_DIR = k8s/overlays/dev
KIND_CONFIG = k8s/kind-config.yaml

.PHONY: up
up: cluster build load deploy

.PHONY: cluster
cluster:
	@if ! kind get clusters | grep -q $(CLUSTER_NAME); then \
		kind create cluster --name $(CLUSTER_NAME) --config $(KIND_CONFIG); \
	fi

.PHONY: build
build: build-app build-ffmpeg build-analyzer build-frontend

.PHONY: build-app
build-app:
	docker build -t pechka-app:latest -f api/Dockerfile.api api

.PHONY: build-ffmpeg
build-ffmpeg:
	docker build -t pechka-ffmpeg:latest -f api/Dockerfile.ffmpeg api

.PHONY: build-analyzer
build-analyzer:
	docker build -t pechka-analyzer:latest batch/thumbnail-analyzer

.PHONY: build-frontend
build-frontend:
	docker build -t pechka-frontend:latest frontend


.PHONY: load
load: load-app load-ffmpeg load-analyzer load-frontend

.PHONY: load-app
load-app:
	kind load docker-image pechka-app:latest --name $(CLUSTER_NAME)

.PHONY: load-ffmpeg
load-ffmpeg:
	kind load docker-image pechka-ffmpeg:latest --name $(CLUSTER_NAME)

.PHONY: load-analyzer
load-analyzer:
	kind load docker-image pechka-analyzer:latest --name $(CLUSTER_NAME)

.PHONY: load-frontend
load-frontend:
	kind load docker-image pechka-frontend:latest --name $(CLUSTER_NAME)


.PHONY: deploy
deploy:
	kubectl apply -k $(K8S_DIR)

.PHONY: down
down:
	kind delete cluster --name $(CLUSTER_NAME)

.PHONY: logs
logs:
	kubectl get pods
	@echo "Example: kubectl logs -f deployment/metadata-service"

.PHONY: reload
reload: build load
	kubectl rollout restart deployment
	kubectl delete job --all
	kubectl apply -k $(K8S_DIR)

.PHONY: reload-app
reload-app: build-app load-app
	kubectl rollout restart deployment/metadata-service deployment/auth-service deployment/catalog-service

.PHONY: reload-frontend
reload-frontend: build-frontend load-frontend
	kubectl rollout restart deployment/frontend

.PHONY: reload-analyzer
reload-analyzer: build-analyzer load-analyzer
	kubectl rollout restart deployment/thumbnail-analyzer

.PHONY: reload-batch
reload-batch: build-app build-ffmpeg load-app load-ffmpeg
	kubectl delete job --all
	kubectl apply -k $(K8S_DIR)

.PHONY: clean
clean: down
	docker image rm pechka-app:latest pechka-ffmpeg:latest pechka-analyzer:latest pechka-frontend:latest || true
