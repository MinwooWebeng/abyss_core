package test

import (
	"fmt"
	"strconv"
	"sync"
	"testing"

	abyss "abyss_neighbor_discovery/interfaces"

	abyss_host "abyss_neighbor_discovery/host"

	"github.com/google/uuid"
)

type HostEventPrinter struct {
	prefix      string
	world_paths map[string]uuid.UUID
	paths_mtx   *sync.Mutex
}

func NewHostEventPrinter(prefix string) *HostEventPrinter {
	return &HostEventPrinter{
		prefix:      prefix,
		world_paths: make(map[string]uuid.UUID),
		paths_mtx:   new(sync.Mutex),
	}
}

func (h *HostEventPrinter) PathToSessionID(path string, _ string) (uuid.UUID, bool) {
	h.paths_mtx.Lock()
	u, b := h.world_paths[path]
	h.paths_mtx.Unlock()
	return u, b
}
func (h *HostEventPrinter) OnSessionRequest(local_session_id uuid.UUID, peer abyss.IANDPeer, peer_session_id uuid.UUID) (ok bool, code int, message string) {
	fmt.Println(h.prefix + "OnSessionRequest " + local_session_id.String() + " " + peer.IDHash() + " " + peer_session_id.String())
	return true, 500, "OK"
}
func (h *HostEventPrinter) OnSessionReady(local_session_id uuid.UUID, peer abyss.IANDPeer, peer_session_id uuid.UUID) {
	fmt.Println(h.prefix + "OnSessionReady " + local_session_id.String() + " " + peer.IDHash() + " " + peer_session_id.String())
}
func (h *HostEventPrinter) OnSessionClose(local_session_id uuid.UUID, peer abyss.IANDPeer, peer_session_id uuid.UUID) {
	fmt.Println(h.prefix + "OnSessionClose " + local_session_id.String() + " " + peer.IDHash() + " " + peer_session_id.String())
}
func (h *HostEventPrinter) OnJoinSuccess(local_session_id uuid.UUID, world_url string) {
	fmt.Println(h.prefix + "OnJoinSuccess " + local_session_id.String() + " " + world_url)
}
func (h *HostEventPrinter) OnJoinFail(local_session_id uuid.UUID, code int, message string) {
	fmt.Println(h.prefix + "OnJoinFail " + local_session_id.String() + " " + strconv.Itoa(code) + " " + message)
}

func (h *HostEventPrinter) SetWorldPath(path string, session_id uuid.UUID) {
	h.paths_mtx.Lock()
	h.world_paths[path] = session_id
	h.paths_mtx.Unlock()
}

func TestNetHost(t *testing.T) {
	mallang, mallang_netserv := abyss_host.NewBetaAbyssNetHost("mallang")
	meiko, meiko_netserv := abyss_host.NewBetaAbyssNetHost("meiko")

	mallang_handle := NewHostEventPrinter("[mallang] ")
	meiko_handle := NewHostEventPrinter("[meiko] ")

	mallang.Handle(mallang_handle)
	meiko.Handle(meiko_handle)

	go mallang.ListenAndServe()
	go meiko.ListenAndServe()

	mallang_url := mallang_netserv.LocalAURL()
	meiko_url := meiko_netserv.LocalAURL()

	fmt.Println(mallang_url.ToString())
	fmt.Println(meiko_url.ToString())

	mallang_home_session_id := uuid.New()
	mallang_handle.SetWorldPath("home", mallang_home_session_id)
	mallang.OpenWorld(mallang_home_session_id, "home")

	//meiko_join_session_id := uuid.New()

	//meiko.JoinWorld(meiko_join_session_id, )
}
