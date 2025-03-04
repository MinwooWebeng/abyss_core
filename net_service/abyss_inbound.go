package net_service

import (
	"context"

	"github.com/fxamacker/cbor/v2"
	"github.com/quic-go/quic-go"

	"abyss_neighbor_discovery/ahmp"
	abyss "abyss_neighbor_discovery/interfaces"
)

type AbyssInbound struct {
	connection   quic.Connection
	identity     string
	cbor_decoder *cbor.Decoder

	AhmpChannel chan any
}

func (h *BetaNetService) PrepareAbyssInbound(ctx context.Context, connection quic.Connection) {
	ahmp_stream, err := connection.AcceptStream(ctx)
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

	result := AbyssInbound{connection, dummy_auth.Name, ahmp_cbor_dec, make(chan any, 8)}
	go result.listenAhmp()
	h.abyssInBound <- result
}

func (h *BetaNetService) PrepareAbystInbound(ctx context.Context, connection quic.Connection) {
	//TODO: peer authentication
	h.abystServerCH <- abyss.AbystInboundSession{PeerHash: "unknown", Connection: connection}
}

func (h *AbyssInbound) listenAhmp() {
	for {
		var ahmp_type int
		h.cbor_decoder.Decode(&ahmp_type)

		switch ahmp_type {
		case ahmp.JN_T:
			var raw_msg ahmp.RawJN
			h.cbor_decoder.Decode(&raw_msg)
			parsed_msg, err := raw_msg.TryParse()
			if err != nil {
				return
			}
			h.AhmpChannel <- parsed_msg
		case ahmp.JOK_T:
			var raw_msg ahmp.RawJOK
			h.cbor_decoder.Decode(&raw_msg)
			parsed_msg, err := raw_msg.TryParse()
			if err != nil {
				return
			}
			h.AhmpChannel <- parsed_msg
		case ahmp.JDN_T:
			var raw_msg ahmp.RawJDN
			h.cbor_decoder.Decode(&raw_msg)
			parsed_msg, err := raw_msg.TryParse()
			if err != nil {
				return
			}
			h.AhmpChannel <- parsed_msg
		case ahmp.JNI_T:
			var raw_msg ahmp.RawJNI
			h.cbor_decoder.Decode(&raw_msg)
			parsed_msg, err := raw_msg.TryParse()
			if err != nil {
				return
			}
			h.AhmpChannel <- parsed_msg
		case ahmp.MEM_T:
			var raw_msg ahmp.RawMEM
			h.cbor_decoder.Decode(&raw_msg)
			parsed_msg, err := raw_msg.TryParse()
			if err != nil {
				return
			}
			h.AhmpChannel <- parsed_msg
		case ahmp.SNB_T:
			var raw_msg ahmp.RawSNB
			h.cbor_decoder.Decode(&raw_msg)
			parsed_msg, err := raw_msg.TryParse()
			if err != nil {
				return
			}
			h.AhmpChannel <- parsed_msg
		case ahmp.CRR_T:
			var raw_msg ahmp.RawCRR
			h.cbor_decoder.Decode(&raw_msg)
			parsed_msg, err := raw_msg.TryParse()
			if err != nil {
				return
			}
			h.AhmpChannel <- parsed_msg
		case ahmp.RST_T:
			var raw_msg ahmp.RawRST
			h.cbor_decoder.Decode(&raw_msg)
			parsed_msg, err := raw_msg.TryParse()
			if err != nil {
				return
			}
			h.AhmpChannel <- parsed_msg
		case ahmp.SOA_T:
			var raw_msg ahmp.RawSOA
			h.cbor_decoder.Decode(&raw_msg)
			parsed_msg, err := raw_msg.TryParse()
			if err != nil {
				return
			}
			h.AhmpChannel <- parsed_msg
		case ahmp.SOD_T:
			var raw_msg ahmp.RawSOD
			h.cbor_decoder.Decode(&raw_msg)
			parsed_msg, err := raw_msg.TryParse()
			if err != nil {
				return
			}
			h.AhmpChannel <- parsed_msg
		default:
			panic("unknown AHMP message type")
		}
	}
}
