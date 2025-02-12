package interfaces

import (
	"github.com/google/uuid"
)

type PeerSession struct {
	Peer          IANDPeer
	PeerSessionID uuid.UUID
}

type PeerSessionInfo struct {
	AURL      AURL
	SessionID uuid.UUID
}

type IRemoteIdentity interface {
	IDHash() string
	ValidateSignature(payload []byte, hash []byte) bool
}

type IANDPeer interface {
	IRemoteIdentity

	AhmpCh() chan IAhmpMessage

	TrySendJN(local_session_id uuid.UUID, path string) bool
	TrySendJOK(peer_session_id uuid.UUID, local_session_id uuid.UUID, world_url string, member_sessions []PeerSession) bool
	TrySendJDN(peer_session_id uuid.UUID, code int, message string) bool
	TrySendJNI(peer_session_id uuid.UUID, member_session PeerSession) bool
	TrySendMEM(peer_session_id uuid.UUID, local_session_id uuid.UUID) bool
	TrySendSNB(peer_session_id uuid.UUID, member_sessions []PeerSessionInfo) bool
	TrySendCRR(peer_session_id uuid.UUID, member_sessions []PeerSessionInfo) bool
	TrySendRST(peer_session_id uuid.UUID, local_session_id uuid.UUID) bool
}
