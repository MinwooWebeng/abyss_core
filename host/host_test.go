package host

import (
	abyss_net "abyss_neighbor_discovery/net_service"
	"context"
	"testing"
)

func TestHost(t *testing.T) {
	netserv, _ := abyss_net.NewBetaNetService()
	NewAbyssNetHost(context.TODO(), netserv, nil)

}
