package net_service

import (
	"bytes"
	"context"
	"crypto/x509"

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
	//get self-signed TLS certificate that the peer presented.
	tls_info := connection.ConnectionState().TLS
	client_tls_cert := tls_info.PeerCertificates[0] //*x509.Certificate
	if client_tls_cert.CheckSignatureFrom(client_tls_cert) != nil {
		//abort connection.
		connection.CloseWithError(0, "peer certificate not self-signed")
		return
	}

	ahmp_stream, err := connection.AcceptStream(ctx)
	if err != nil {
		connection.CloseWithError(0, err.Error())
		return
	}
	ahmp_cbor_enc := cbor.NewEncoder(ahmp_stream)
	ahmp_cbor_dec := cbor.NewDecoder(ahmp_stream)

	//receive client-side self-authentication payload
	var client_self_auth_raw []byte
	if ahmp_cbor_dec.Decode(client_self_auth_raw) != nil {
		connection.CloseWithError(0, "malformed self-authentication payload")
		return
	}
	//decrypt the payload with local private key - verify if this payload is for me.
	client_self_auth_decrypted, err := h.localIdentity.DecryptHandshake(client_self_auth_raw)
	if err != nil {
		connection.CloseWithError(0, "failed to decrypt client-auth payload")
		return
	}
	//parse client hash, self-signed root cert, tls-verify cert
	var client_hash string
	rest, err := cbor.UnmarshalFirst(client_self_auth_decrypted, &client_hash)
	if err != nil {
		connection.CloseWithError(0, "malformed self-authentication payload")
		return
	}
	var client_self_signed_cert_raw []byte
	rest, err = cbor.UnmarshalFirst(rest, client_self_signed_cert_raw)
	if err != nil {
		connection.CloseWithError(0, "malformed self-authentication payload")
		return
	}
	client_self_signed_cert, err := x509.ParseCertificate(client_self_signed_cert_raw)
	if err != nil {
		connection.CloseWithError(0, "malformed self-authentication payload")
		return
	}
	var client_tls_verify_cert_raw []byte
	if cbor.Unmarshal(rest, client_tls_verify_cert_raw) != nil {
		connection.CloseWithError(0, "malformed self-authentication payload")
		return
	}
	client_tls_verify_cert, err := x509.ParseCertificate(client_tls_verify_cert_raw)
	if err != nil {
		connection.CloseWithError(0, "malformed self-authentication payload")
		return
	}

	//verify tls public key
	if client_tls_cert.Subject.CommonName != client_tls_cert.Issuer.CommonName ||
		client_tls_cert.Issuer.CommonName != client_tls_verify_cert.Subject.CommonName ||
		client_tls_verify_cert.Subject.CommonName != client_tls_verify_cert.Issuer.CommonName ||
		client_tls_verify_cert.Issuer.CommonName != client_self_signed_cert.Subject.CommonName ||
		client_self_signed_cert.Subject.CommonName != client_self_signed_cert.Issuer.CommonName {
		connection.CloseWithError(0, "issuer mismatch")
		return
	}
	if !bytes.Equal(client_tls_verify_cert.RawSubjectPublicKeyInfo, client_tls_cert.RawSubjectPublicKeyInfo) {
		connection.CloseWithError(0, "invalid TLS session")
		return
	}
	if client_tls_cert.CheckSignatureFrom(client_tls_verify_cert) != nil {
		connection.CloseWithError(0, "TLS session verification failed (1)")
		return
	}
	if client_tls_verify_cert.CheckSignatureFrom(client_self_signed_cert) != nil {
		connection.CloseWithError(0, "TLS session verification failed (2)")
		return
	}

	//authenticate ourself

	result := AbyssInbound{connection, client_self_signed_cert.Issuer.CommonName, ahmp_cbor_dec, make(chan any, 8)}
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
			//fmt.Println("receiving JN")
			var raw_msg ahmp.RawJN
			h.cbor_decoder.Decode(&raw_msg)
			parsed_msg, err := raw_msg.TryParse()
			if err != nil {
				return
			}
			h.AhmpChannel <- parsed_msg
		case ahmp.JOK_T:
			//fmt.Println("receiving JOK")
			var raw_msg ahmp.RawJOK
			h.cbor_decoder.Decode(&raw_msg)
			parsed_msg, err := raw_msg.TryParse()
			if err != nil {
				return
			}
			h.AhmpChannel <- parsed_msg
		case ahmp.JDN_T:
			//fmt.Println("receiving JDN")
			var raw_msg ahmp.RawJDN
			h.cbor_decoder.Decode(&raw_msg)
			parsed_msg, err := raw_msg.TryParse()
			if err != nil {
				return
			}
			h.AhmpChannel <- parsed_msg
		case ahmp.JNI_T:
			//fmt.Println("receiving JNI")
			var raw_msg ahmp.RawJNI
			h.cbor_decoder.Decode(&raw_msg)
			parsed_msg, err := raw_msg.TryParse()
			if err != nil {
				return
			}
			h.AhmpChannel <- parsed_msg
		case ahmp.MEM_T:
			//fmt.Println("receiving MEM")
			var raw_msg ahmp.RawMEM
			h.cbor_decoder.Decode(&raw_msg)
			parsed_msg, err := raw_msg.TryParse()
			if err != nil {
				return
			}
			h.AhmpChannel <- parsed_msg
		case ahmp.SNB_T:
			//fmt.Println("receiving SNB")
			var raw_msg ahmp.RawSNB
			h.cbor_decoder.Decode(&raw_msg)
			parsed_msg, err := raw_msg.TryParse()
			if err != nil {
				return
			}
			h.AhmpChannel <- parsed_msg
		case ahmp.CRR_T:
			//fmt.Println("receiving CRR")
			var raw_msg ahmp.RawCRR
			h.cbor_decoder.Decode(&raw_msg)
			parsed_msg, err := raw_msg.TryParse()
			if err != nil {
				return
			}
			h.AhmpChannel <- parsed_msg
		case ahmp.RST_T:
			//fmt.Println("receiving RST")
			var raw_msg ahmp.RawRST
			h.cbor_decoder.Decode(&raw_msg)
			parsed_msg, err := raw_msg.TryParse()
			if err != nil {
				return
			}
			h.AhmpChannel <- parsed_msg
		case ahmp.SOA_T:
			//fmt.Println("receiving SOA")
			var raw_msg ahmp.RawSOA
			h.cbor_decoder.Decode(&raw_msg)
			parsed_msg, err := raw_msg.TryParse()
			if err != nil {
				return
			}
			h.AhmpChannel <- parsed_msg
		case ahmp.SOD_T:
			//fmt.Println("receiving SOD")
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
