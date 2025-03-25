#!/bin/bash

N=3
DAYS_VALID=365

for i in $(seq 1 $N); do
    PARTY_KEY="party${i}.key"
    PARTY_CERT="party${i}.crt"
    CONFIG_FILE="party${i}.cnf"

    cat > $CONFIG_FILE <<EOF
[ req ]
default_bits       = 2048
distinguished_name = req_distinguished_name
req_extensions     = v3_ext
x509_extensions    = v3_ext

[ req_distinguished_name ]
commonName = party-${i}
commonName_default = party-${i}

[ v3_ext ]
subjectAltName = @alt_names
basicConstraints = CA:FALSE
keyUsage = digitalSignature, keyEncipherment
extendedKeyUsage = serverAuth, clientAuth

[ alt_names ]
DNS.1 = tss-${i}
EOF

    echo "🔹 Generating key and self-signed certificate for Party $i..."
    openssl genpkey -algorithm RSA -out ./../configs/certs/$PARTY_KEY
    openssl req -new -key ./../configs/certs/$PARTY_KEY -out party${i}.csr -config $CONFIG_FILE
    openssl x509 -req -in party${i}.csr -signkey ./../configs/certs/$PARTY_KEY -out ./../configs/certs/$PARTY_CERT -days $DAYS_VALID -extensions v3_ext -extfile $CONFIG_FILE

    # Cleanup CSR and config file
    rm -f party${i}.csr $CONFIG_FILE
done

echo "✅ Self-signed certificates generated successfully!"
