package net_service

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"math/big"
	"net"
	"strconv"
	"sync"
	"time"

	"github.com/quic-go/quic-go"
	"github.com/quic-go/quic-go/http3"

	aurl "abyss_neighbor_discovery/aurl"
	abyss "abyss_neighbor_discovery/interfaces"
)

type BetaNetService struct {
	localIdentity   abyss.ILocalIdentity
	addressSelector abyss.IAddressSelector

	quicTransport *quic.Transport
	tlsConf       *tls.Config
	quicConf      *quic.Config

	local_aurl *aurl.AURL

	preAccepter abyss.IPreAccepter

	abyssInBound  chan AbyssInbound
	abyssOutBound chan AbyssOutbound

	outbound_ongoing     map[string]*net.UDPAddr //outbound host identity -> connected IP. if connecting, nil.
	outbound_ongoing_mtx *sync.Mutex

	abyssPeerCH   chan abyss.IANDPeer
	abystServerCH chan abyss.IAbystServerPeer
}

func _getLocalIP() (string, error) {
	conn, err := net.DialUDP("udp", nil, &net.UDPAddr{
		IP:   net.IPv4(8, 8, 8, 8), // Google's public DNS as an example
		Port: 53,
	})
	if err != nil {
		return "", err
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)
	return localAddr.IP.String(), nil
}

func NewBetaNetService(local_identity abyss.ILocalIdentity, address_selector abyss.IAddressSelector) (*BetaNetService, error) {
	result := new(BetaNetService)

	result.localIdentity = local_identity
	result.addressSelector = address_selector

	udpConn, err := net.ListenUDP("udp4", &net.UDPAddr{IP: net.IPv4zero, Port: 0})
	if err != nil {
		return nil, err
	}
	result.quicTransport = &quic.Transport{Conn: udpConn}
	if result.tlsConf, err = NewDefaultTlsConf(); err != nil {
		return nil, err
	}
	result.quicConf = NewDefaultQuicConf()

	local_port := strconv.Itoa(udpConn.LocalAddr().(*net.UDPAddr).Port)
	local_ip, err := _getLocalIP()
	if err != nil {
		return nil, err
	}
	result.local_aurl, err = aurl.ParseAURL("abyss:" +
		local_identity.IDHash() +
		":" + local_ip + ":" + local_port +
		"|127.0.0.1:" + local_port)
	if err != nil {
		return nil, err
	}

	result.abyssInBound = make(chan AbyssInbound, 4)
	result.abyssOutBound = make(chan AbyssOutbound, 4)

	result.outbound_ongoing = make(map[string]*net.UDPAddr)
	result.outbound_ongoing_mtx = new(sync.Mutex)

	result.abyssPeerCH = make(chan abyss.IANDPeer, 32)
	result.abystServerCH = make(chan abyss.IAbystServerPeer, 64)

	return result, nil
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

func (h *BetaNetService) LocalAURL() *aurl.AURL {
	return h.local_aurl
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
			h.abystServerCH <- connection
		default:
			connection.CloseWithError(0, "unknown TLS ALPN protocol ID")
		}
	}
}

func (h *BetaNetService) ConstructingAbyssPeers(ctx context.Context) {
	//TODO: handle 'old' partial connections.
	abyssInParts := make(map[string]AbyssInbound)
	abyssOutParts := make(map[string]AbyssOutbound)

	for {
		select {
		case <-ctx.Done():
			return
		case inbound := <-h.abyssInBound:
			if outbound, ok := abyssOutParts[inbound.identity]; ok {
				h.abyssPeerCH <- NewAbyssPeer(h, inbound, outbound)
			} else {
				abyssInParts[inbound.identity] = inbound
			}
		case outbound := <-h.abyssOutBound:
			if inbound, ok := abyssInParts[outbound.identity.IDHash()]; ok {
				h.abyssPeerCH <- NewAbyssPeer(h, inbound, outbound)
			} else {
				abyssOutParts[outbound.identity.IDHash()] = outbound
			}
		}
	}
}

func (h *BetaNetService) ConnectAbyssAsync(ctx context.Context, url *aurl.AURL) error {
	if url.Scheme() != "abyss" {
		return errors.New("url scheme mismatch")
	}

	candidate_addresses := h.addressSelector.FilterAddressCandidates(url.Addresses())
	if len(candidate_addresses) == 0 {
		return errors.New("no valid IP address")
	}

	h.outbound_ongoing_mtx.Lock()
	if _, ok := h.outbound_ongoing[url.Hash()]; ok {
		return errors.New("redundant connect trial")
	}
	h.outbound_ongoing[url.Hash()] = nil //now raise outbound connection ongoing flag
	h.outbound_ongoing_mtx.Unlock()

	go h._connectAbyss(ctx, candidate_addresses, url.Hash())
	return nil
}

func (h *BetaNetService) _connectAbyss(ctx context.Context, addresses []*net.UDPAddr, identity string) {
	connection, err := h.quicTransport.Dial(ctx, addresses[0], h.tlsConf, h.quicConf)
	if err != nil {
		h.PrepareAbyssOutbound(ctx, nil, identity)
	}
	h.PrepareAbyssOutbound(ctx, connection, identity)
}

func (h *BetaNetService) ConnectAbyst(ctx context.Context, url *aurl.AURL) (abyss.IAbystClientPeer, error) {
	if url.Scheme() != "abyst" {
		return nil, errors.New("url scheme mismatch")
	}

	var net_addr *net.UDPAddr

	h.outbound_ongoing_mtx.Lock()
	net_addr, ok := h.outbound_ongoing[url.Hash()]
	h.outbound_ongoing_mtx.Unlock()

	if !ok {
		return nil, errors.New("no abyss connection")
	}
	connection, err := h.quicTransport.Dial(ctx, net_addr, h.tlsConf, h.quicConf)
	if err != nil {
		return nil, err
	}

	return connection, nil
}

func (h *BetaNetService) confirmPeerShutdown(peer abyss.IANDPeer) {
	h.outbound_ongoing_mtx.Lock()
	delete(h.outbound_ongoing, peer.IDHash())
	h.outbound_ongoing_mtx.Unlock()
}

func (h *BetaNetService) GetAbyssPeerChannel() chan abyss.IANDPeer {
	return h.abyssPeerCH
}
func (h *BetaNetService) GetAbystServerPeerChannel() chan abyss.IAbystServerPeer {
	return h.abystServerCH
}
