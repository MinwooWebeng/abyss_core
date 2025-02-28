package host

import (
	abyss_and "abyss_neighbor_discovery/and"
	abyss_net "abyss_neighbor_discovery/net_service"
)

func NewBetaAbyssHost(name string) (*AbyssHost, *DefaultPathResolver) {
	local_identity := abyss_net.NewBetaLocalIdentity(name)
	address_selector := abyss_net.NewBetaAddressSelector()
	path_resolver := NewDefaultPathResolver()
	netserv, _ := abyss_net.NewBetaNetService(local_identity, address_selector)

	return NewAbyssHost(netserv, abyss_and.NewAND(name), path_resolver), path_resolver
}
