package interfaces

import (
	"abyss_neighbor_discovery/aurl"
	"context"
	"net"
)

type ILocalIdentity interface {
	IDHash() string
	Sign(payload []byte) []byte
}

type IPreAccepter interface {
	PreAccept(identity IRemoteIdentity, address *net.UDPAddr) (bool, int, string)
}

// 1. AbyssAsync 'always' succeeds, resulting in IANDPeer -> if connection failed, IANDPeer methods return error.
// 2. Abyst may fail at any moment
type INetworkService interface {
	HandlePreAccept(preaccept_handler IPreAccepter) // if false, return status code and message

	ListenAndServe(ctx context.Context) error

	ConnectAbyssAsync(ctx context.Context, url *aurl.AURL) error
	ConnectAbyst(ctx context.Context, url *aurl.AURL) (IAbystClientPeer, error)

	GetAbyssPeerChannel() chan IANDPeer               //abyss mutual connection
	GetAbystServerPeerChannel() chan IAbystServerPeer //abyst server-side connection
}

type IAddressSelector interface {
	FilterAddressCandidates(addresses []*net.UDPAddr) []*net.UDPAddr
}

// TLS ALPN code
const NextProtoAbyss = "abyss"
