package abyss_neighbor_discovery_test

import (
	"abyss_neighbor_discovery/and"
	"abyss_neighbor_discovery/tools/dacp"

	"github.com/google/uuid"
)

type INetworkHandler interface {
	ConnectAsync(address string)
	TryAccept() (and.IANDPeer, bool)
}

type AbyssHost struct {
	globalActionPool *dacp.DiscreteActionPool

	id_hash string
	and     *and.AND

	peer_sessions map[uuid.UUID]map[uuid.UUID]and.IANDPeer //local session -> peer session -> peer

	last_event_action int
}

func NewAbyssHost(globalActionPool *dacp.DiscreteActionPool, id_hash string) *AbyssHost {
	return &AbyssHost{
		globalActionPool:  globalActionPool,
		id_hash:           id_hash,
		and:               and.NewAND(id_hash),
		peer_sessions:     make(map[uuid.UUID]map[uuid.UUID]and.IANDPeer),
		last_event_action: 0,
	}
}

func (h *AbyssHost) ReserveAllEventProcessing() {
	for len(h.and.EventCh) > 0 {
		raw_event := <-h.and.EventCh
		h.last_event_action = h.globalActionPool.AddAction(dacp.NewDiscreteAction(func() {
			switch event := raw_event.(type) {
			case and.SessionRequest:
				h.OnSessionRequest(event)
			case and.SessionReady:
				h.OnSessionReady(event)
			case and.SessionClose:
				h.OnSessionClose(event)
			case and.JoinSuccess:
				h.OnJoinSuccess(event)
			case and.JoinFail:
				h.OnJoinFail(event)
			case and.ConnectRequest:
				h.OnConnectRequest(event)
			case and.TimerRequest:
				h.OnTimerRequest(event)
			case and.DebugMessage:
				panic("[debug] " + h.id_hash + ": " + event.Message)
			default:
				panic("unknown event type")
			}
		}, h.last_event_action))
	}
}

func (h *AbyssHost) OnSessionRequest(params and.SessionRequest) {
	h.and.AcceptSession(params.LocalSessionID, params.PeerSession)
}
func (h *AbyssHost) OnSessionReady(params and.SessionReady) {
	target_world, ok := h.peer_sessions[params.LocalSessionID]
	if !ok {
		panic("AND raised OnSessionReady for non-existing session")
	}

	if _, ok = target_world[params.PeerSessionID]; ok {
		panic("duplicate OnSessionReady")
	}

	target_world[params.PeerSessionID] = params.Peer
}
func (h *AbyssHost) OnSessionClose(params and.SessionClose) {
	target_world, ok := h.peer_sessions[params.LocalSessionID]
	if !ok {
		panic("AND raised OnSessionClose for non-existing session")
	}

	if _, ok = target_world[params.PeerSessionID]; !ok {
		panic("peer not found in OnSessionClose")
	}

	delete(target_world, params.PeerSessionID)
}
func (h *AbyssHost) OnJoinSuccess(params and.JoinSuccess) {
	h.peer_sessions[params.LocalSessionID] = make(map[uuid.UUID]and.IANDPeer)
}
func (h *AbyssHost) OnJoinFail(params and.JoinFail) {
	//currently, no-op
}
func (h *AbyssHost) OnConnectRequest(params and.ConnectRequest) {

}
func (h *AbyssHost) OnTimerRequest(params and.TimerRequest) {

}

func (h *AbyssHost) GetJoinablePeers() []string {
	return nil
}

func (h *AbyssHost) TryJoin(string) {

}
