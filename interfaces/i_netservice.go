package interfaces

import (
	"abyss_neighbor_discovery/aurl"
	"context"
	"net"

	"github.com/quic-go/quic-go"
)

type IPreAccepter interface {
	PreAccept(peer_hash string, address *net.UDPAddr) (bool, int, string)
}

type AbystInboundSession struct {
	PeerHash   string
	Connection quic.Connection
}

// 1. AbyssAsync 'always' succeeds, resulting in IANDPeer -> if connection failed, IANDPeer methods return error.
// 2. Abyst may fail at any moment
type INetworkService interface {
	LocalIdentity() IHostIdentity
	LocalAURL() *aurl.AURL

	HandlePreAccept(preaccept_handler IPreAccepter) // if false, return status code and message

	ListenAndServe(ctx context.Context) error

	AppendKnownPeer(root_cert string, handshake_key_cert string) error
	AppendKnownPeerDer(root_cert []byte, handshake_key_cert []byte) error
	RemoveKnownPeer(peer_hash string)

	ConnectAbyssAsync(ctx context.Context, url *aurl.AURL) error
	GetAbyssPeerChannel() chan IANDPeer //abyss mutual connection
	CloseAbyssPeer(peer IANDPeer)

	ConnectAbyst(ctx context.Context, peer_hash string) (quic.Connection, error)
}

type IAddressSelector interface {
	FilterAddressCandidates(addresses []*net.UDPAddr) []*net.UDPAddr
}

// TLS ALPN code
const NextProtoAbyss = "abyss"
