package host

import (
	"context"

	abyss_and "abyss_neighbor_discovery/and"
	abyss "abyss_neighbor_discovery/interfaces"
	abyss_net "abyss_neighbor_discovery/net_service"
)

func NewBetaAbyssNetHost(name string) (*AbyssNetHost, abyss.INetworkService) {
	local_identity := abyss_net.NewBetaLocalIdentity(name)
	address_selector := abyss_net.NewBetaAddressSelector()
	netserv, _ := abyss_net.NewBetaNetService(local_identity, address_selector)

	return NewAbyssNetHost(context.Background(), netserv, abyss_and.NewAND(name)), netserv
}
