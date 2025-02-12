package interfaces

import "github.com/google/uuid"

type IPathResolver interface {
	PathToSessionID(path string) (uuid.UUID, bool)
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
