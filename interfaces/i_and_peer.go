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

	AhmpCh() chan IAhmpMessage

	TrySendJN(local_session_id uuid.UUID, path string) bool
	TrySendJOK(peer_session_id uuid.UUID, local_session_id uuid.UUID, world_url string, member_sessions []ANDPeerSession) bool
	TrySendJDN(peer_session_id uuid.UUID, code int, message string) bool
	TrySendJNI(peer_session_id uuid.UUID, member_session ANDPeerSession) bool
	TrySendMEM(peer_session_id uuid.UUID, local_session_id uuid.UUID) bool
	TrySendSNB(peer_session_id uuid.UUID, member_sessions []ANDPeerSessionInfo) bool
	TrySendCRR(peer_session_id uuid.UUID, member_sessions []ANDPeerSessionInfo) bool
	TrySendRST(peer_session_id uuid.UUID, local_session_id uuid.UUID) bool

	TrySendSOA(peer_session_id uuid.UUID, local_session_id uuid.UUID, objects []ObjectInfo) bool
	TrySendSOD(peer_session_id uuid.UUID, local_session_id uuid.UUID, objectIDs []uuid.UUID) bool

	Close()
}
