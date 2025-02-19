package net_service

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"math/big"
	"net"
	"time"

	"github.com/quic-go/quic-go"
	"github.com/quic-go/quic-go/http3"

	aurl "abyss_neighbor_discovery/aurl"
	abyss "abyss_neighbor_discovery/interfaces"
)

type BetaNetService struct {
	localIdentity abyss.ILocalIdentity
	quicTransport *quic.Transport
	tlsConf       *tls.Config
	quicConf      *quic.Config

	preAccepter abyss.IPreAccepter

	abyssInBound  chan AbyssInbound
	abyssOutBound chan AbyssOutbound

	abyssPeerCH   chan abyss.IANDPeer
	abystClientCH chan abyss.IAbystClientPeer
	abystServerCH chan abyss.IAbystServerPeer
}

func NewBetaNetService() (*BetaNetService, bool) {
	result := new(BetaNetService)

	udpConn, err := net.ListenUDP("udp4", &net.UDPAddr{IP: net.IPv4zero, Port: 0})
	if err != nil {
		return nil, false
	}
	result.quicTransport = &quic.Transport{Conn: udpConn}
	if result.tlsConf, err = NewDefaultTlsConf(); err != nil {
		return nil, false
	}
	result.quicConf = NewDefaultQuicConf()

	result.abyssInBound = make(chan AbyssInbound, 4)
	result.abyssOutBound = make(chan AbyssOutbound, 4)

	result.abyssPeerCH = make(chan abyss.IANDPeer, 32)
	result.abystClientCH = make(chan abyss.IAbystClientPeer, 64)
	result.abystServerCH = make(chan abyss.IAbystServerPeer, 64)

	return result, true
}

func NewDefaultTlsConf() (*tls.Config, error) {
	ed25519_public_key, ed25519_private_key, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, err
	}

	template := x509.Certificate{
		SerialNumber:          big.NewInt(0),
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}
	derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, ed25519_public_key, ed25519_private_key)
	if err != nil {
		return nil, err
	}
	result := &tls.Config{
		Certificates: []tls.Certificate{
			{
				Certificate: [][]byte{derBytes},
				PrivateKey:  ed25519_private_key,
			},
		},
		NextProtos:         []string{abyss.NextProtoAbyss, http3.NextProtoH3},
		InsecureSkipVerify: true,
	}
	return result, nil
}

func NewDefaultQuicConf() *quic.Config {
	return &quic.Config{
		MaxIdleTimeout:                time.Minute * 10,
		AllowConnectionWindowIncrease: func(conn quic.Connection, delta uint64) bool { return true },
		KeepAlivePeriod:               time.Minute * 3,
		Allow0RTT:                     true,
		EnableDatagrams:               true,
	}
}

func (h *BetaNetService) HandlePreAccept(preaccept_handler abyss.IPreAccepter) {
	h.preAccepter = preaccept_handler
}

func (h *BetaNetService) ListenAndServe(ctx context.Context) error {
	listener, err := h.quicTransport.Listen(h.tlsConf, h.quicConf)
	if err != nil {
		return err
	}
	go h.ConstructingAbyssPeers(ctx)

	for {
		connection, err := listener.Accept(ctx)
		if err != nil {
			return err
		}
		switch connection.ConnectionState().TLS.NegotiatedProtocol {
		case abyss.NextProtoAbyss:
			go h.PrepareAbyssInbound(ctx, connection)
		case http3.NextProtoH3:
			h.abystClientCH <- connection
		default:
			connection.CloseWithError(0, "unknown TLS ALPN protocol ID")
		}
	}
}

func (h *BetaNetService) ConstructingAbyssPeers(ctx context.Context) {
	abyssInParts := make(map[string]AbyssInbound)
	abyssOutParts := make(map[string]AbyssOutbound)

	for {
		select {
		case <-ctx.Done():
			return
		case inbound := <-h.abyssInBound:
			if outbound, ok := abyssOutParts[inbound.identity]; ok {
				h.abyssPeerCH <- NewAbyssPeer(inbound, outbound)
			} else {
				abyssInParts[inbound.identity] = inbound
			}
		case outbound := <-h.abyssOutBound:
			if inbound, ok := abyssInParts[outbound.identity]; ok {
				h.abyssPeerCH <- NewAbyssPeer(inbound, outbound)
			} else {
				abyssOutParts[outbound.identity] = outbound
			}
		}
	}
}

func (h *BetaNetService) ConnectAsync(address string) error {
	parsed_url, err := aurl.ParseAURL(address)
	if err != nil {
		return err
	}

	candidate_addresses := parsed_url.Addresses()
	//TODO

	return nil
}

func (h *BetaNetService) GetAbyssPeerChannel() chan abyss.IANDPeer {
	return h.abyssPeerCH
}
func (h *BetaNetService) GetAbystClientPeerChannel() chan abyss.IAbystClientPeer {
	return h.abystClientCH
}
func (h *BetaNetService) GetAbystServerPeerChannel() chan abyss.IAbystServerPeer {
	return h.abystServerCH
}
func (h *BetaNetService) GetConnectionFailChannel() chan abyss.ConnectionFailInfo {
	return nil
}
