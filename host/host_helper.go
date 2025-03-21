package host

import (
	abyss_and "abyss_neighbor_discovery/and"
	abyss_net "abyss_neighbor_discovery/net_service"
	"crypto"
)

func NewBetaAbyssHost(root_private_key crypto.PrivateKey) (*AbyssHost, *DefaultPathResolver) {
	address_selector := abyss_net.NewBetaAddressSelector()
	path_resolver := NewDefaultPathResolver()
	netserv, _ := abyss_net.NewBetaNetService(root_private_key, address_selector)

	return NewAbyssHost(netserv, abyss_and.NewAND(netserv.LocalAURL().Hash), path_resolver), path_resolver
}
