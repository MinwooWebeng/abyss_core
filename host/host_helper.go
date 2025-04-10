package host

import (
	abyss_and "abyss_core/and"
	abyss_net "abyss_core/net_service"

	"github.com/quic-go/quic-go/http3"
)

func NewBetaAbyssHost(root_private_key abyss_net.PrivateKey, abyst_server *http3.Server) (*AbyssHost, *SimplePathResolver) {
	address_selector := abyss_net.NewBetaAddressSelector()
	path_resolver := NewSimplePathResolver()
	netserv, _ := abyss_net.NewBetaNetService(root_private_key, address_selector, abyst_server)

	return NewAbyssHost(netserv, abyss_and.NewAND(netserv.LocalAURL().Hash), path_resolver), path_resolver
}
