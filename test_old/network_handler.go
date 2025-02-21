package abyss_neighbor_discovery_test

import (
	"abyss_neighbor_discovery/and"
	"abyss_neighbor_discovery/tools/dacp"
)

type TestNetworkHandler struct {
	globalActionPool *dacp.DiscreteActionPool

	connectionState map[string]int // 0: disconnected, 1: requested, 2: accepted
}

func NewTestNetworkHandler(globalActionPool *dacp.DiscreteActionPool, all_ids []string) *TestNetworkHandler {
	result := &TestNetworkHandler{
		globalActionPool: globalActionPool,
		connectionState:  make(map[string]int),
	}
	for _, id := range all_ids {
		for _, ido := range all_ids {
			if id == ido {
				continue
			}
			result.connectionState[id+">"+ido] = 0
		}
	}
	return result
}

type TestHostNH struct {
	handler  *TestNetworkHandler
	local_id string
}

func NewTestHostNH(handler *TestNetworkHandler, local_id string) *TestHostNH {
	return &TestHostNH{
		handler:  handler,
		local_id: local_id,
	}
}

func (n *TestHostNH) ConnectAsync(address string) {
	state, ok := n.handler.connectionState[n.local_id+">"+address]
	if !ok {
		panic("unknown peers!")
	}
	switch state {
	case 0:
		n.handler.connectionState[n.local_id+">"+address] = 1

	}
}

func (n *TestHostNH) TryAccept() (and.IANDPeer, bool) {
	return nil, false
}
