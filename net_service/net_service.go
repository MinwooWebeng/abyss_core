package net_service

import (
	"context"
	"crypto/tls"
	"encoding/pem"
	"errors"
	"net"
	"strconv"
	"sync"
	"time"

	"github.com/quic-go/quic-go"
	"github.com/quic-go/quic-go/http3"

	"github.com/MinwooWebeng/abyss_core/aurl"
	abyss "github.com/MinwooWebeng/abyss_core/interfaces"
	"github.com/MinwooWebeng/abyss_core/watchdog"
)

type BetaNetService struct {
	localIdentity   *RootSecrets
	addressSelector abyss.IAddressSelector

	quicTransport *quic.Transport
	tlsIdentity   *TLSIdentity
	abyssTlsConf  *tls.Config
	quicConf      *quic.Config

	local_aurl *aurl.AURL

	preAccepter abyss.IPreAccepter

	known_peers     map[string]*PeerIdentity
	known_peers_mtx *sync.Mutex

	abyssInBound  chan AbyssInbound
	abyssOutBound chan AbyssOutbound

	outbound_ongoing     map[string]*net.UDPAddr //outbound host identity -> connected IP. if connecting, nil.
	outbound_ongoing_mtx *sync.Mutex

	abyssPeerCH chan abyss.IANDPeer

	abystTlsConf *tls.Config
	abystServer  *http3.Server
}

// func _getLocalIP() (string, error) {
// 	conn, err := net.DialUDP("udp", nil, &net.UDPAddr{
// 		IP:   net.IPv4(8, 8, 8, 8), // Google's public DNS as an example
// 		Port: 53,
// 	})
// 	if err != nil {
// 		return "", err
// 	}
// 	defer conn.Close()

// 	localAddr := conn.LocalAddr().(*net.UDPAddr)
// 	return localAddr.IP.String(), nil
// }

func NewBetaNetService(local_private_key PrivateKey, address_selector abyss.IAddressSelector, abyst_server *http3.Server) (*BetaNetService, error) {
	result := new(BetaNetService)

	root_secret, err := NewRootIdentity(local_private_key)
	if err != nil {
		return nil, err
	}

	result.localIdentity = root_secret
	result.addressSelector = address_selector

	tls_identity, err := root_secret.NewTLSIdentity()
	result.tlsIdentity = tls_identity
	if err != nil {
		return nil, err
	}
	result.abyssTlsConf = NewDefaultTlsConf(tls_identity)

	udpConn, err := net.ListenUDP("udp4", &net.UDPAddr{IP: net.IPv4zero, Port: 0})
	if err != nil {
		return nil, err
	}
	result.quicTransport = &quic.Transport{Conn: udpConn}
	result.quicConf = NewDefaultQuicConf()

	local_port := strconv.Itoa(udpConn.LocalAddr().(*net.UDPAddr).Port)
	local_ip := address_selector.LocalPrivateIPAddr().String()
	result.local_aurl, err = aurl.TryParse("abyss:" +
		root_secret.IDHash() +
		":" + local_ip + ":" + local_port +
		"|127.0.0.1:" + local_port)
	if err != nil {
		return nil, err
	}

	result.known_peers = make(map[string]*PeerIdentity)
	result.known_peers_mtx = new(sync.Mutex)

	result.abyssInBound = make(chan AbyssInbound, 4)
	result.abyssOutBound = make(chan AbyssOutbound, 4)

	result.outbound_ongoing = make(map[string]*net.UDPAddr)
	result.outbound_ongoing_mtx = new(sync.Mutex)

	result.abyssPeerCH = make(chan abyss.IANDPeer, 8)

	result.abystTlsConf = NewDefaultTlsConf(tls_identity)
	result.abystTlsConf.NextProtos = []string{http3.NextProtoH3} //abyst only.
	result.abystServer = abyst_server

	return result, nil
}

func NewDefaultTlsConf(tls_identity *TLSIdentity) *tls.Config {
	return &tls.Config{
		Certificates: []tls.Certificate{
			{
				Certificate: [][]byte{tls_identity.tls_self_cert},
				PrivateKey:  tls_identity.priv_key,
			},
		},
		VerifyConnection: func(cs tls.ConnectionState) error {
			if len(cs.PeerCertificates) > 1 {
				return errors.New("too many TLS peer certificate")
			}
			cert := cs.PeerCertificates[0]
			if err := cert.CheckSignatureFrom(cert); err != nil {
				return errors.Join(errors.New("TLS Verify Failed"), err)
			}
			return nil
		},
		NextProtos:         []string{abyss.NextProtoAbyss, http3.NextProtoH3},
		ServerName:         "abyss",
		ClientAuth:         tls.RequireAnyClientCert,
		InsecureSkipVerify: true,
	}
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

func (h *BetaNetService) LocalIdentity() abyss.IHostIdentity {
	return h.localIdentity
}
func (h *BetaNetService) LocalAURL() *aurl.AURL {
	return h.local_aurl
}

func (h *BetaNetService) HandlePreAccept(preaccept_handler abyss.IPreAccepter) {
	h.preAccepter = preaccept_handler
}

func (h *BetaNetService) ListenAndServe(ctx context.Context) error {
	listener, err := h.quicTransport.Listen(h.abyssTlsConf, h.quicConf)
	if err != nil {
		return err
	}
	go h.constructingAbyssPeers(ctx)

	for {
		connection, err := listener.Accept(ctx)
		if err != nil {
			return err
		}
		switch connection.ConnectionState().TLS.NegotiatedProtocol {
		case abyss.NextProtoAbyss:
			go h.PrepareAbyssInbound(ctx, connection)
		case http3.NextProtoH3:
			go h.PrepareAbystInbound(ctx, connection)
		default:
			connection.CloseWithError(0, "unknown TLS ALPN protocol ID")
		}
	}
}

func (h *BetaNetService) constructingAbyssPeers(ctx context.Context) {
	//TODO: handle 'old' partial connections.
	abyssInParts := make(map[string]AbyssInbound)
	abyssOutParts := make(map[string]AbyssOutbound)

	for {
		select {
		case <-ctx.Done():
			return
		case inbound := <-h.abyssInBound:
			if inbound.err != nil {
				watchdog.Error(inbound.err)
				//fmt.Println(h.localIdentity.root_id_hash + "(inbound): " + inbound.err.Error())
				//TODO: handle faulty inbound connection.
				//vulnerable
				if inbound.connection != nil {
					inbound.connection.CloseWithError(0, "abyss handshake failed (inbound)")
				}
				continue
			}
			//watchdog.Info("inbound established")

			if outbound, ok := abyssOutParts[inbound.peer_hash]; ok {
				h.NotifyEstablishedPeerAddress(inbound.peer_hash, outbound.connection.RemoteAddr().(*net.UDPAddr))
				h.abyssPeerCH <- NewAbyssPeer(h, inbound, outbound)
			} else {
				abyssInParts[inbound.peer_hash] = inbound
			}
		case outbound := <-h.abyssOutBound:
			if outbound.err != nil {
				watchdog.Error(outbound.err)
				//fmt.Println(h.localIdentity.root_id_hash + "(outbound): " + outbound.err.Error())
				//TODO: handle connection failure.
				//vulnerable
				if outbound.connection != nil {
					outbound.connection.CloseWithError(0, "abyss handshake failed (outbound)")
				}
				continue
			}
			//watchdog.Info("outbound established")

			if inbound, ok := abyssInParts[outbound.peer_identity.root_id_hash]; ok {
				h.NotifyEstablishedPeerAddress(outbound.peer_identity.root_id_hash, outbound.connection.RemoteAddr().(*net.UDPAddr))
				h.abyssPeerCH <- NewAbyssPeer(h, inbound, outbound)
			} else {
				abyssOutParts[outbound.peer_identity.root_id_hash] = outbound
			}
		}
	}
}

func (h *BetaNetService) AppendKnownPeer(root_cert string, handshake_key_cert string) error {
	root_cert_block, _ := pem.Decode([]byte(root_cert))
	handshake_key_cert_block, _ := pem.Decode([]byte(handshake_key_cert))
	if root_cert_block == nil || handshake_key_cert_block == nil {
		return errors.New("failed to parse peer certificates")
	}

	return h.AppendKnownPeerDer(root_cert_block.Bytes, handshake_key_cert_block.Bytes)
}
func (h *BetaNetService) AppendKnownPeerDer(root_cert []byte, handshake_key_cert []byte) error {
	peer_identity, err := NewPeerIdentity(root_cert, handshake_key_cert)
	if err != nil {
		return err
	}

	h.known_peers_mtx.Lock()
	defer h.known_peers_mtx.Unlock()

	h.known_peers[peer_identity.root_id_hash] = peer_identity
	return nil
}
func (h *BetaNetService) RemoveKnownPeer(peer_hash string) {
	h.known_peers_mtx.Lock()
	defer h.known_peers_mtx.Unlock()

	delete(h.known_peers, peer_hash)
}
func (h *BetaNetService) findKnownPeer(peer_hash string) (*PeerIdentity, bool) {
	h.known_peers_mtx.Lock()
	defer h.known_peers_mtx.Unlock()

	res, ok := h.known_peers[peer_hash]
	return res, ok
}

// func (h *BetaNetService) waitForPeerIdentity(ctx context.Context, peer_hash string) (*PeerIdentity, error) {
// 	select {
// 	case <-ctx.Done():
// 		return nil, ctx.Err()
// 	}
// }

func (h *BetaNetService) ConnectAbyssAsync(ctx context.Context, url *aurl.AURL) error {
	if url.Scheme != "abyss" {
		return errors.New("url scheme mismatch")
	}

	candidate_addresses := h.addressSelector.FilterAddressCandidates(url.Addresses)
	if len(candidate_addresses) == 0 {
		return errors.New("no valid IP address")
	}

	h.outbound_ongoing_mtx.Lock()
	defer h.outbound_ongoing_mtx.Unlock()

	if _, ok := h.outbound_ongoing[url.Hash]; ok {
		return nil
	}
	h.outbound_ongoing[url.Hash] = nil //now raise outbound connection ongoing flag

	go h._connectAbyss(ctx, candidate_addresses, url.Hash)
	return nil
}

func (h *BetaNetService) _connectAbyss(ctx context.Context, addresses []*net.UDPAddr, peer_hash string) {
	connection, err := h.quicTransport.Dial(ctx, addresses[0], h.abyssTlsConf, h.quicConf)
	if err != nil {
		h.abyssOutBound <- AbyssOutbound{nil, nil, nil, nil, err}
		return
	}
	h.PrepareAbyssOutbound(ctx, connection, peer_hash, addresses)
}

func (h *BetaNetService) GetAbyssPeerChannel() chan abyss.IANDPeer {
	return h.abyssPeerCH
}

func (h *BetaNetService) NotifyEstablishedPeerAddress(peer_hash string, addr *net.UDPAddr) {
	h.outbound_ongoing_mtx.Lock()
	defer h.outbound_ongoing_mtx.Unlock()

	if addr, ok := h.outbound_ongoing[peer_hash]; !(ok && addr == nil) {
		panic("just connected peer now missing or already notified")
	}
	h.outbound_ongoing[peer_hash] = addr
}

func (h *BetaNetService) CloseAbyssPeer(peer abyss.IANDPeer) {
	h.outbound_ongoing_mtx.Lock()
	defer h.outbound_ongoing_mtx.Unlock()

	delete(h.outbound_ongoing, peer.IDHash())
}

func (h *BetaNetService) ConnectAbyst(ctx context.Context, peer_hash string) (quic.Connection, error) {
	var net_addr *net.UDPAddr

	if peer_hash == h.localIdentity.root_id_hash { //loopback
		connection, err := h.quicTransport.Dial(ctx, h.local_aurl.Addresses[len(h.local_aurl.Addresses)-1], h.abystTlsConf, h.quicConf)
		if err != nil {
			return nil, err
		}

		return connection, nil
	}

	h.outbound_ongoing_mtx.Lock()
	net_addr, ok := h.outbound_ongoing[peer_hash]
	h.outbound_ongoing_mtx.Unlock()

	if !ok {
		return nil, errors.New("no abyss connection")
	}
	if net_addr == nil {
		return nil, errors.New("abyss connect pending")
	}
	connection, err := h.quicTransport.Dial(ctx, net_addr, h.abystTlsConf, h.quicConf)
	if err != nil {
		return nil, err
	}

	return connection, nil
}
