package net_service

import (
	"context"
	"errors"

	"github.com/fxamacker/cbor/v2"
	"github.com/quic-go/quic-go"

	"abyss_core/ahmp"
)

type AbyssInbound struct {
	connection   quic.Connection
	peer_hash    string
	cbor_decoder *cbor.Decoder
	AhmpChannel  chan any
	err          error
}

func (h *BetaNetService) PrepareAbyssInbound(ctx context.Context, connection quic.Connection) {
	result := AbyssInbound{connection, "", nil, nil, errors.New("unknown error")}
	defer func() {
		h.abyssInBound <- result
	}()

	//get self-signed TLS certificate that the peer presented.
	tls_info := connection.ConnectionState().TLS
	client_tls_cert := tls_info.PeerCertificates[0] //*x509.Certificate, validated

	ahmp_stream, err := connection.AcceptStream(ctx)
	if err != nil {
		result.err = err
		return
	}
	ahmp_cbor_enc := cbor.NewEncoder(ahmp_stream)
	ahmp_cbor_dec := cbor.NewDecoder(ahmp_stream)

	//receive connecter-side self-authentication payload
	var handshake_1_raw []byte
	if err := ahmp_cbor_dec.Decode(&handshake_1_raw); err != nil {
		result.err = err
		return
	}
	handshake_1, err := h.localIdentity.DecryptHandshake(handshake_1_raw)
	if err != nil {
		result.err = err
		return
	}

	//parse handshake 1
	var peer_hash string
	rest, err := cbor.UnmarshalFirst(handshake_1, &peer_hash)
	if err != nil {
		result.err = err
		return
	}
	var abyss_bind_cert []byte
	if err := cbor.Unmarshal(rest, &abyss_bind_cert); err != nil {
		result.err = err
		return
	}

	//retrieve known identity and verify
	known_identity, ok := h.findKnownPeer(peer_hash) //TODO: change to waitForPeerIdentity
	if !ok {
		result.err = errors.New("unknown peer")
		return
	}
	if err := known_identity.VerifyTLSBinding(abyss_bind_cert, client_tls_cert); err != nil {
		result.err = err
		return
	}

	//send local tls-abyss binding cert
	if err := ahmp_cbor_enc.Encode(h.tlsIdentity.abyss_bind_cert); err != nil {
		result.err = err
		return
	}

	result = AbyssInbound{connection, known_identity.root_id_hash, ahmp_cbor_dec, make(chan any, 8), nil}
	go result.listenAhmp()
}

func (h *BetaNetService) PrepareAbystInbound(ctx context.Context, connection quic.Connection) {
	//TODO: peer authentication
	//h.abystServerCH <- abyss.AbystInboundSession{PeerHash: "unknown", Connection: connection}
	h.abystServer.ServeQUICConn(connection)
}

func (h *AbyssInbound) listenAhmp() {
	for {
		var ahmp_type int
		h.cbor_decoder.Decode(&ahmp_type)

		switch ahmp_type {
		case ahmp.JN_T:
			//fmt.Println("receiving JN")
			var raw_msg ahmp.RawJN
			h.cbor_decoder.Decode(&raw_msg)
			parsed_msg, err := raw_msg.TryParse()
			if err != nil {
				h.AhmpChannel <- ahmp.INVAL{Err: err}
				continue
			}
			h.AhmpChannel <- parsed_msg
		case ahmp.JOK_T:
			//fmt.Println("receiving JOK")
			var raw_msg ahmp.RawJOK
			h.cbor_decoder.Decode(&raw_msg)
			parsed_msg, err := raw_msg.TryParse()
			if err != nil {
				h.AhmpChannel <- ahmp.INVAL{Err: err}
				continue
			}
			h.AhmpChannel <- parsed_msg
		case ahmp.JDN_T:
			//fmt.Println("receiving JDN")
			var raw_msg ahmp.RawJDN
			h.cbor_decoder.Decode(&raw_msg)
			parsed_msg, err := raw_msg.TryParse()
			if err != nil {
				h.AhmpChannel <- ahmp.INVAL{Err: err}
				continue
			}
			h.AhmpChannel <- parsed_msg
		case ahmp.JNI_T:
			//fmt.Println("receiving JNI")
			var raw_msg ahmp.RawJNI
			h.cbor_decoder.Decode(&raw_msg)
			parsed_msg, err := raw_msg.TryParse()
			if err != nil {
				h.AhmpChannel <- ahmp.INVAL{Err: err}
				continue
			}
			h.AhmpChannel <- parsed_msg
		case ahmp.MEM_T:
			//fmt.Println("receiving MEM")
			var raw_msg ahmp.RawMEM
			h.cbor_decoder.Decode(&raw_msg)
			parsed_msg, err := raw_msg.TryParse()
			if err != nil {
				h.AhmpChannel <- ahmp.INVAL{Err: err}
				continue
			}
			h.AhmpChannel <- parsed_msg
		case ahmp.SNB_T:
			//fmt.Println("receiving SNB")
			var raw_msg ahmp.RawSNB
			h.cbor_decoder.Decode(&raw_msg)
			parsed_msg, err := raw_msg.TryParse()
			if err != nil {
				h.AhmpChannel <- ahmp.INVAL{Err: err}
				continue
			}
			h.AhmpChannel <- parsed_msg
		case ahmp.CRR_T:
			//fmt.Println("receiving CRR")
			var raw_msg ahmp.RawCRR
			h.cbor_decoder.Decode(&raw_msg)
			parsed_msg, err := raw_msg.TryParse()
			if err != nil {
				h.AhmpChannel <- ahmp.INVAL{Err: err}
				continue
			}
			h.AhmpChannel <- parsed_msg
		case ahmp.RST_T:
			//fmt.Println("receiving RST")
			var raw_msg ahmp.RawRST
			h.cbor_decoder.Decode(&raw_msg)
			parsed_msg, err := raw_msg.TryParse()
			if err != nil {
				h.AhmpChannel <- ahmp.INVAL{Err: err}
				continue
			}
			h.AhmpChannel <- parsed_msg
		case ahmp.SOA_T:
			//fmt.Println("receiving SOA")
			var raw_msg ahmp.RawSOA
			h.cbor_decoder.Decode(&raw_msg)
			parsed_msg, err := raw_msg.TryParse()
			if err != nil {
				h.AhmpChannel <- ahmp.INVAL{Err: err}
				continue
			}
			h.AhmpChannel <- parsed_msg
		case ahmp.SOD_T:
			//fmt.Println("receiving SOD")
			var raw_msg ahmp.RawSOD
			h.cbor_decoder.Decode(&raw_msg)
			parsed_msg, err := raw_msg.TryParse()
			if err != nil {
				h.AhmpChannel <- ahmp.INVAL{Err: err}
				continue
			}
			h.AhmpChannel <- parsed_msg
		default:
			h.AhmpChannel <- ahmp.INVAL{Err: errors.New("unknown AHMP message type")}
			continue
		}
	}
}
