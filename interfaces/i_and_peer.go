package interfaces

import (
	"context"

	"github.com/MinwooWebeng/abyss_core/aurl"

	"github.com/google/uuid"
)

type ANDPeerSession struct {
	Peer          IANDPeer
	PeerSessionID uuid.UUID
}

type ANDPeerSessionInfo struct {
	PeerHash  string
	SessionID uuid.UUID
}

type ANDFullPeerSessionInfo struct {
	AURL                       *aurl.AURL
	SessionID                  uuid.UUID
	RootCertificateDer         []byte
	HandshakeKeyCertificateDer []byte
}

type IANDPeer interface {
	IDHash() string
	RootCertificateDer() []byte
	HandshakeKeyCertificateDer() []byte

	IsConnected() bool
	AURL() *aurl.AURL

	//inactivity check
	Context() context.Context
	Activate()
	Renew()
	Deactivate()
	Error() error

	AhmpCh() chan any

	TrySendJN(local_session_id uuid.UUID, path string) bool
	TrySendJOK(local_session_id uuid.UUID, peer_session_id uuid.UUID, world_url string, member_sessions []ANDPeerSession) bool
	TrySendJDN(peer_session_id uuid.UUID, code int, message string) bool
	TrySendJNI(local_session_id uuid.UUID, peer_session_id uuid.UUID, member_session ANDPeerSession) bool
	TrySendMEM(local_session_id uuid.UUID, peer_session_id uuid.UUID) bool
	TrySendSJN(local_session_id uuid.UUID, peer_session_id uuid.UUID, member_sessions []ANDPeerSessionInfo) bool
	TrySendCRR(local_session_id uuid.UUID, peer_session_id uuid.UUID, member_sessions []ANDPeerSessionInfo) bool
	TrySendRST(local_session_id uuid.UUID, peer_session_id uuid.UUID) bool

	TrySendSOA(local_session_id uuid.UUID, peer_session_id uuid.UUID, objects []ObjectInfo) bool
	TrySendSOD(local_session_id uuid.UUID, peer_session_id uuid.UUID, objectIDs []uuid.UUID) bool
}
