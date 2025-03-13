package interfaces

type IRemoteIdentity interface {
	IDHash() string
	ValidateSignature(payload []byte, signature []byte) bool
}

type ILocalIdentity interface {
	IDHash() string
	Sign(payload []byte) []byte
	Decrypt(payload []byte) ([]byte, bool)
}
