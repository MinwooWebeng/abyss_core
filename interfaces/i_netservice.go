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

type INetworkService interface {
	HandlePreAccept(preaccept_handler IPreAccepter) // if false, return status code and message

	ListenAndServe(ctx context.Context)

	ConnectAsync(address string)
	AcceptChannel() chan IANDPeer
}
