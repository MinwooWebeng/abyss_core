package test

import (
	abyss_net "abyss_neighbor_discovery/net_service"
	"context"
	"fmt"
	"testing"
)

func TestNetHost(t *testing.T) {
	local_identity := abyss_net.NewBetaLocalIdentity("mallang")
	address_selector := &abyss_net.AddressSelector{
		LocalPrivateAddr: []byte{192, 168, 56, 1},
		LocalPublicAddr:  []byte{143, 248, 49, 116},
	}
	netserv, err := abyss_net.NewBetaNetService(local_identity, address_selector)
	if err != nil {
		t.Fatal(err.Error())
	}
	url := netserv.LocalAURL()

	fmt.Println(url.ToString())

	go netserv.ListenAndServe(context.Background())

	netserv.ConnectAbyssAsync(context.Background(), url)

	peer_ch := netserv.GetAbyssPeerChannel()
	self_peer := <-peer_ch

	fmt.Println(self_peer.IDHash())
}
