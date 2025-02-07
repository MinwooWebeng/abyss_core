package and

import "github.com/google/uuid"

type PeerSessionInfo struct {
	PeerHash  string
	AURL      string
	SessionID uuid.UUID
}

type IANDPeer interface {
	IDHash() string
	Status() int //0: OK | others: implementation-dependent error code.

	TrySendJN(local_session_id uuid.UUID, path string) bool
	TrySendJOK(local_session_id uuid.UUID, peer_session_id uuid.UUID, world_url string, member_sessions []PeerSession) bool
	TrySendJDN(peer_session_id uuid.UUID, code int, message string) bool
	TrySendJNI(peer_session_id uuid.UUID, member_session PeerSession) bool
	TrySendMEM(local_session_id uuid.UUID, peer_session_id uuid.UUID) bool
	TrySendSNB(peer_session_id uuid.UUID, member_sessions []PeerSessionInfo) bool
	TrySendCRR(peer_session_id uuid.UUID, member_sessions []PeerSessionInfo) bool
	TrySendRST(local_session_id uuid.UUID, peer_session_id uuid.UUID) bool
}
