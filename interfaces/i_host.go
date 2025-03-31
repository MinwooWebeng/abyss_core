package interfaces

import (
	"abyss_neighbor_discovery/aurl"
	"context"

	"github.com/google/uuid"
	"github.com/quic-go/quic-go/http3"
)

type IPathResolver interface {
	PathToSessionID(path string, peer_hash string) (uuid.UUID, bool)
}

type ObjectInfo struct {
	ID   uuid.UUID
	Addr string
}

type IAbyssPeer interface {
	Hash() string
	AppendObjects(objects []ObjectInfo) bool
	DeleteObjects(objectIDs []uuid.UUID) bool
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
type EWorldTerminate struct{}

type IAbyssWorld interface {
	SessionID() uuid.UUID
	GetEventChannel() chan any
}

type IAbyssHost interface {
	GetLocalAbyssURL() *aurl.AURL

	OpenOutboundConnection(abyss_url *aurl.AURL)

	//Abyss
	OpenWorld(web_url string) (IAbyssWorld, error)
	JoinWorld(ctx context.Context, abyss_url *aurl.AURL) (IAbyssWorld, error)
	LeaveWorld(world IAbyssWorld) //this does not wait for world-related resource cleanup.
	// Each world should wait for its world termination event.

	//Abyst
	GetAbystClientConnection(ctx context.Context, peer_hash string) (*http3.ClientConn, error)
	GetAbystServerPeerChannel() chan AbystInboundSession
}
