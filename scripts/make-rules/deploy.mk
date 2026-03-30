# Copyright 2025 qc <2192629378@qq.com>. All Rights Reserved.
# Use of this source code is governed by a MIT style
# License that can be found in the LICENSE file.

# ==============================================================================
# Makefile helper functions for deploy
#

KUBECTL := kubectl
HELM := helm

# ==================== 可通过命令行/env 完全覆盖的核心变量 ====================
PROJECT_NAME     ?= demo-svc                  # dtk init 时自动替换成 --name 的值
KUBE_NAMESPACE   ?= $(PROJECT_NAME)
KUBE_CONTEXT     ?= ""                        # 留空 = 使用当前 kubectl context
CHART_DIR        ?= $(if $(wildcard $(ROOT_DIR)/deployments/$(PROJECT_NAME)),$(ROOT_DIR)/deployments/$(PROJECT_NAME))

# 支持 dtk deploy --context / --namespace 传入
NAMESPACE ?= $(KUBE_NAMESPACE)
CONTEXT   ?= $(KUBE_CONTEXT)

DEPLOYS ?= $(if $(IMAGES),$(IMAGES),$(BINS))

# ====================== 部署核心 target ======================
.PHONY: deploy.run.all
deploy.run.all:
	@echo "===========> Deploying all components"
	@$(MAKE) deploy.run

.PHONY: deploy.install
deploy.install:
	@echo "===========> Installing chart $(PROJECT_NAME) to $(NAMESPACE)"
	@$(HELM) upgrade --install $(PROJECT_NAME) $(CHART_DIR) \
		--namespace $(NAMESPACE) \
		--create-namespace \
		--set global.version=$(VERSION) \
		--set global.arch=$(ARCH) \
		--wait \
		--timeout 120s \
		$(if $(strip $(CONTEXT)),--kube-context $(CONTEXT)) \
		$(if $(strip $(KUBE_CONFIG)),--kubeconfig $(KUBE_CONFIG))

.PHONY: deploy.full
deploy.full: deploy.build deploy.push deploy.install deploy.run.all

.PHONY: deploy.build
deploy.build:
	@$(foreach img,$(IMAGES), \
		docker build -t $(REGISTRY_PREFIX)/$(img)-$(ARCH):$(VERSION) \
		-f $(ROOT_DIR)/build/docker/$(img)/Dockerfile \
		--build-arg SERVICE_NAME=$(img) $(ROOT_DIR); \
	)

.PHONY: deploy.push
deploy.push:
	@$(foreach img,$(IMAGES), \
		if docker image inspect $(REGISTRY_PREFIX)/$(img)-$(ARCH):$(VERSION) > /dev/null 2>&1; then \
			echo "===========> Pushing $(img):$(VERSION)"; \
			docker push $(REGISTRY_PREFIX)/$(img)-$(ARCH):$(VERSION) \
			|| { echo "✘ docker push failed"; exit 1; }; \
		else \
			echo "===========> Skipping push $(img):$(VERSION) (not built locally, already on registry)"; \
		fi; \
	)
	
.PHONY: deploy.run
deploy.run: $(addprefix deploy.run., $(DEPLOYS))

.PHONY: deploy.run.%
deploy.run.%:
	@echo "===========> Deploying $* $(VERSION) on $(ARCH)"
	@$(KUBECTL) \
		$(if $(strip $(KUBE_CONFIG)),--kubeconfig $(KUBE_CONFIG)) \
		$(if $(strip $(CONTEXT)),--context $(CONTEXT)) \
		--namespace $(NAMESPACE) \
		set image deployment/$* $*=$(REGISTRY_PREFIX)/$*-$(ARCH):$(VERSION)
	@$(KUBECTL) \
		$(if $(strip $(KUBE_CONFIG)),--kubeconfig $(KUBE_CONFIG)) \
		$(if $(strip $(CONTEXT)),--context $(CONTEXT)) \
		--namespace $(NAMESPACE) \
		rollout status deployment/$* --timeout=300s