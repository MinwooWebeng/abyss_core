package interfaces

import (
	"github.com/google/uuid"
)

type AhmpMessageType int

const (
	JN AhmpMessageType = iota + 2
	JOK
	JDN
	JNI
	MEM
	SNB
	CRR
	RST
)

type IAhmpMessage interface {
	Type() AhmpMessageType
	Sender() IANDPeer
	SenderSessionID() uuid.UUID
	RecverSessionID() uuid.UUID
	Neighbors() []ANDPeerSessionInfo
	Texts() []string
	Text() string
	Code() int
}
