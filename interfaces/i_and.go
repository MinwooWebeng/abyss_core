package interfaces

import "github.com/google/uuid"

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
	PeerSession
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
	JoinWorld(local_session_id uuid.UUID, peer IANDPeer, path string) ANDERROR
	CancelJoin(local_session_id uuid.UUID) ANDERROR
	AcceptSession(local_session_id uuid.UUID, peer_session PeerSession) ANDERROR
	DeclineSession(local_session_id uuid.UUID, peer_session PeerSession, code int, message string) ANDERROR
	CloseWorld(local_session_id uuid.UUID) ANDERROR
	TimerExpire(local_session_id uuid.UUID) ANDERROR

	//ahmp messages
	JN(local_session_id uuid.UUID, peer_session PeerSession) ANDERROR
	JOK(local_session_id uuid.UUID, peer_session PeerSession, world_url string, member_sessions []PeerSessionInfo) ANDERROR
	JDN(local_session_id uuid.UUID, peer IANDPeer, code int, message string) ANDERROR
	JNI(local_session_id uuid.UUID, peer_session PeerSession, member_session PeerSessionInfo) ANDERROR
	MEM(local_session_id uuid.UUID, peer_session PeerSession) ANDERROR
	SNB(local_session_id uuid.UUID, peer_session PeerSession, member_hashes []string) ANDERROR
	CRR(local_session_id uuid.UUID, peer_session PeerSession, member_hashes []string) ANDERROR
	RST(local_session_id uuid.UUID, peer_session PeerSession) ANDERROR
}
