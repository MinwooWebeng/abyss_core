package net_service

import (
	"bytes"
	"context"
	"errors"
	"net"

	"github.com/fxamacker/cbor/v2"
	"github.com/quic-go/quic-go"
)

type AbyssOutbound struct {
	connection    quic.Connection
	peer_identity *PeerIdentity
	addresses     []*net.UDPAddr
	cbor_encoder  *cbor.Encoder
	err           error
}

func (h *BetaNetService) PrepareAbyssOutbound(ctx context.Context, connection quic.Connection, peer_hash string, addresses []*net.UDPAddr) {
	//watchdog.Info("outbound detected")
	result := AbyssOutbound{connection, nil, nil, nil, errors.New("unknown error")}
	defer func() {
		h.abyssOutBound <- result
	}()

	//get self-signed TLS certificate that the peer presented.
	tls_info := connection.ConnectionState().TLS
	client_tls_cert := tls_info.PeerCertificates[0] //*x509.Certificate, validated

	ahmp_stream, err := connection.OpenStreamSync(ctx)
	if err != nil {
		result.err = err
		return
	}
	ahmp_cbor_enc := cbor.NewEncoder(ahmp_stream)
	ahmp_cbor_dec := cbor.NewDecoder(ahmp_stream)

	known_identity, ok := h.findKnownPeer(peer_hash) //TODO: change to waitForPeerIdentity
	if !ok {
		result.err = errors.New("unknown peer")
		return
	}

	//send {local peer_hash, local tls-abyss binding cert} encrypted with remote handshake key.
	var handshake_1_buf bytes.Buffer
	cbor.MarshalToBuffer(h.localIdentity.IDHash(), &handshake_1_buf)
	cbor.MarshalToBuffer(h.tlsIdentity.abyss_bind_cert, &handshake_1_buf)
	handshake_1_payload, err := known_identity.EncryptHandshake(handshake_1_buf.Bytes())
	if err != nil {
		result.err = err
		return
	}
	if err := ahmp_cbor_enc.Encode(handshake_1_payload); err != nil {
		result.err = err
		return
	}

	//receive accepter-side self-authentication
	var handshake_2_payload []byte
	if err := ahmp_cbor_dec.Decode(&handshake_2_payload); err != nil {
		result.err = err
		return
	}
	if err := known_identity.VerifyTLSBinding(handshake_2_payload, client_tls_cert); err != nil {
		result.err = err
		return
	}

	result = AbyssOutbound{connection, known_identity, addresses, ahmp_cbor_enc, nil}
}
