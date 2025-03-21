package interfaces

// type IRemoteIdentity interface {
// 	IDHash() string
// }

type ILocalIdentity interface {
	IDHash() string
	RootCertificate() string         //pem
	HandshakeKeyCertificate() string //pem
}
