package net_service

type BetaRemoteIdentity struct {
	hash string
}

func NewBetaRemoteIdentity(hash string) *BetaRemoteIdentity {
	return &BetaRemoteIdentity{hash}
}

func (i *BetaRemoteIdentity) IDHash() string {
	return i.hash
}
func (i *BetaRemoteIdentity) ValidateSignature(payload []byte, hash []byte) bool {
	return true
}
