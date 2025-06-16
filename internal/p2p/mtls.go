package p2p

import (
	"crypto/x509"
	"encoding/pem"
	"fmt"

	"github.com/Bridgeless-Project/tss-svc/internal/core"
	"github.com/pkg/errors"
)

type AuthorizedParties map[string]core.Address

func (p *AuthorizedParties) Add(cert *x509.Certificate, addr core.Address) error {
	pubkeyRaw, err := x509.MarshalPKIXPublicKey(cert.PublicKey)
	if err != nil {
		return errors.Wrap(err, "failed to marshal certificate public key")
	}

	(*p)[string(pubkeyRaw)] = addr

	return nil
}

func (p *AuthorizedParties) Get(cert *x509.Certificate) *core.Address {
	pubkeyRaw, err := x509.MarshalPKIXPublicKey(cert.PublicKey)
	if err != nil {
		return nil
	}

	addr, exists := (*p)[string(pubkeyRaw)]
	if !exists {
		return nil
	}

	return &addr
}

func ConfigurePartiesCertPool(parties []Party) (*x509.CertPool, *AuthorizedParties, error) {
	certPool := x509.NewCertPool()
	authorizedParties := make(AuthorizedParties, len(parties))
	for _, party := range parties {
		partyCert := party.PEMCert()
		if !certPool.AppendCertsFromPEM(partyCert) {
			return nil, nil, errors.New(fmt.Sprintf("invalid PEM certificate for party %q", party.CoreAddress))
		}

		cert, err := parsePEMCertificate(partyCert)
		if err != nil {
			return nil, nil, errors.Wrap(err, fmt.Sprintf("failed to parse certificate for party %q", party.CoreAddress))
		}

		if err := authorizedParties.Add(cert, party.CoreAddress); err != nil {
			return nil, nil, errors.Wrap(err, fmt.Sprintf("failed to add party %q to authorized parties", party.CoreAddress))
		}
	}

	return certPool, &authorizedParties, nil
}

func parsePEMCertificate(pemCert []byte) (*x509.Certificate, error) {
	block, _ := pem.Decode(pemCert)
	if block == nil {
		return nil, errors.New("invalid PEM block")
	}

	// Parse the x509 certificate
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse certificate")
	}

	return cert, nil
}
