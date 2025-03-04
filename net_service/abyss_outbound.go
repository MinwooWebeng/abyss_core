package net_service

import (
	"context"
	"net"

	"github.com/fxamacker/cbor/v2"
	"github.com/quic-go/quic-go"

	"abyss_neighbor_discovery/ahmp"
	abyss "abyss_neighbor_discovery/interfaces"
)

type AbyssOutbound struct {
	connection   quic.Connection
	identity     abyss.IRemoteIdentity
	addresses    []*net.UDPAddr
	cbor_encoder *cbor.Encoder
}

func (h *BetaNetService) PrepareAbyssOutbound(ctx context.Context, connection quic.Connection, identity string, addresses []*net.UDPAddr) {
	if connection == nil {
		//failed to connect
		h.abyssOutBound <- AbyssOutbound{nil, nil, nil, nil}
		return
	}

	ahmp_stream, err := connection.OpenStreamSync(ctx)
	if err != nil {
		connection.CloseWithError(0, err.Error())
		return
	}
	ahmp_cbor_enc := cbor.NewEncoder(ahmp_stream)
	ahmp_cbor_dec := cbor.NewDecoder(ahmp_stream)

	ahmp_cbor_enc.Encode(ahmp.DummyAuth{Name: h.localIdentity.IDHash()})

	var dummy_auth ahmp.DummyAuth
	if err = ahmp_cbor_dec.Decode(&dummy_auth); err != nil {
		return
	}

	h.abyssOutBound <- AbyssOutbound{connection, NewBetaRemoteIdentity(identity), addresses, ahmp_cbor_enc}
}
