.PHONY: help

protogen: ## Generate protobuf types using buf
	cd proto && \
	buf generate deposit --template=./templates/deposit.yaml --config=buf.yaml && \
	buf generate p2p --template=./templates/p2p.yaml --config=buf.yaml && \
	buf generate api --template=./templates/api.yaml --config=buf.yaml

account: ## Generate a new cosmos account
	go run main.go helpers generate cosmos-account

preparams-f: ## Generate preparams file
	go run main.go helpers generate preparams -o file --path=./internal/config/preparams.json


help: ## Display this help screen
	@grep -E '^[a-zA-Z0-9_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-15s\033[0m %s\n", $$1, $$2}'


.DEFAULT_GOAL := help