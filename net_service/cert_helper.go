package net_service

import (
	"bytes"
	"crypto"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"fmt"
	"math/big"
	"time"

	"github.com/btcsuite/btcutil/base58"
	"golang.org/x/crypto/sha3"
)

type RootSecrets struct {
	root_priv_key       crypto.PrivateKey
	root_self_cert_x509 *x509.Certificate
	root_self_cert      string //pem
	root_id_hash        string

	handshake_priv_key *rsa.PrivateKey //may support others in future
	handshake_key_cert string          //pem
}

// To generate root key, use ed25519.GenerateKey(rand.Reader)
func NewRootIdentity(root_private_key crypto.PrivateKey) (*RootSecrets, error) {
	root_public_key := root_private_key.(*ed25519.PrivateKey).Public()

	//root certificate
	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128) // 2^128
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		return nil, err
	}
	peer_hash, err := AbyssIdFromKey(root_public_key)
	if err != nil {
		return nil, err
	}
	r_template := x509.Certificate{
		Issuer: pkix.Name{
			CommonName: peer_hash,
		},
		Subject: pkix.Name{
			CommonName: peer_hash,
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
	if err != nil {
		return nil, err
	}
	serialNumber, err = rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		return nil, err
	}
	h_template := x509.Certificate{
		Issuer: pkix.Name{
			CommonName: peer_hash,
		},
		Subject: pkix.Name{
			CommonName: "H-OAEP-SHA3-256-" + peer_hash, //handshake encryption key, RSA OAEP with SHA3-256 hash function.
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

	var root_cert_buf bytes.Buffer
	err = pem.Encode(&root_cert_buf, &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: r_derBytes,
	})
	if err != nil {
		return nil, err
	}

	var handshake_cert_buf bytes.Buffer
	err = pem.Encode(&handshake_cert_buf, &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: h_derBytes,
	})
	if err != nil {
		return nil, err
	}
	return &RootSecrets{
		root_priv_key:       root_private_key,
		root_self_cert_x509: r_x509,
		root_self_cert:      root_cert_buf.String(),
		root_id_hash:        peer_hash,

		handshake_priv_key: handshake_private_key,
		handshake_key_cert: handshake_cert_buf.String(),
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

func (r *RootSecrets) IDHash() string {
	return r.root_id_hash
}
func (r *RootSecrets) DecryptHandshake(body []byte) ([]byte, error) {
	res, err := rsa.DecryptOAEP(sha3.New256(), rand.Reader, r.handshake_priv_key, body, nil)
	return res, err
}
func (r *RootSecrets) RootCertificate() string {
	return r.root_self_cert
}
func (r *RootSecrets) HandshakeKeyCertificate() string {
	return r.handshake_key_cert
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
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment | x509.KeyUsageCertSign,
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
	root_self_cert_x509 *x509.Certificate
	root_id_hash        string
	handshake_pub_key   *rsa.PublicKey
}

func NewPeerIdentity(root_self_cert []byte, handshake_key_cert []byte) (*PeerIdentity, error) {
	root_self_cert_x509, err := x509.ParseCertificate(root_self_cert)
	if err != nil {
		return nil, err
	}
	handshake_key_cert_x509, err := x509.ParseCertificate(handshake_key_cert)
	if err != nil {
		return nil, err
	}

	if root_self_cert_x509.Issuer.CommonName != root_self_cert_x509.Subject.CommonName {
		return nil, errors.New("invalid root certificate")
	}
	peer_hash, err := AbyssIdFromKey(root_self_cert_x509.PublicKey)
	if err != nil {
		return nil, err
	}
	if peer_hash != root_self_cert_x509.Issuer.CommonName {
		return nil, errors.New("invalid root certificate")
	}

	if handshake_key_cert_x509.Issuer.CommonName != root_self_cert_x509.Issuer.CommonName {
		return nil, errors.New("issuer mismatch")
	}
	if handshake_key_cert_x509.Subject.CommonName != "H-OAEP-SHA3-256-"+root_self_cert_x509.Issuer.CommonName {
		return nil, errors.New("unsupported public key encryption scheme")
	}
	if err := handshake_key_cert_x509.CheckSignatureFrom(root_self_cert_x509); err != nil {
		return nil, err
	}
	pkey, ok := handshake_key_cert_x509.PublicKey.(*rsa.PublicKey)
	if !ok {
		return nil, errors.New("unsupported public key")
	}
	return &PeerIdentity{
		root_self_cert_x509: root_self_cert_x509,
		root_id_hash:        peer_hash,
		handshake_pub_key:   pkey,
	}, nil
}

func (p *PeerIdentity) IDHash() string {
	return p.root_id_hash
}
func (p *PeerIdentity) EncryptHandshake(payload []byte) ([]byte, error) {
	res, err := rsa.EncryptOAEP(sha3.New256(), rand.Reader, p.handshake_pub_key, payload, nil)
	return res, err
}
func (p *PeerIdentity) VerifyTLSBinding(abyss_bind_cert_der []byte, tls_cert *x509.Certificate) error {
	abyss_bind_cert, err := x509.ParseCertificate(abyss_bind_cert_der)
	if err != nil {
		return err
	}
	//compare abyss_bind_cert.PublicKey and tls_cert.PublicKey
	if !abyss_bind_cert.PublicKey.(ed25519.PublicKey).Equal(tls_cert.PublicKey) {
		return errors.New("tls public key mismatch")
	}

	if abyss_bind_cert.Issuer.CommonName != p.root_self_cert_x509.Issuer.CommonName {
		return errors.New("issuer mismatch")
	}
	if abyss_bind_cert.Subject.CommonName != "T-"+p.root_self_cert_x509.Issuer.CommonName {
		return errors.New("subject mismatch")
	}
	return errors.Join(errors.New("VerifyTLSBinding"), abyss_bind_cert.CheckSignatureFrom(p.root_self_cert_x509))
}
