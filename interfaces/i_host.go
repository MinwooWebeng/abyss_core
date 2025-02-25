package interfaces

import (
	"abyss_neighbor_discovery/aurl"
	"context"

	"github.com/google/uuid"
	"github.com/quic-go/quic-go"
	"github.com/quic-go/quic-go/http3"
)

type IPathResolver interface {
	PathToSessionID(path string, peer_hash string) (uuid.UUID, bool)
}
type ISessionRequestHandler interface {
	OnSessionRequest(local_session_id uuid.UUID, peer IANDPeer, peer_session_id uuid.UUID) (ok bool, code int, message string)
}
type ISessionReadyHandler interface {
	OnSessionReady(local_session_id uuid.UUID, peer IANDPeer, peer_session_id uuid.UUID)
}
type ISessionCloseHandler interface {
	OnSessionClose(local_session_id uuid.UUID, peer IANDPeer, peer_session_id uuid.UUID)
}
type IJoinSuccessHandler interface {
	OnJoinSuccess(local_session_id uuid.UUID, world_url string)
}
type IJoinFailHandler interface {
	OnJoinFail(local_session_id uuid.UUID, code int, message string)
}

type IAllHandler interface {
	IPathResolver
	ISessionRequestHandler
	ISessionReadyHandler
	ISessionCloseHandler
	IJoinSuccessHandler
	IJoinFailHandler
}

type SessionRequestArguments struct {
	LocalSessionID uuid.UUID
	PeerSession
}

type AbystInboundSession struct {
	PeerHash   string
	Connection quic.Connection
}

type IAbyssHost interface {
	GetLocalAbyssURL() *aurl.AURL

	//Abyss
	HandlePathResolution(handler IPathResolver)

	JoinWorld(ctx context.Context, session_id uuid.UUID, abyss_url string) chan SessionRequestArguments
	OpenWorld(session_id uuid.UUID, web_url string) chan SessionRequestArguments

	RespondSessionRequest(origin SessionRequestArguments, ok bool, code int, message string)

	LeaveWorld(session_id uuid.UUID)

	//Abyst
	GetAbystAcceptChannel() chan AbystInboundSession
	GetAbystClientConnection(peer_hash string) (*http3.ClientConn, bool)
}
