package test

import (
	"context"
	"fmt"
	"testing"
	"time"

	abyss "abyss_neighbor_discovery/interfaces"

	"github.com/google/uuid"
)

func TestHost(t *testing.T) {
	var hostA abyss.IAbyssHost
	var hostB abyss.IAbyssHost

	host_a_session := uuid.New()
	host_a_ch := hostA.OpenWorld(host_a_session, "http://a.world.com")
	fmt.Println("[hostA] Opened World")
	go func() {
		select {
		case <-time.After(10 * time.Second):
			hostA.LeaveWorld(host_a_session)
			fmt.Println("[hostA] Left World")
			return
		case sra := <-host_a_ch:
			fmt.Println("[hostA] Session Request: " + sra.Peer.IDHash())
			hostA.RespondSessionRequest(sra, true, 500, "OK")
		}
	}()

	host_a_aurl := hostA.GetLocalAbyssURL()
	host_a_aurl.Path = "a_home"

	<-time.After(time.Second)
	host_b_session := uuid.New()
	host_b_ch := hostB.JoinWorld(context.Background(), host_b_session, host_a_aurl.ToString())
	fmt.Println("[hostB] Joined World")
	select {
	case <-time.After(5 * time.Second):
		hostB.LeaveWorld(host_b_session)
		fmt.Println("[hostB] Left World")
		return
	case sra := <-host_b_ch:
		fmt.Println("[hostB] Session Request: " + sra.Peer.IDHash())
		hostA.RespondSessionRequest(sra, true, 500, "OK")
	}
}
