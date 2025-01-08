VAULT_PATH=http://localhost:8200
VAULT_TOKEN=root
MOUNT_PATH=secret

protogen-p2p:
	cd proto/p2p && buf generate

account:
	VAULT_PATH=${VAULT_PATH} VAULT_TOKEN=${VAULT_TOKEN} MOUNT_PATH=${MOUNT_PATH} go run main.go helpers generate cosmos-account -o vault --config=./internal/config/config.yaml

preparams-f:
	go run main.go helpers generate preparams -o file --path=./internal/config/preparams.json

preparams:
	VAULT_PATH=${VAULT_PATH} VAULT_TOKEN=${VAULT_TOKEN} MOUNT_PATH=${MOUNT_PATH} go run main.go helpers generate preparams -o vault --config=./internal/config/config.yaml