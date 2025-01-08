#!/usr/bin/env sh

function check_vault_init {
  RESPONSE=$(curl --insecure --silent --header "X-Vault-Token: $VAULT_TOKEN" http://$VAULT_ADDR/v1/tss1/data/keygen_preparams)

  if echo "$RESPONSE" | grep -q '"data"'; then
    echo "Secrets exists."
    return 0
  else
    echo "Setting secrets..."
    return 1
  fi
}

function wait_vault {
    echo "Waiting for Vault..."

    while true
    do
        RESPONSE=$(curl --insecure --silent -H "X-Vault-Token: $VAULT_TOKEN" $VAULT_ADDR/v1/sys/health)

        if echo "$RESPONSE" | grep -q '"initialized":true'; then
            echo "Vault is initialized!"
            break
        fi

        echo "Vault is Initializing..."
        sleep 2
    done
}

function init_tss1 {
  curl \
    --header "X-Vault-Token: $VAULT_TOKEN" \
    --header "Content-Type: application/json" \
    --request POST \
    --data '{"data": {"token": "5795087518:AAGHFFEFcz8D_7bEizY-8khUlbMBGApeOAg"}}' \
    $VAULT_ADDR/v1/secret/data/telegram
}

echo "Initializing Vault..."

wait_vault

if check_vault_init; then
  echo "Initialization complete. Exiting."
  exit 0
fi

init_tss1
