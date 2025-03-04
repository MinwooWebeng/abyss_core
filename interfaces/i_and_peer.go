package interfaces

import (
	"abyss_neighbor_discovery/aurl"

	"github.com/google/uuid"
)

type ANDPeerSession struct {
	Peer          IANDPeer
	PeerSessionID uuid.UUID
}

type ANDPeerSessionInfo struct {
	AURL      *aurl.AURL
	SessionID uuid.UUID
}

type IANDPeer interface {
	IRemoteIdentity
	AURL() *aurl.AURL

	AhmpCh() chan any

	TrySendJN(local_session_id uuid.UUID, path string) bool
	TrySendJOK(local_session_id uuid.UUID, peer_session_id uuid.UUID, world_url string, member_sessions []ANDPeerSession) bool
	TrySendJDN(peer_session_id uuid.UUID, code int, message string) bool
	TrySendJNI(local_session_id uuid.UUID, peer_session_id uuid.UUID, member_session ANDPeerSession) bool
	TrySendMEM(local_session_id uuid.UUID, peer_session_id uuid.UUID) bool
	TrySendSNB(local_session_id uuid.UUID, peer_session_id uuid.UUID, member_sessions []ANDPeerSessionInfo) bool
	TrySendCRR(local_session_id uuid.UUID, peer_session_id uuid.UUID, member_sessions []ANDPeerSessionInfo) bool
	TrySendRST(local_session_id uuid.UUID, peer_session_id uuid.UUID) bool

	TrySendSOA(local_session_id uuid.UUID, peer_session_id uuid.UUID, objects []ObjectInfo) bool
	TrySendSOD(local_session_id uuid.UUID, peer_session_id uuid.UUID, objectIDs []uuid.UUID) bool
}
