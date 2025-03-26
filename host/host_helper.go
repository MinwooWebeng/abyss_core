package host

import (
	abyss_and "abyss_neighbor_discovery/and"
	abyss_net "abyss_neighbor_discovery/net_service"
)

func NewBetaAbyssHost(root_private_key abyss_net.PrivateKey) (*AbyssHost, *SimplePathResolver) {
	address_selector := abyss_net.NewBetaAddressSelector()
	path_resolver := NewSimplePathResolver()
	netserv, _ := abyss_net.NewBetaNetService(root_private_key, address_selector)

	return NewAbyssHost(netserv, abyss_and.NewAND(netserv.LocalAURL().Hash), path_resolver), path_resolver
}
