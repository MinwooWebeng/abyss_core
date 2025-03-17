package interfaces

type IRemoteIdentity interface {
	IDHash() string
	Encrypt(payload []byte) ([]byte, error)
}

type ILocalIdentity interface {
	Certificate() []byte //x509 certificate, not pem
	SignTLSCertificate() ([]byte, error)
	IDHash() string
	Decrypt(payload []byte) ([]byte, error)
}
