package interfaces

type IRemoteIdentity interface {
	IDHash() string
	ValidateSignature(payload []byte, hash []byte) bool
}

type ILocalIdentity interface {
	IDHash() string
	Sign(payload []byte) []byte
}
