package net_service

type BetaLocalIdentity struct {
	hash string
}

func NewBetaLocalIdentity(hash string) *BetaLocalIdentity {
	return &BetaLocalIdentity{hash}
}

func (i *BetaLocalIdentity) IDHash() string {
	return i.hash
}
func (i *BetaLocalIdentity) Sign(payload []byte) []byte {
	return []byte(i.hash)
}
