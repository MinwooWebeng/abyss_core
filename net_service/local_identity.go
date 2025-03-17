package net_service

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"time"

	"github.com/btcsuite/btcutil/base58"
	"golang.org/x/crypto/sha3"
)

type LocalIdentity struct {
	priv_key   crypto.PrivateKey
	cert       *x509.Certificate
	cert_bytes []byte
	hash       string
}

func NewLocalIdentity(priv_key crypto.PrivateKey) (*LocalIdentity, error) {
	pubKeyBytes := x509.MarshalPKCS1PublicKey(&priv_key.PublicKey)
	hasher := sha3.New512()
	hasher.Write(pubKeyBytes)
	digest := "I" + base58.Encode(hasher.Sum(nil))

	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128) // 2^128
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		return nil, err
	}
	certTemplate := x509.Certificate{
		Subject: pkix.Name{
			CommonName: digest,
		},
		NotBefore:             time.Now().Add(time.Duration(-1) * time.Second), //1-sec backdate, for badly synced peers.
		SerialNumber:          serialNumber,
		KeyUsage:    x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
		IsCA:                  true,
		BasicConstraintsValid: true,
	}
	cert_bytes, err := x509.CreateCertificate(rand.Reader, &certTemplate, &certTemplate, &priv_key.PublicKey, priv_key)
	if err != nil {
		return nil, err
	}

	cert, err := x509.ParseCertificate(cert_bytes)
	if err != nil {
		return nil, err
	}

	return &LocalIdentity{
		priv_key:   priv_key,
		cert:       cert,
		cert_bytes: cert_bytes,
		hash:       digest,
	}, nil
}

func (i *LocalIdentity) Certificate() []byte {
	return i.cert_bytes
}
func (i *LocalIdentity) SignTLSCertificate(tls_cert *x509.Certificate) ([]byte, error) {
	certTemplate := x509.Certificate{
		Subject: pkix.Name{
			CommonName: i.hash,
		},
		NotBefore: time.Now(),
		NotAfter:  time.Now().Add(7 * 24 * time.Hour),
		PublicKeyAlgorithm: ,
	}
}
func (i *LocalIdentity) IDHash() string {
	return i.hash
}
func (i *LocalIdentity) Decrypt(payload []byte) ([]byte, error) {

}
