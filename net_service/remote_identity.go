package net_service

// import (
// 	"crypto/rand"
// 	"crypto/rsa"
// 	"crypto/x509"
// 	"errors"

// 	"golang.org/x/crypto/sha3"
// )

// type RemoteIdentity struct {
// 	cert   *x509.Certificate
// 	pubkey *rsa.PublicKey
// 	hash   string
// }

// func NewRemoteIdentity(cert *x509.Certificate) (*RemoteIdentity, error) {
// 	pubkey, ok := cert.PublicKey.(*rsa.PublicKey)
// 	if !ok {
// 		return nil, errors.New("unsupported public key")
// 	}

// 	return &RemoteIdentity{
// 		cert:   cert,
// 		pubkey: pubkey,
// 		hash:   cert.Issuer.CommonName,
// 	}, nil
// }

// func (i *RemoteIdentity) IDHash() string {
// 	return i.hash
// }
// func (i *RemoteIdentity) Encrypt(payload []byte) ([]byte, error) {
// 	encryptedPaylaod, err := rsa.EncryptOAEP(sha3.New512(), rand.Reader, i.pubkey, payload, nil)
// 	return encryptedPaylaod, err
// }
