package interfaces

import (
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

type ConnectionFailInfo struct {
	address string
	code    int
	message string
}

type INetworkService interface {
	HandlePreAccept(preaccept_handler IPreAccepter) // if false, return status code and message

	ListenAndServe(ctx context.Context) error

	ConnectAsync(address string) error

	GetAbyssPeerChannel() chan IANDPeer               //abyss mutual connection
	GetAbystClientPeerChannel() chan IAbystClientPeer //abyst client-side connection
	GetAbystServerPeerChannel() chan IAbystServerPeer //abyst server-side connection

	GetConnectionFailChannel() chan ConnectionFailInfo
}

// TLS ALPN code
const NextProtoAbyss = "abyss"
