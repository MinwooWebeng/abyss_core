package host

import (
	"context"

	abyss_and "abyss_neighbor_discovery/and"
	abyss_net "abyss_neighbor_discovery/net_service"
)

func NewBetaAbyssNetHost(name string) *AbyssNetHost {
	local_identity := abyss_net.NewBetaLocalIdentity("mallang")
	address_selector := abyss_net.NewBetaAddressSelector()
	netserv, _ := abyss_net.NewBetaNetService(local_identity, address_selector)

	return NewAbyssNetHost(context.Background(), netserv, abyss_and.NewAND(name))
}
