package net_service

import (
	"crypto/tls"

	"github.com/quic-go/quic-go"

	abyss "abyss_neighbor_discovery/interfaces"
)

type BetaNetService struct {
	LocalIdentity abyss.ILocalIdentity
	QuicTransport *quic.Transport
	TlsConf       *tls.Config
	QuicConf      *quic.Config
}

func NewHost() *BetaNetService {
	return nil
}
