package ahmp

import (
	"github.com/google/uuid"

	abyss "abyss_neighbor_discovery/interfaces"
)

type JN struct {
	SenderSessionID uuid.UUID
	Text            string
}
type JOK struct {
	SenderSessionID uuid.UUID
	RecverSessionID uuid.UUID
	Neighbors       []abyss.ANDFullPeerSessionInfo
	Text            string
}
type JDN struct {
	RecverSessionID uuid.UUID
	Text            string
	Code            int
}
type JNI struct {
	SenderSessionID uuid.UUID
	RecverSessionID uuid.UUID
	Neighbor        abyss.ANDFullPeerSessionInfo
}
type MEM struct {
	SenderSessionID uuid.UUID
	RecverSessionID uuid.UUID
}
type SNB struct {
	SenderSessionID uuid.UUID
	RecverSessionID uuid.UUID
	MemberInfos     []abyss.ANDPeerSessionInfo
}
type CRR struct {
	SenderSessionID uuid.UUID
	RecverSessionID uuid.UUID
	MemberInfos     []abyss.ANDPeerSessionInfo
}
type RST struct {
	SenderSessionID uuid.UUID //may nil.
	RecverSessionID uuid.UUID
}

type SOA struct {
	SenderSessionID uuid.UUID
	RecverSessionID uuid.UUID
	Objects         []abyss.ObjectInfo
}
type SOD struct {
	SenderSessionID uuid.UUID
	RecverSessionID uuid.UUID
	ObjectIDs       []uuid.UUID
}
