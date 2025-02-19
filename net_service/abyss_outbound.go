package net_service

import (
	ahmp_message "abyss_neighbor_discovery/message"
	"context"

	"github.com/fxamacker/cbor/v2"
	"github.com/quic-go/quic-go"
)

type AbyssOutbound struct {
	connection quic.Connection
	identity   string
}

func (h *BetaNetService) PrepareAbyssOutbound(ctx context.Context, connection quic.Connection) {
	ahmp_stream, err := connection.OpenStreamSync(ctx)
	if err != nil {
		connection.CloseWithError(0, err.Error())
		return
	}
	ahmp_cbor_enc := cbor.NewEncoder(ahmp_stream)
	ahmp_cbor_dec := cbor.NewDecoder(ahmp_stream)

	ahmp_cbor_enc.Encode(ahmp_message.DummyAuth{Name: h.localIdentity.IDHash()})

	var dummy_auth ahmp_message.DummyAuth
	if err = ahmp_cbor_dec.Decode(&dummy_auth); err != nil {
		return
	}

	h.abyssOutBound <- AbyssOutbound{connection, dummy_auth.Name}
}
