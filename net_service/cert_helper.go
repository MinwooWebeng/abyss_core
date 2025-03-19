package net_service

import (
	"crypto"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"fmt"
	"math/big"
	"time"

	"github.com/btcsuite/btcutil/base58"
	"golang.org/x/crypto/sha3"
)

type RootSecrets struct {
	root_priv_key       crypto.PrivateKey
	root_self_cert_x509 *x509.Certificate
	root_self_cert      []byte //der
	root_id_hash        string

	handshake_priv_key *rsa.PrivateKey //may support others in future
	handshake_key_cert []byte          //der
}

func GenerateRootIdentity() (*RootSecrets, error) {
	root_public_key, root_private_key, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, err
	}

	res, err := LoadRootIdentity(root_public_key, root_private_key)
	return res, err
}

func LoadRootIdentity(root_public_key crypto.PublicKey, root_private_key crypto.PrivateKey) (*RootSecrets, error) {
	//root certificate
	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128) // 2^128
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		return nil, err
	}
	id_hash, err := AbyssIdFromKey(root_public_key)
	if err != nil {
		return nil, err
	}
	r_template := x509.Certificate{
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
	r_derBytes, err := x509.CreateCertificate(rand.Reader, &r_template, &r_template, root_public_key, root_private_key)
	if err != nil {
		return nil, err
	}
	r_x509, err := x509.ParseCertificate(r_derBytes)
	if err != nil {
		return nil, err
	}

	//handshake key
	handshake_private_key, err := rsa.GenerateKey(rand.Reader, 2048)
	serialNumber, err = rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		return nil, err
	}
	h_template := x509.Certificate{
		Issuer: pkix.Name{
			CommonName: id_hash,
		},
		Subject: pkix.Name{
			CommonName: "H-OAEP-SHA3-256-" + id_hash, //handshake encryption key, RSA OAEP with SHA3-256 hash function.
		},
		NotBefore:             time.Now().Add(time.Duration(-1) * time.Second), //1-sec backdate, for badly synced peers.
		SerialNumber:          serialNumber,
		KeyUsage:              x509.KeyUsageEncipherOnly,
		BasicConstraintsValid: true,
	}
	h_derBytes, err := x509.CreateCertificate(rand.Reader, &h_template, &r_template, &handshake_private_key.PublicKey, root_private_key)
	if err != nil {
		return nil, err
	}

	return &RootSecrets{
		root_priv_key:       root_private_key,
		root_self_cert_x509: r_x509,
		root_self_cert:      r_derBytes,
		root_id_hash:        id_hash,

		handshake_priv_key: handshake_private_key,
		handshake_key_cert: h_derBytes,
	}, nil
}
func AbyssIdFromKey(pub crypto.PublicKey) (string, error) {
	derBytes, err := x509.MarshalPKIXPublicKey(pub)
	if err != nil {
		return "", fmt.Errorf("unable to marshal public key to DER: %v", err)
	}
	hasher := sha3.New512()
	hasher.Write(derBytes)
	return "I" + base58.Encode(hasher.Sum(nil)), nil
}

func (r *RootSecrets) DecryptHandshake(body []byte) ([]byte, error) {
	res, err := rsa.DecryptOAEP(sha3.New256(), rand.Reader, r.handshake_priv_key, body, nil)
	return res, err
}

type TLSIdentity struct {
	priv_key        crypto.PrivateKey
	tls_self_cert   []byte //der
	abyss_bind_cert []byte //der
}

func (r *RootSecrets) NewTLSIdentity() (*TLSIdentity, error) {
	ed25519_public_key, ed25519_private_key, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, err
	}

	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128) // 2^128
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		return nil, err
	}
	self_template := x509.Certificate{
		NotBefore:             time.Now().Add(time.Duration(-1) * time.Second), //1-sec backdate, for badly synced peers.
		NotAfter:              time.Now().Add(7 * 24 * time.Hour),              // Valid for 7 days
		SerialNumber:          serialNumber,
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
		IsCA:                  true,
		BasicConstraintsValid: true,
	}
	self_derBytes, err := x509.CreateCertificate(rand.Reader, &self_template, &self_template, ed25519_public_key, ed25519_private_key)
	if err != nil {
		return nil, err
	}

	serialNumber, err = rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		return nil, err
	}
	auth_template := x509.Certificate{
		Issuer: pkix.Name{
			CommonName: r.root_id_hash,
		},
		Subject: pkix.Name{
			CommonName: "T-" + r.root_id_hash,
		},
		NotBefore:             time.Now().Add(time.Duration(-1) * time.Second), //1-sec backdate, for badly synced peers.
		NotAfter:              time.Now().Add(7 * 24 * time.Hour),              // Valid for 7 days
		SerialNumber:          serialNumber,
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
		BasicConstraintsValid: true,
	}
	auth_derBytes, err := x509.CreateCertificate(rand.Reader, &auth_template, r.root_self_cert_x509, ed25519_public_key, r.root_priv_key)
	if err != nil {
		return nil, err
	}

	return &TLSIdentity{
		priv_key:        ed25519_private_key,
		tls_self_cert:   self_derBytes,
		abyss_bind_cert: auth_derBytes,
	}, nil
}

type PeerIdentity struct {
	// [TODO]
	// construct from peer root cert and handshake key cert
	// encrypt payload with handshake public key
	// verify TLS-abyss binding
}
