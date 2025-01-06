protogen-p2p:
	cd proto/p2p && buf generate

preparams:
	VAULT_PATH=http://localhost:8200 VAULT_TOKEN=root MOUNT_PATH=secret go run main.go helpers generate preparams -o vault --config=configs/config.yaml