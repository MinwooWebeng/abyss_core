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

type ObjectInfo struct {
	ID      uuid.UUID
	Address string
}

type IAbyssPeer interface {
	Hash() string
	AppendObjects(objects []ObjectInfo) bool
	DeleteObjects(objectIDs []uuid.UUID) bool
	Close() //confirm cleanup, must be called only once after receiving EWorldPeerLeave
}

type EWorldPeerRequest struct {
	PeerHash string
	Accept   func()
	Decline  func(code int, message string)
}
type EWorldPeerReady struct {
	Peer IAbyssPeer
}
type EPeerObjectAppend struct {
	PeerHash string
	Objects  []ObjectInfo
}
type EPeerObjectDelete struct {
	PeerHash  string
	ObjectIDs []uuid.UUID
}
type EWorldPeerLeave struct { //now, the peer must be closed as soon as possible.
	PeerHash string
}

type IAbyssWorld interface {
	SessionID() uuid.UUID
	GetEventChannel() chan any
}

type AbystInboundSession struct {
	PeerHash   string
	Connection quic.Connection
}

type IAbyssHost interface {
	GetLocalAbyssURL() *aurl.AURL

	//Abyss
	OpenWorld(web_url string) (IAbyssWorld, error)
	JoinWorld(ctx context.Context, abyss_url *aurl.AURL) (IAbyssWorld, error)
	LeaveWorld(world IAbyssWorld)

	//Abyst
	GetAbystAcceptChannel() chan AbystInboundSession
	GetAbystClientConnection(peer_hash string) (*http3.ClientConn, bool)
}
