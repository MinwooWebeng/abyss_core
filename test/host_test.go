package test

import (
	"context"
	"fmt"
	"testing"
	"time"

	abyss_host "abyss_neighbor_discovery/host"
	abyss "abyss_neighbor_discovery/interfaces"
	"abyss_neighbor_discovery/tools/functional"

	"github.com/google/uuid"
)

func printWorldEvents(prefix string, host abyss.IAbyssHost, world abyss.IAbyssWorld) {
	ev_ch := world.GetEventChannel()

	for {
		select {
		case <-time.After(10 * time.Second):
			host.LeaveWorld(world)
			fmt.Println(prefix + " Left World")
			return
		case event_unknown := <-ev_ch:
			switch event := event_unknown.(type) {
			case abyss.EWorldPeerRequest:
				fmt.Println(prefix + " accepting " + event.PeerHash)
				event.Accept()
			case abyss.EWorldPeerReady:
				fmt.Println(prefix + " peer ready: " + event.Peer.Hash())
				//event.Peer.AppendObjects([]abyss.ObjectInfo{abyss.ObjectInfo{ID: uuid.New(), Address: "https://abyssal.com/cat.obj"}})
			case abyss.EPeerObjectAppend:
				fmt.Println(prefix + " " + event.PeerHash + " appended" + functional.Accum_all(event.Objects, "", func(obj abyss.ObjectInfo, accum string) string {
					return accum + " " + obj.ID.String() + "|" + obj.Address
				}))
			case abyss.EPeerObjectDelete:
				fmt.Println(prefix + " " + event.PeerHash + " deleted" + functional.Accum_all(event.ObjectIDs, "", func(obj uuid.UUID, accum string) string { return accum + " " + obj.String() }))
			case abyss.EWorldPeerLeave:
				fmt.Println(prefix + " peer leave: " + event.PeerHash)
			default:
				panic("unknown world event")
			}
		}
	}
}

func TestHost(t *testing.T) {
	hostA, hostA_pathRes := abyss_host.NewBetaAbyssHost("hostA")
	hostB, _ := abyss_host.NewBetaAbyssHost("hostB")

	A_world, err := hostA.OpenWorld("http://a.world.com")
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	fmt.Println("[hostA] Opened World")
	hostA_pathRes.SetMapping("/home", A_world.SessionID()) //this opens the world for join from A's side
	go printWorldEvents("[hostA] ", hostA, A_world)

	join_url := hostA.GetLocalAbyssURL()
	join_url.Path = "/home"
	B_A_world, err := hostB.JoinWorld(context.Background(), join_url)
	if err != nil {
		fmt.Println(err.Error())
		return
	}

	printWorldEvents("[hostB] ", hostB, B_A_world)
}
