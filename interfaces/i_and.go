package interfaces

import (
	"abyss_neighbor_discovery/aurl"

	"github.com/google/uuid"
)

type NeighborEventType int

const (
	ANDSessionRequest NeighborEventType = iota + 2
	ANDSessionReady
	ANDSessionClose
	ANDJoinSuccess
	ANDJoinFail
	ANDConnectRequest
	ANDTimerRequest
	ANDNeighborEventDebug
)

type NeighborEvent struct {
	Type           NeighborEventType
	LocalSessionID uuid.UUID
	ANDPeerSession
	Text   string
	Value  int
	Object interface{}
}

type ANDERROR int

const (
	_      ANDERROR = iota //no error
	EINVAL                 //invalid argument
	EPANIC                 //unrecoverable internal error (must not occur)
)

type INeighborDiscovery interface { // all calls must be thread-safe
	EventChannel() chan NeighborEvent
	ResetPeerSession(local_session_id uuid.UUID, peer IANDPeer, peer_session_id uuid.UUID)

	//calls
	PeerConnected(peer IANDPeer) ANDERROR
	PeerClose(peer IANDPeer) ANDERROR
	OpenWorld(local_session_id uuid.UUID, world_url string) ANDERROR
	JoinWorld(local_session_id uuid.UUID, abyss_url *aurl.AURL) ANDERROR
	CancelJoin(local_session_id uuid.UUID) ANDERROR //This requests the join procedure to yield ANDJoinFail event. However, the request may be ignored.
	AcceptSession(local_session_id uuid.UUID, peer_session ANDPeerSession) ANDERROR
	DeclineSession(local_session_id uuid.UUID, peer_session ANDPeerSession, code int, message string) ANDERROR
	ConfirmLeave(local_session_id uuid.UUID, peer_session ANDPeerSession)
	CloseWorld(local_session_id uuid.UUID) ANDERROR
	TimerExpire(local_session_id uuid.UUID) ANDERROR

	//ahmp messages
	JN(local_session_id uuid.UUID, peer_session ANDPeerSession) ANDERROR
	JOK(local_session_id uuid.UUID, peer_session ANDPeerSession, world_url string, member_sessions []ANDPeerSessionInfo) ANDERROR
	JDN(local_session_id uuid.UUID, peer IANDPeer, code int, message string) ANDERROR
	JNI(local_session_id uuid.UUID, peer_session ANDPeerSession, member_session ANDPeerSessionInfo) ANDERROR
	MEM(local_session_id uuid.UUID, peer_session ANDPeerSession) ANDERROR
	SNB(local_session_id uuid.UUID, peer_session ANDPeerSession, member_hashes []string) ANDERROR
	CRR(local_session_id uuid.UUID, peer_session ANDPeerSession, member_hashes []string) ANDERROR
	RST(local_session_id uuid.UUID, peer_session ANDPeerSession) ANDERROR
}
