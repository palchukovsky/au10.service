TAG = dev-latest
ORGANIZATION = au10
PRODUCT = service
CODE_REPO = bitbucket.org/au10/service
IMAGES_REPO = registry.gitlab.com/$(ORGANIZATION)/

GO_VER = 1.13
PROTOC_VERSION = 3.11.1
NODE_OS_NAME = alpine
NODE_OS_TAG = 3.10
ENVOY_TAG = v1.12.2

GO_GET_CMD = go get -v

IMAGE_TAG_PROTOC = $(IMAGES_REPO)protoc:$(PROTOC_VERSION)
IMAGE_TAG_GOLANG = $(IMAGES_REPO)$(PRODUCT).golang:${GO_VER}-${NODE_OS_NAME}${NODE_OS_TAG}
IMAGE_TAG_ENVOY = $(IMAGES_REPO)envoy:${ENVOY_TAG}
IMAGE_TAG_ACCESSPOINT = $(IMAGES_REPO)$(PRODUCT).accesspoint:$(TAG)
IMAGE_TAG_ACCESSPOINT_PROXY = $(IMAGES_REPO)$(PRODUCT).accesspoint-proxy:$(TAG)

THIS_FILE := $(lastword $(MAKEFILE_LIST))
COMMA := ,

.DEFAULT_GOAL := help


define build_docker_builder_image
	$(eval BUILDER_SOURCE_TAG = ${GO_VER}-${NODE_OS_NAME}${NODE_OS_TAG})
	$(eval BUILDER_TAG = $(IMAGES_REPO)$(PRODUCT).builder:$(BUILDER_SOURCE_TAG))
	docker build \
		--network none \
		--build-arg PROTOC=$(IMAGE_TAG_PROTOC) \
		--build-arg GOLANG=$(IMAGE_TAG_GOLANG) \
		--file "$(CURDIR)/build/builder/builder.Dockerfile" \
		--tag $(BUILDER_TAG) \
		./
endef
define build_docker_cmd_image
	$(if $(BUILDER_TAG),, $(call build_docker_builder_image))
	docker build \
		--build-arg NODE_OS_NAME=$(NODE_OS_NAME) \
		--build-arg NODE_OS_TAG=$(NODE_OS_TAG) \
		--build-arg BUILDER=$(BUILDER_TAG) \
		--file "$(CURDIR)/$(1)/Dockerfile" \
		--tag $(2) \
		./
endef

define gen_mock
	mockgen -source=$(1).go -destination=./mock/$(1).go $(2)
endef
define gen_mock_aux
	mockgen -source=$(1).go -destination=./mock/$(1).go -aux_files=$(3) $(2)
endef
define gen_mock_ext
	-cd ./mock/ && mkdir $(3)
	mockgen $(1) $(2) > ./mock/$(3)/$(3).go 
endef

define make_target
	$(MAKE) -f ./$(THIS_FILE) $(1)
endef

define echo_success
	@echo ================================================================================
	@echo :
	@echo : SUCCESS: $(@)
	@echo :
	@echo ================================================================================
endef


.PHONY: help \
	install install-protoc install-mock install-mock-deps \
	stub \
	mock \
	build\
		build-full \
		build-builder build-builder-protoc build-builder-golang build-builder-envoy \
		build-accesspoint build-accesspoint-proxy


help: ## Show this help.
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "};	{printf "\033[36m%-16s\033[0m %s\n", $$1, $$2}'


install: ## Install package and all dependencies.
	$(call make_target,install-protoc)
	$(call make_target,stub)

	$(GO_GET_CMD) ./...

	$(call make_target,install-mock)

	go mod tidy

	@$(call echo_success)

install-protoc: ## Install proto compilator.
	$(GO_GET_CMD) github.com/palchukovsky/protoc-install
	protoc-install -type cli -ver $(PROTOC_VERSION) -out ./build/bin
	$(GO_GET_CMD) github.com/golang/protobuf/protoc-gen-go
	@$(call echo_success)

install-mock: ## Install mock compilator and generate mock.
	$(call make_target,install-mock-deps)
	$(call make_target,mock)
	@$(call echo_success)
install-mock-deps: ## Install mock compilator components.
	$(GO_GET_CMD) github.com/stretchr/testify/assert
	$(GO_GET_CMD) github.com/golang/mock/gomock
	$(GO_GET_CMD) github.com/golang/mock/mockgen
	@$(call echo_success)

stub: ## Generate stubs.
	-cd ./accesspoint/ && mkdir proto
	./build/bin/protoc -I ./accesspoint/ ./accesspoint/accesspoint.proto  --go_out=plugins=grpc:./accesspoint/proto/
	@$(call echo_success)


mock: ## Generate mock interfaces for unit-tests.
# "go list ... " in the next run required as a workaround for error - first start mockgen fails with errot at "go list ...":
	-go list -e -compiled=true -test=true ./*

	$(call gen_mock,au10/factory,Factory)
	$(call gen_mock,au10/service,Service)
	$(call gen_mock,au10/streamreader,StreamReader)
	$(call gen_mock,au10/streamwriter,StreamWriter)
	$(call gen_mock_aux,au10/log,Log LogSubscription,$(CODE_REPO)/au10=au10/subscription.go$(COMMA)$(CODE_REPO)/au10=au10/member.go)
	$(call gen_mock,au10/member,Memeber)
	$(call gen_mock,au10/group,Rights Membership)
	$(call gen_mock_aux,au10/user,User,$(CODE_REPO)/au10=au10/member.go)
	$(call gen_mock,au10/users,Users)
	$(call gen_mock_aux,au10/post,Post,$(CODE_REPO)/au10=au10/member.go)
	$(call gen_mock_aux,au10/posts,Posts,$(CODE_REPO)/au10=au10/member.go)
	
	$(call gen_mock_ext,github.com/Shopify/sarama,AsyncProducer$(COMMA)ConsumerGroup$(COMMA)ConsumerGroupSession$(COMMA)ConsumerGroupClaim,sarama)

	@$(call echo_success)


build-full: ## Build all docker images from actual local sources.
	$(call make_target,build-builder)
	$(call make_target,build)
	@$(call echo_success)

build: ## Build docker images with all project services from actual local sources.
	$(call make_target,build-accesspoint)
	$(call make_target,build-accesspoint-proxy)
	@$(call echo_success)

build-accesspoint: ## Build access point node docker image from actual local sources.
	$(call build_docker_cmd_image,accesspoint,$(IMAGE_TAG_ACCESSPOINT))
	@$(call echo_success)
build-accesspoint-proxy: ## Build access point proxy node docker image from actual local sources.
	docker build --build-arg ENVOY=$(IMAGE_TAG_ENVOY) --file "./accesspoint/proxy/Dockerfile" --tag $(IMAGE_TAG_ACCESSPOINT_PROXY) ./
	@$(call echo_success)

build-builder: ## Build all docker images for builder.
	$(call make_target,build-builder-protoc)
	$(call make_target,build-builder-golang)
	$(call make_target,build-builder-envoy)
	@$(call echo_success)
build-builder-protoc: ## Build docker protoc-image.
	docker build --file "./build/builder/protoc.Dockerfile" --tag $(IMAGE_TAG_PROTOC)  ./
	@$(call echo_success)
build-builder-golang: ## Build docker golang base node image.
	docker build --file "./build/builder/golang.Dockerfile" --build-arg GOLANG_TAG=${GO_VER}-${NODE_OS_NAME}${NODE_OS_TAG} --tag $(IMAGE_TAG_GOLANG)  ./
	@$(call echo_success)
build-builder-envoy: ## Build docker envoy base image.
	docker build --file "./accesspoint/proxy/envoy.Dockerfile" --build-arg TAG=${ENVOY_TAG} --tag $(IMAGE_TAG_ENVOY)  ./
	@$(call echo_success)

release: ## Push all images on the hub.
	docker push $(IMAGE_TAG_ENVOY)
	docker push $(IMAGE_TAG_ACCESSPOINT)
	docker push $(IMAGE_TAG_ACCESSPOINT_PROXY)
	docker push $(IMAGE_TAG_PROTOC)
	docker push $(IMAGE_TAG_GOLANG)
	@$(call echo_success)

