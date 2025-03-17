package net_service

import (
	"crypto"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"fmt"
	"math/big"
	"time"

	"github.com/btcsuite/btcutil/base58"
	"golang.org/x/crypto/sha3"
)

type RootSecrets struct {
	root_priv_key  crypto.PrivateKey
	root_self_cert []byte
	root_id_hash   string
}

func GenerateRootIdentity() (*RootSecrets, error) {
	ed25519_public_key, ed25519_private_key, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, err
	}

	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128) // 2^128
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		return nil, err
	}
	id_hash, err := IdentityHashFromRootPublicKey(ed25519_public_key)
	if err != nil {
		return nil, err
	}
	template := x509.Certificate{
		Issuer: pkix.Name{
			CommonName: id_hash,
		},
		Subject: pkix.Name{
			CommonName: id_hash,
		},
		NotBefore:             time.Now().Add(time.Duration(-1) * time.Second), //1-sec backdate, for badly synced peers.
		SerialNumber:          serialNumber,
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		IsCA:                  true,
		BasicConstraintsValid: true,
	}
	derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, ed25519_public_key, ed25519_private_key)
	if err != nil {
		return nil, err
	}
	return &RootSecrets{
		root_priv_key:  ed25519_private_key,
		root_self_cert: derBytes,
		root_id_hash:   id_hash,
	}, nil
}

type TLSIdentity struct {
	priv_key   crypto.PrivateKey
	cert_bytes []byte
}

func NewTLSIdentity() (*TLSIdentity, error) {
	ed25519_public_key, ed25519_private_key, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, err
	}

	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128) // 2^128
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		return nil, err
	}
	template := x509.Certificate{
		NotBefore:             time.Now().Add(time.Duration(-1) * time.Second), //1-sec backdate, for badly synced peers.
		NotAfter:              time.Now().Add(7 * 24 * time.Hour),              // Valid for 7 days
		SerialNumber:          serialNumber,
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
		IsCA:                  true,
		BasicConstraintsValid: true,
	}
	derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, ed25519_public_key, ed25519_private_key)
	if err != nil {
		return nil, err
	}
}

func IdentityHashFromRootPublicKey(pub crypto.PublicKey) (string, error) {
	derBytes, err := x509.MarshalPKIXPublicKey(pub)
	if err != nil {
		return "", fmt.Errorf("unable to marshal public key to DER: %v", err)
	}
	hasher := sha3.New512()
	hasher.Write(derBytes)
	return "I" + base58.Encode(hasher.Sum(nil)), nil
}
