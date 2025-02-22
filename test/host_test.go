package test

import (
	"context"
	"testing"

	abyss "abyss_neighbor_discovery/interfaces"

	abyss_host "abyss_neighbor_discovery/host"
	abyss_net "abyss_neighbor_discovery/net_service"
)

func TestHost(t *testing.T) {
	var local_identity abyss.ILocalIdentity
	address_selector := &abyss_net.AddressSelector{
		LocalPrivateAddr: []byte{192, 168, 56, 1},
		LocalPublicAddr:  []byte{143, 248, 49, 116},
	}
	netserv, _ := abyss_net.NewBetaNetService(local_identity, address_selector)
	abyss_host.NewAbyssNetHost(context.TODO(), netserv, nil)
}
