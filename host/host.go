package host

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"sync"
	"time"

	"abyss_neighbor_discovery/aurl"
	abyss "abyss_neighbor_discovery/interfaces"
	"abyss_neighbor_discovery/tools/functional"

	"github.com/google/uuid"
	"github.com/quic-go/quic-go"
	"github.com/quic-go/quic-go/http3"
)

type AbyssHost struct {
	ctx         context.Context //set at ListenAndServe(ctx)
	listen_done chan bool
	event_done  chan bool

	networkService             abyss.INetworkService
	neighborDiscoveryAlgorithm abyss.INeighborDiscovery
	pathResolver               abyss.IPathResolver

	abystClientTr *http3.Transport

	worlds     map[uuid.UUID]*World
	worlds_mtx *sync.Mutex

	join_queue map[uuid.UUID]chan abyss.NeighborEvent //forwarding of AND join result event.
	join_q_mtx *sync.Mutex
}

func NewAbyssHost(netServ abyss.INetworkService, nda abyss.INeighborDiscovery, path_resolver abyss.IPathResolver) *AbyssHost {
	return &AbyssHost{
		listen_done:                make(chan bool, 1),
		event_done:                 make(chan bool, 1),
		networkService:             netServ,
		neighborDiscoveryAlgorithm: nda,
		pathResolver:               path_resolver,
		abystClientTr: &http3.Transport{
			Dial: func(ctx context.Context, addr string, tlsCfg *tls.Config, cfg *quic.Config) (quic.EarlyConnection, error) {
				return nil, errors.New("dialing in abyst transport is prohibited")
			},
		},
		worlds:     make(map[uuid.UUID]*World),
		worlds_mtx: new(sync.Mutex),
		join_queue: make(map[uuid.UUID]chan abyss.NeighborEvent),
		join_q_mtx: new(sync.Mutex),
	}
}

func (h *AbyssHost) GetLocalAbyssURL() *aurl.AURL {
	origin := h.networkService.LocalAURL()
	return &aurl.AURL{
		Scheme: origin.Scheme,
		Hash:   origin.Hash,
		Addresses: functional.Accum_all(
			origin.Addresses,
			make([]*net.UDPAddr, 0, len(origin.Addresses)),
			func(addr *net.UDPAddr, acc []*net.UDPAddr) []*net.UDPAddr {
				return append(acc, addr)
			},
		),
		Path: origin.Path,
	}
}

func (h *AbyssHost) OpenOutboundConnection(abyss_url *aurl.AURL) {
	h.networkService.ConnectAbyssAsync(h.ctx, abyss_url)
}

func (h *AbyssHost) OpenWorld(world_url string) (abyss.IAbyssWorld, error) {
	local_session_id := uuid.New()
	new_world := NewWorld(h.neighborDiscoveryAlgorithm, local_session_id)
	h.worlds_mtx.Lock()
	h.worlds[local_session_id] = new_world
	h.worlds_mtx.Unlock()

	retval := h.neighborDiscoveryAlgorithm.OpenWorld(local_session_id, world_url)
	if retval == abyss.EINVAL {
		h.worlds_mtx.Lock()
		delete(h.worlds, local_session_id)
		h.worlds_mtx.Unlock()
		return nil, errors.New("OpenWorld: invalid arguments")
	} else if retval == abyss.EPANIC {
		panic("fatal:::AND corrupted while opening world")
	}
	return new_world, nil
}
func (h *AbyssHost) JoinWorld(ctx context.Context, abyss_url *aurl.AURL) (abyss.IAbyssWorld, error) {
	local_session_id := uuid.New()

	retval := h.neighborDiscoveryAlgorithm.JoinWorld(local_session_id, abyss_url)
	if retval == abyss.EINVAL {
		return nil, errors.New("failed to join world::unknown error")
	} else if retval == abyss.EPANIC {
		panic("fatal:::AND corrupted while joining world")
	}

	join_res_ch := make(chan abyss.NeighborEvent)

	h.join_q_mtx.Lock()
	h.join_queue[local_session_id] = join_res_ch
	h.join_q_mtx.Unlock()

	ctx_done_waiter := make(chan bool, 1)
	go func() { //context watchdog
		select {
		case <-ctx.Done():
			h.neighborDiscoveryAlgorithm.CancelJoin(local_session_id) //this request AND module to early-return join failure
		case <-ctx_done_waiter:
			return
		}
	}()

	//wait for join result.
	join_res := <-join_res_ch

	//as join result arrived, kill the context watchdog goroutine.
	ctx_done_waiter <- true

	if join_res.Type != abyss.ANDJoinSuccess {
		return nil, errors.New(join_res.Text)
	}

	new_world := NewWorld(h.neighborDiscoveryAlgorithm, local_session_id)
	h.worlds_mtx.Lock()
	h.worlds[local_session_id] = new_world
	h.worlds_mtx.Unlock()
	return new_world, nil
}
func (h *AbyssHost) LeaveWorld(world abyss.IAbyssWorld) {
	if h.neighborDiscoveryAlgorithm.CloseWorld(world.SessionID()) != 0 {
		panic("World Leave failed")
	}
	h.worlds_mtx.Lock()
	delete(h.worlds, world.SessionID())
	h.worlds_mtx.Unlock()
}

func (h *AbyssHost) GetAbystClientConnection(ctx context.Context, peer_hash string) (*http3.ClientConn, error) {
	conn, err := h.networkService.ConnectAbyst(ctx, peer_hash)
	if err != nil {
		return nil, err
	}
	return h.abystClientTr.NewClientConn(conn), nil
}
func (h *AbyssHost) GetAbystServerPeerChannel() chan abyss.AbystInboundSession {
	return h.networkService.GetAbystServerPeerChannel()
}

func (h *AbyssHost) ListenAndServe(ctx context.Context) {
	if h.ctx != nil {
		panic("ListenAndServe called twice")
	}
	h.ctx = ctx

	net_done := make(chan bool, 1)
	go func() {
		if err := h.networkService.ListenAndServe(ctx); err != nil {
			fmt.Println(time.Now().Format("00:00:00.000") + "[network service failed] " + err.Error())
		}
		net_done <- true
	}()
	go h.listenLoop()
	go h.eventLoop()

	<-h.listen_done
	<-h.event_done

	<-net_done
}

func (h *AbyssHost) listenLoop() {
	var wg sync.WaitGroup

	accept_ch := h.networkService.GetAbyssPeerChannel()
	for {
		select {
		case <-h.ctx.Done():
			wg.Wait()
			h.listen_done <- true
			return
		case peer := <-accept_ch:
			wg.Add(1)
			go func() {
				defer wg.Done()
				h.serveLoop(peer)
			}()
		}
	}
}

func (h *AbyssHost) serveLoop(peer abyss.IANDPeer) {
	retval := h.neighborDiscoveryAlgorithm.PeerConnected(peer)
	if retval != 0 {
		return
	}

	ahmp_channel := peer.AhmpCh()
	for {
		select {
		case <-h.ctx.Done():
			return
		case message := <-ahmp_channel:
			switch t := message.Type(); t {
			case abyss.JN:
				local_session_id, ok := h.pathResolver.PathToSessionID(message.Text(), peer.IDHash())
				if !ok {
					continue // TODO: respond with proper error code
				}
				h.neighborDiscoveryAlgorithm.JN(local_session_id, abyss.ANDPeerSession{Peer: message.Sender(), PeerSessionID: message.SenderSessionID()})
			case abyss.JOK:
				h.neighborDiscoveryAlgorithm.JOK(message.RecverSessionID(), abyss.ANDPeerSession{Peer: message.Sender(), PeerSessionID: message.SenderSessionID()}, message.Text(), message.Neighbors())
			case abyss.JDN:
				h.neighborDiscoveryAlgorithm.JDN(message.RecverSessionID(), message.Sender(), message.Code(), message.Text())
			case abyss.JNI:
				joiner := message.Neighbors()[0]
				h.neighborDiscoveryAlgorithm.JNI(message.SenderSessionID(), abyss.ANDPeerSession{Peer: message.Sender(), PeerSessionID: message.SenderSessionID()}, joiner)
			case abyss.MEM:
				h.neighborDiscoveryAlgorithm.MEM(message.RecverSessionID(), abyss.ANDPeerSession{Peer: message.Sender(), PeerSessionID: message.SenderSessionID()})
			case abyss.SNB:
				h.neighborDiscoveryAlgorithm.SNB(message.RecverSessionID(), abyss.ANDPeerSession{Peer: message.Sender(), PeerSessionID: message.SenderSessionID()}, message.Texts())
			case abyss.CRR:
				h.neighborDiscoveryAlgorithm.CRR(message.RecverSessionID(), abyss.ANDPeerSession{Peer: message.Sender(), PeerSessionID: message.SenderSessionID()}, message.Texts())
			case abyss.RST:
				h.neighborDiscoveryAlgorithm.RST(message.RecverSessionID(), abyss.ANDPeerSession{Peer: message.Sender(), PeerSessionID: message.SenderSessionID()})
			}
		}
	}
}

func (h *AbyssHost) eventLoop() {
	event_ch := h.neighborDiscoveryAlgorithm.EventChannel()

	var wg sync.WaitGroup

	for {
		select {
		case <-h.ctx.Done():
			wg.Wait()
			h.event_done <- true
			return
		case e := <-event_ch:
			switch e.Type {
			case abyss.ANDSessionRequest:
				h.worlds_mtx.Lock()
				world := h.worlds[e.LocalSessionID]
				h.worlds_mtx.Unlock()

				world.RaisePeerRequest(abyss.ANDPeerSession{
					Peer:          e.Peer,
					PeerSessionID: e.PeerSessionID,
				})
			case abyss.ANDSessionReady:
				h.worlds_mtx.Lock()
				world := h.worlds[e.LocalSessionID]
				h.worlds_mtx.Unlock()

				world.RaisePeerReady(abyss.ANDPeerSession{
					Peer:          e.Peer,
					PeerSessionID: e.PeerSessionID,
				})
			case abyss.ANDSessionClose:
				h.worlds_mtx.Lock()
				world := h.worlds[e.LocalSessionID]
				h.worlds_mtx.Unlock()

				world.RaisePeerLeave(e.Peer.IDHash())
			case abyss.ANDJoinSuccess, abyss.ANDJoinFail:
				h.join_q_mtx.Lock()
				join_res_ch := h.join_queue[e.LocalSessionID]
				delete(h.join_queue, e.LocalSessionID)
				h.join_q_mtx.Unlock()

				join_res_ch <- e
			case abyss.ANDConnectRequest:
				h.networkService.ConnectAbyssAsync(h.ctx, e.Object.(*aurl.AURL))
			case abyss.ANDTimerRequest:
				target_local_session := e.LocalSessionID
				duration := e.Value
				wg.Add(1)
				go func() {
					defer wg.Done()
					select {
					case <-h.ctx.Done():
					case <-time.NewTimer(time.Duration(duration) * time.Millisecond).C:
						h.neighborDiscoveryAlgorithm.TimerExpire(target_local_session)
					}
				}()
			case abyss.ANDNeighborEventDebug:
				fmt.Println(time.Now().Format("00:00:00.000") + " " + e.Text)
			}
		}
	}
}
