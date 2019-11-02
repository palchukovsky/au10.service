TAG = latest
ORGANIZATION = au10
PRODUCT = service
REPO = bitbucket.org/au10/service

PROTOC_VERSION = 3.11.1

export GO111MODULE = on
GO_GET_CMD = go get -v

IMAGE_TAG_ACCESSPOINT_PROXY = $(ORGANIZATION)/$(PRODUCT).accesspoint-proxy:$(TAG)

THIS_FILE := $(lastword $(MAKEFILE_LIST))
COMMA := ,

.DEFAULT_GOAL := help


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


.PHONY: help install stub mock build build-accesspoint-proxy


help: ## Show this help.
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "};	{printf "\033[36m%-16s\033[0m %s\n", $$1, $$2}'


install: ## Install package and all dependencies.
	$(GO_GET_CMD) github.com/palchukovsky/protoc-install
	protoc-install -type cli -ver $(PROTOC_VERSION) -out ./build/bin
	$(GO_GET_CMD) github.com/golang/protobuf/protoc-gen-go
	$(call make_target,stub)

	$(GO_GET_CMD) ./...

	$(GO_GET_CMD) github.com/stretchr/testify/assert
	$(GO_GET_CMD) github.com/golang/mock/gomock
	$(GO_GET_CMD) github.com/golang/mock/mockgen
	$(call make_target,mock)


stub: ## Generate stubs.
	./build/bin/protoc -I ./accesspoint/ ./accesspoint/accesspoint.proto  --go_out=plugins=grpc:./accesspoint/lib/


mock: ## Generate mock interfaces for unit-tests.
	$(call gen_mock,au10/factory,Factory)
	$(call gen_mock,au10/service,Service)
	$(call gen_mock,au10/streamreader,StreamReader)
	$(call gen_mock_aux,au10/streamwriter,StreamWriter)
	$(call gen_mock_aux,au10/log,Log LogSubscription,$(REPO)/au10=au10/subscription.go$(COMMA)$(REPO)/au10=au10/member.go)
	$(call gen_mock,au10/member,Memeber)
	$(call gen_mock,au10/group,Rights Membership)
	$(call gen_mock_aux,au10/user,User,$(REPO)/au10=au10/member.go)
	$(call gen_mock,au10/users,Users)
	
	$(call gen_mock_ext,github.com/Shopify/sarama,AsyncProducer$(COMMA)ConsumerGroup$(COMMA)ConsumerGroupSession$(COMMA)ConsumerGroupClaim,sarama)


build: ## Build all docker images from actual local sources.
	$(call make_target,build-accesspoint-proxy)

build-accesspoint-proxy: ## Build access point proxy node docker image from actual local sources.
	docker build \
		--rm \
		--file "./accesspoint/proxy/Dockerfile" \
		--tag $(IMAGE_TAG_ACCESSPOINT_PROXY) \
		./
