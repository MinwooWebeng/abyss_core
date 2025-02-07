package idea

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"net"
	"sync"
)

type AURL interface {
	ToString() string
	Protocol() string
	Hash() string
	Addresses() []net.UDPAddr
	Path() string // does not include starting "/".
}

type PeerSession struct {
	Peer          IANDPeer
	PeerSessionID uuid.UUID
}

type PeerSessionInfo struct {
	AURL      AURL
	SessionID uuid.UUID
}

type AhmpMessageType int

const (
	JN AhmpMessageType = iota + 2
	JOK
	JDN
	JNI
	MEM
	SNB
	CRR
	RST
)

type IAhmpMessage interface {
	Type() AhmpMessageType
	Sender() IANDPeer
	SenderSessionID() uuid.UUID
	RecverSessionID() uuid.UUID
	Neighbors() []PeerSessionInfo
	Texts() []string
	Text() string
	Code() int
}

type IRemoteIdentity interface {
	IDHash() string
	ValidateSignature(payload []byte, hash []byte) bool
}

type IANDPeer interface {
	IRemoteIdentity

	AhmpCh() chan IAhmpMessage

	TrySendJN(local_session_id uuid.UUID, path string) bool
	TrySendJOK(peer_session_id uuid.UUID, local_session_id uuid.UUID, world_url string, member_sessions []PeerSession) bool
	TrySendJDN(peer_session_id uuid.UUID, code int, message string) bool
	TrySendJNI(peer_session_id uuid.UUID, member_session PeerSession) bool
	TrySendMEM(peer_session_id uuid.UUID, local_session_id uuid.UUID) bool
	TrySendSNB(peer_session_id uuid.UUID, member_sessions []PeerSessionInfo) bool
	TrySendCRR(peer_session_id uuid.UUID, member_sessions []PeerSessionInfo) bool
	TrySendRST(peer_session_id uuid.UUID, local_session_id uuid.UUID) bool
}

type INetworkService interface {
	ReservePreAccept(func() (bool, int, string)) // if false, return status code and message

	ListenAndServe(ctx context.Context)

	ConnectAsync(address string)
	TryAccept() (IANDPeer, bool)
}

type NeighborEventType int

const (
	SessionRequest NeighborEventType = iota + 2
	SessionReady
	SessionClose
	JoinSuccess
	JoinFail
	ConnectRequest
	TimerRequest
	NeighborEventDebug
)

type NeighborEvent struct {
	Type           NeighborEventType
	LocalSessionID uuid.UUID
	Peer           IANDPeer
	PeerSessionID  uuid.UUID
	Text           string
	Value          int
}

type ANDERROR int

const (
	_      ANDERROR = iota //no error
	EINVAL                 //invalid argument
	EPANIC                 //unrecoverable internal error (must not occur)
)

type INeighborDiscovery interface {
	EventChannel() chan NeighborEvent
	ResetPeerSession(local_session_id uuid.UUID, peer IANDPeer, peer_session_id uuid.UUID)

	//calls
	PeerConnected(peer IANDPeer) ANDERROR
	PeerClose(peer IANDPeer) ANDERROR
	OpenWorld(local_session_id uuid.UUID, world_url string) ANDERROR
	JoinWorld(local_session_id uuid.UUID, peer IANDPeer, path string) ANDERROR
	CancelJoin(local_session_id uuid.UUID) ANDERROR
	AcceptSession(local_session_id uuid.UUID, peer_session PeerSession) ANDERROR
	DeclineSession(local_session_id uuid.UUID, peer_session PeerSession, code int, message string) ANDERROR
	CloseWorld(local_session_id uuid.UUID) ANDERROR
	TimerExpire(local_session_id uuid.UUID) ANDERROR

	//ahmp messages
	JN(local_session_id uuid.UUID, peer_session PeerSession) ANDERROR
	JOK(local_session_id uuid.UUID, peer_session PeerSession, world_url string, member_sessions []PeerSessionInfo) ANDERROR
	JDN(local_session_id uuid.UUID, peer IANDPeer, code int, message string) ANDERROR
	JNI(local_session_id uuid.UUID, peer_session PeerSession, member_session PeerSessionInfo) ANDERROR
	MEM(local_session_id uuid.UUID, peer_session PeerSession) ANDERROR
	SNB(local_session_id uuid.UUID, peer_session PeerSession, member_hashes []string) ANDERROR
	CRR(local_session_id uuid.UUID, peer_session PeerSession, member_hashes []string) ANDERROR
	RST(local_session_id uuid.UUID, peer_session PeerSession) ANDERROR
}

type ILocalIdentity interface {
	IDHash() string
	Sign(payload []byte) []byte
}

type AbyssNetHost struct {
	ctx context.Context
	//localIdentity              ILocalIdentity
	networkService             INetworkService
	neighborDiscoveryAlgorithm INeighborDiscovery
	neighborDiscoveryMtx       *sync.Mutex

	pathResolver          IPathResolver
	sessionRequestHandler ISessionRequestHandler
	sessionReadyHandler   ISessionReadyHandler
	sessionCloseHandler   ISessionCloseHandler
	joinSuccessHandler    IJoinSuccessHandler
	joinFailHandler       IJoinFailHandler
}

type IPathResolver interface {
	PathToSessionID(path string) (uuid.UUID, bool)
}
type ISessionRequestHandler interface {
	OnSessionRequest(local_session_id uuid.UUID, peer IANDPeer, peer_session_id uuid.UUID) (ok bool, code int, message string)
}
type ISessionReadyHandler interface {
	OnSessionReady(local_session_id uuid.UUID, peer IANDPeer, peer_session_id uuid.UUID)
}
type ISessionCloseHandler interface {
	OnSessionClose(local_session_id uuid.UUID, peer IANDPeer, peer_session_id uuid.UUID)
}
type IJoinSuccessHandler interface {
	OnJoinSuccess(local_session_id uuid.UUID, world_url string)
}
type IJoinFailHandler interface {
	OnJoinFail(local_session_id uuid.UUID, code int, message string)
}

func (h *AbyssNetHost) HandlePathResolve(handler IPathResolver) {
	h.pathResolver = handler
}
func (h *AbyssNetHost) HandleSessionRequest(handler ISessionRequestHandler) {
	h.sessionRequestHandler = handler
}
func (h *AbyssNetHost) HandleSessionReady(handler ISessionReadyHandler) {
	h.sessionReadyHandler = handler
}
func (h *AbyssNetHost) HandleSessionClose(handler ISessionCloseHandler) {
	h.sessionCloseHandler = handler
}
func (h *AbyssNetHost) HandleJoinSuccess(handler IJoinSuccessHandler) {
	h.joinSuccessHandler = handler
}
func (h *AbyssNetHost) HandleJoinFail(handler IJoinFailHandler) {
	h.joinFailHandler = handler
}

// ConnectRequest
// TimerRequest
// NeighborEventDebug

func (h *AbyssNetHost) ListenAndServe() {
	h.networkService.ListenAndServe(h.ctx)

	go h.listenLoop()
	go h.eventLoop()
}

func (h *AbyssNetHost) listenLoop() {
	for h.ctx.Err() == nil {
		peer, ok := h.networkService.TryAccept()
		if !ok {
			return
		}

		go h.serveLoop(peer)
	}
}

func (h *AbyssNetHost) serveLoop(peer IANDPeer) {
	var retval ANDERROR

	h.neighborDiscoveryMtx.Lock()
	retval = h.neighborDiscoveryAlgorithm.PeerConnected(peer)
	h.neighborDiscoveryMtx.Unlock()
	if retval != 0 {
		return
	}

	ahmp_channel := peer.AhmpCh()
	for {
		select {
		case <-h.ctx.Done():
			return
		case message := <-ahmp_channel:
			h.neighborDiscoveryMtx.Lock()
			switch t := message.Type(); t {
			case JN:
				local_session_id, ok := h.pathResolver.PathToSessionID(message.Text())
				if !ok {
					continue
				}
				h.neighborDiscoveryAlgorithm.JN(local_session_id, PeerSession{message.Sender(), message.SenderSessionID()})
			case JOK:
				h.neighborDiscoveryAlgorithm.JOK(message.RecverSessionID(), PeerSession{message.Sender(), message.SenderSessionID()}, message.Text(), message.Neighbors())
			case JDN:
				h.neighborDiscoveryAlgorithm.JDN(message.RecverSessionID(), message.Sender(), message.Code(), message.Text())
			case JNI:
				joiner := message.Neighbors()[0]
				h.neighborDiscoveryAlgorithm.JNI(message.SenderSessionID(), PeerSession{message.Sender(), message.SenderSessionID()}, joiner)
			case MEM:
				h.neighborDiscoveryAlgorithm.MEM(message.RecverSessionID(), PeerSession{message.Sender(), message.SenderSessionID()})
			case SNB:
				h.neighborDiscoveryAlgorithm.SNB(message.RecverSessionID(), PeerSession{message.Sender(), message.SenderSessionID()}, message.Texts())
			case CRR:
				h.neighborDiscoveryAlgorithm.CRR(message.RecverSessionID(), PeerSession{message.Sender(), message.SenderSessionID()}, message.Texts())
			case RST:
				h.neighborDiscoveryAlgorithm.RST(message.RecverSessionID(), PeerSession{message.Sender(), message.SenderSessionID()})
			}
			h.neighborDiscoveryMtx.Unlock()
		}
	}
}

func (h *AbyssNetHost) eventLoop() {
	event_ch := h.neighborDiscoveryAlgorithm.EventChannel()

	for {
		select {
		case <-h.ctx.Done():
			return
		case e := <-event_ch:
			switch e.Type {
			case SessionRequest:
				ok, code, message := h.sessionRequestHandler.OnSessionRequest(e.LocalSessionID, e.Peer, e.PeerSessionID)
				h.neighborDiscoveryMtx.Lock()
				if ok {
					h.neighborDiscoveryAlgorithm.AcceptSession(e.LocalSessionID, PeerSession{e.Peer, e.PeerSessionID})
				} else {
					h.neighborDiscoveryAlgorithm.DeclineSession(e.LocalSessionID, PeerSession{e.Peer, e.PeerSessionID}, code, message)
				}
				h.neighborDiscoveryMtx.Unlock()
			case SessionReady:
				h.sessionReadyHandler.OnSessionReady(e.LocalSessionID, e.Peer, e.PeerSessionID)
			case SessionClose:
				h.sessionCloseHandler.OnSessionClose(e.LocalSessionID, e.Peer, e.PeerSessionID)
			case JoinSuccess:
				h.joinSuccessHandler.OnJoinSuccess(e.LocalSessionID, e.Text)
			case JoinFail:
				h.joinFailHandler.OnJoinFail(e.LocalSessionID, e.Value, e.Text)
			case ConnectRequest:
				h.networkService.ConnectAsync(e.Text)
			case TimerRequest:
				target_local_session := e.LocalSessionID
				duration := e.Value
				go func() {
					select {
					case <-h.ctx.Done():
					case <-time.NewTimer(time.Duration(duration) * time.Millisecond).C:
						h.neighborDiscoveryMtx.Lock()
						h.neighborDiscoveryAlgorithm.TimerExpire(target_local_session)
						h.neighborDiscoveryMtx.Unlock()
					}
				}()
			case NeighborEventDebug:
				fmt.Println(time.Now().Format("00:00:00.000") + " " + e.Text)
			}
		}
	}
}
