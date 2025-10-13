update-start-time:
	@echo "Running update_start_time.sh..."
	./scripts/update_start_time.sh

update-keygen:
	@echo "Updating command to run keygen"
	./scripts/update_command.sh "tss-svc service run keygen -o vault\""

update-sign:
	@echo "Updating command to run sign"
	./scripts/update_command.sh "tss-svc service run sign\""

docker-up: update-start-time
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