package interfaces

import (
	"abyss_neighbor_discovery/aurl"
	"context"
	"net"

	"github.com/quic-go/quic-go"
)

type IPreAccepter interface {
	PreAccept(identity IRemoteIdentity, address *net.UDPAddr) (bool, int, string)
}

type AbystInboundSession struct {
	PeerHash   string
	Connection quic.Connection
}

// 1. AbyssAsync 'always' succeeds, resulting in IANDPeer -> if connection failed, IANDPeer methods return error.
// 2. Abyst may fail at any moment
type INetworkService interface {
	LocalAURL() *aurl.AURL

	HandlePreAccept(preaccept_handler IPreAccepter) // if false, return status code and message

	ListenAndServe(ctx context.Context) error

	ConnectAbyssAsync(ctx context.Context, url *aurl.AURL) error
	GetAbyssPeerChannel() chan IANDPeer //abyss mutual connection
	CloseAbyssPeer(peer IANDPeer)

	ConnectAbyst(ctx context.Context, peer_hash string) (quic.Connection, error)
	GetAbystServerPeerChannel() chan AbystInboundSession //abyst server-side connection
}

type IAddressSelector interface {
	FilterAddressCandidates(addresses []*net.UDPAddr) []*net.UDPAddr
}

// TLS ALPN code
const NextProtoAbyss = "abyss"
