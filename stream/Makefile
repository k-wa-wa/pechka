# Makefile for kind development environment
# Use rootful Docker socket to allow BPF operations inside kind nodes.
# The system Docker daemon (rootful) runs on /run/docker.sock.
export DOCKER_HOST = unix:///run/docker.sock

CLUSTER_NAME = pechka-cluster
K8S_DIR = k8s/overlays/dev
KIND_CONFIG = k8s/kind-config.yaml
CILIUM_VERSION = v1.16.2
CILIUM_HELM_VERSION = 1.16.2
CILIUM_ENVOY_TAG = v1.29.9-1726784081-a90146d13b4cd7d168d573396ccf2b3db5a3b047

.PHONY: up
up: cluster prefetch-cilium setup-cilium build load deploy

.PHONY: cluster
cluster:
	@if ! kind get clusters | grep -q $(CLUSTER_NAME); then \
		kind create cluster --name $(CLUSTER_NAME) --config $(KIND_CONFIG); \
	fi

# Pre-pull Cilium images on the host (host DNS works) then load directly into
# containerd in the kind node, bypassing the broken in-cluster DNS.
# Note: docker save | ctr import automatically creates digest references that
# match what the Helm chart specifies, so no extra tagging is needed.
.PHONY: prefetch-cilium
prefetch-cilium:
	@echo "Pulling Cilium images on host..."
	docker pull --platform linux/amd64 quay.io/cilium/cilium:$(CILIUM_VERSION)
	docker pull --platform linux/amd64 quay.io/cilium/operator-generic:$(CILIUM_VERSION)
	docker pull --platform linux/amd64 quay.io/cilium/cilium-envoy:$(CILIUM_ENVOY_TAG)
	@echo "Loading Cilium images into kind node..."
	docker save quay.io/cilium/cilium:$(CILIUM_VERSION) \
		| docker exec -i $(CLUSTER_NAME)-control-plane ctr --namespace=k8s.io images import -
	docker save quay.io/cilium/operator-generic:$(CILIUM_VERSION) \
		| docker exec -i $(CLUSTER_NAME)-control-plane ctr --namespace=k8s.io images import -
	docker save quay.io/cilium/cilium-envoy:$(CILIUM_ENVOY_TAG) \
		| docker exec -i $(CLUSTER_NAME)-control-plane ctr --namespace=k8s.io images import -

.PHONY: setup-cilium
setup-cilium:
	@echo "Installing Gateway API CRDs..."
	kubectl apply -f https://github.com/kubernetes-sigs/gateway-api/releases/download/v1.1.0/experimental-install.yaml
	@echo "Installing Cilium via Helm..."
	helm repo add cilium https://helm.cilium.io/ 2>/dev/null || true
	helm repo update
	helm upgrade --install cilium cilium/cilium --namespace kube-system \
		--version $(CILIUM_HELM_VERSION) \
		--set gatewayAPI.enabled=true \
		--set gatewayAPI.gatewayClass.enabled=true \
		--set kubeProxyReplacement=false \
		--set nodePort.enabled=true \
		--set operator.replicas=1
	@echo "Waiting for Cilium pods to appear..."
	@count=0; until kubectl get pods -l app.kubernetes.io/name=cilium-agent -n kube-system -o name 2>/dev/null | grep -q "pod/"; do \
		count=$$((count+1)); \
		if [ $$count -ge 90 ]; then echo "ERROR: Timeout (180s) - Cilium pods did not appear. Check: kubectl get pods -n kube-system"; exit 1; fi; \
		sleep 2; \
	done
	kubectl wait --for=condition=ready pod -l app.kubernetes.io/name=cilium-agent -n kube-system --timeout=300s

.PHONY: build
build: build-app build-ffmpeg build-analyzer build-frontend build-dev-proxy

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

.PHONY: build-dev-proxy
build-dev-proxy:
	docker build -t dev-proxy:latest tests/dev-proxy


.PHONY: load
load: load-app load-ffmpeg load-analyzer load-frontend load-dev-proxy

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

.PHONY: load-dev-proxy
load-dev-proxy:
	kind load docker-image dev-proxy:latest --name $(CLUSTER_NAME)


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
	kubectl rollout restart deployment/auth-service deployment/catalog-service

.PHONY: reload-frontend
reload-frontend: build-frontend load-frontend
	kubectl rollout restart deployment/frontend

.PHONY: reload-analyzer
reload-analyzer: build-analyzer load-analyzer
	kubectl rollout restart deployment/thumbnail-analyzer

.PHONY: reload-dev-proxy
reload-dev-proxy: build-dev-proxy load-dev-proxy
	kubectl rollout restart deployment/dev-proxy

.PHONY: reload-batch
reload-batch: build-app build-ffmpeg load-app load-ffmpeg
	kubectl delete job --all
	kubectl apply -k $(K8S_DIR)

.PHONY: clean
clean: down
	docker image rm pechka-app:latest pechka-ffmpeg:latest pechka-analyzer:latest pechka-frontend:latest dev-proxy:latest || true
