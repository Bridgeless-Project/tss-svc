.PHONY: help

update-start-time:
	@echo "Running update_start_time.sh..."
	./scripts/update_start_time.sh

update-keygen:
	@echo "Updating command to run keygen"
	./scripts/update_command.sh "tss-svc service run keygen -o vault\""

update-sign:
	@echo "Updating command to run sign"
	./scripts/update_command.sh "tss-svc service run sign\""

docker-up: update-start-time ## Update session start time and start Docker Compose
	@echo "Starting Docker Compose..."
	docker-compose up

keygen: update-keygen docker-up

sign: update-sign docker-up

rebuild:
	docker-compose down
	docker-compose build
	make docker-up

rebuild-v:
	docker-compose down -v
	docker-compose build
	make docker-up

help: ## Display this help screen
	@grep -E '^[a-zA-Z0-9_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-15s\033[0m %s\n", $$1, $$2}'

.DEFAULT_GOAL := help