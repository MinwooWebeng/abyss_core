package host

import (
	"context"
	"fmt"
	"sync"
	"time"

	"abyss_neighbor_discovery/aurl"
	abyss "abyss_neighbor_discovery/interfaces"

	"github.com/google/uuid"
)

type AbyssNetHost struct {
	ctx         context.Context //set at ListenAndServe(ctx)
	listen_done chan bool
	event_done  chan bool

	networkService             abyss.INetworkService
	neighborDiscoveryAlgorithm abyss.INeighborDiscovery
	pathResolver               abyss.IPathResolver

	worlds map[uuid.UUID]
}

func NewAbyssNetHost(netServ abyss.INetworkService, nda abyss.INeighborDiscovery, path_resolver abyss.IPathResolver) *AbyssNetHost {
	return &AbyssNetHost{
		listen_done:                make(chan bool, 1),
		event_done:                 make(chan bool, 1),
		networkService:             netServ,
		neighborDiscoveryAlgorithm: nda,
		pathResolver:               path_resolver,
	}
}

func (h *AbyssNetHost) OpenWorld(local_session_id uuid.UUID, world_url string) (chan abyss.IWorldEvent, error) {
	retval := h.neighborDiscoveryAlgorithm.OpenWorld(local_session_id, world_url)

	if retval == abyss.EINVAL {
		return false
	} else if retval == abyss.EPANIC {
		panic("fatal:::AND corrupted while opening world")
	}
	return true
}
func (h *AbyssNetHost) JoinWorld(ctx context.Context, local_session_id uuid.UUID, url string) (chan abyss.IWorldEvent, error) {

}
func (h *AbyssNetHost) joinPeerWorld(local_session_id uuid.UUID, peer abyss.IANDPeer, path string) bool {
	retval := h.neighborDiscoveryAlgorithm.JoinWorld(local_session_id, peer, path)

	if retval == abyss.EINVAL {
		return false
	} else if retval == abyss.EPANIC {
		panic("fatal:::AND corrupted while joining world")
	}
	return true
}
func (h *AbyssNetHost) cancelJoin(local_session_id uuid.UUID) bool {
	retval := h.neighborDiscoveryAlgorithm.CancelJoin(local_session_id)

	if retval == abyss.EINVAL {
		return false
	} else if retval == abyss.EPANIC {
		panic("fatal:::AND corrupted while canceling join")
	}
	return true
}

func (h *AbyssNetHost) ListenAndServe(ctx context.Context) {
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

func (h *AbyssNetHost) listenLoop() {
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

func (h *AbyssNetHost) serveLoop(peer abyss.IANDPeer) {
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
				h.neighborDiscoveryAlgorithm.JN(local_session_id, abyss.PeerSession{Peer: message.Sender(), PeerSessionID: message.SenderSessionID()})
			case abyss.JOK:
				h.neighborDiscoveryAlgorithm.JOK(message.RecverSessionID(), abyss.PeerSession{Peer: message.Sender(), PeerSessionID: message.SenderSessionID()}, message.Text(), message.Neighbors())
			case abyss.JDN:
				h.neighborDiscoveryAlgorithm.JDN(message.RecverSessionID(), message.Sender(), message.Code(), message.Text())
			case abyss.JNI:
				joiner := message.Neighbors()[0]
				h.neighborDiscoveryAlgorithm.JNI(message.SenderSessionID(), abyss.PeerSession{Peer: message.Sender(), PeerSessionID: message.SenderSessionID()}, joiner)
			case abyss.MEM:
				h.neighborDiscoveryAlgorithm.MEM(message.RecverSessionID(), abyss.PeerSession{Peer: message.Sender(), PeerSessionID: message.SenderSessionID()})
			case abyss.SNB:
				h.neighborDiscoveryAlgorithm.SNB(message.RecverSessionID(), abyss.PeerSession{Peer: message.Sender(), PeerSessionID: message.SenderSessionID()}, message.Texts())
			case abyss.CRR:
				h.neighborDiscoveryAlgorithm.CRR(message.RecverSessionID(), abyss.PeerSession{Peer: message.Sender(), PeerSessionID: message.SenderSessionID()}, message.Texts())
			case abyss.RST:
				h.neighborDiscoveryAlgorithm.RST(message.RecverSessionID(), abyss.PeerSession{Peer: message.Sender(), PeerSessionID: message.SenderSessionID()})
			}
		}
	}
}

func (h *AbyssNetHost) eventLoop() {
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
				h.handlerMtx.Lock()
				ok, code, message := h.sessionRequestHandler.OnSessionRequest(e.LocalSessionID, e.Peer, e.PeerSessionID)
				h.handlerMtx.Unlock()
				if ok {
					h.neighborDiscoveryAlgorithm.AcceptSession(e.LocalSessionID, abyss.PeerSession{Peer: e.Peer, PeerSessionID: e.PeerSessionID})
				} else {
					h.neighborDiscoveryAlgorithm.DeclineSession(e.LocalSessionID, abyss.PeerSession{Peer: e.Peer, PeerSessionID: e.PeerSessionID}, code, message)
				}
			case abyss.ANDSessionReady:
				h.handlerMtx.Lock()
				h.sessionReadyHandler.OnSessionReady(e.LocalSessionID, e.Peer, e.PeerSessionID)
				h.handlerMtx.Unlock()
			case abyss.ANDSessionClose:
				h.handlerMtx.Lock()
				h.sessionCloseHandler.OnSessionClose(e.LocalSessionID, e.Peer, e.PeerSessionID)
				h.handlerMtx.Unlock()
			case abyss.ANDJoinSuccess:
				h.handlerMtx.Lock()
				h.joinSuccessHandler.OnJoinSuccess(e.LocalSessionID, e.Text)
				h.handlerMtx.Unlock()
			case abyss.ANDJoinFail:
				h.handlerMtx.Lock()
				h.joinFailHandler.OnJoinFail(e.LocalSessionID, e.Value, e.Text)
				h.handlerMtx.Unlock()
			case abyss.ANDConnectRequest:
				url, err := aurl.ParseAURL(e.Text)
				if err != nil {
					h.networkService.ConnectAbyssAsync(h.ctx, url)
				}
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
