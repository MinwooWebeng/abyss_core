package net_service

import (
	"net"
	"sync"
)

type BetaAddressSelector struct {
	localPrivateAddr net.IP
	localPublicAddr  net.IP //can be added later

	mtx *sync.Mutex
}

func NewBetaAddressSelector() *BetaAddressSelector {
	conn, err := net.DialUDP("udp", nil, &net.UDPAddr{
		IP:   net.IPv4(8, 8, 8, 8), // Google's public DNS as an example
		Port: 53,
	})
	if err != nil {
		panic("no network interface detected")
	}

	localAddr := conn.LocalAddr().(*net.UDPAddr)
	conn.Close()

	return &BetaAddressSelector{
		localAddr.IP,
		net.IPv4zero,
		new(sync.Mutex),
	}
}

func (s *BetaAddressSelector) SetPublicIP(ip net.IP) {
	s.mtx.Lock()
	s.localPublicAddr = ip
	s.mtx.Unlock()
}

func (s *BetaAddressSelector) FilterAddressCandidates(addresses []*net.UDPAddr) []*net.UDPAddr {
	public_addresses := make([]*net.UDPAddr, 0)

	var loopbackaddr *net.UDPAddr
	var privateaddr *net.UDPAddr

	for _, address := range addresses {
		if address.IP.Equal(net.IPv4zero) || address.IP.Equal(net.IPv4bcast) {
			continue
		}

		if address.IP.Equal([]byte{127, 0, 0, 1}) {
			loopbackaddr = address
			continue
		}

		if address.IP[0] == 192 && address.IP[1] == 168 {
			privateaddr = address
			continue
		}

		s.mtx.Lock()
		is_pub_eq := address.IP.Equal(s.localPublicAddr)
		s.mtx.Unlock()
		if is_pub_eq {
			continue //ignore same public address
		}

		public_addresses = append(public_addresses, address)
	}

	if len(public_addresses) == 0 { //no public address found
		if privateaddr != nil && !privateaddr.IP.Equal(s.localPrivateAddr) {
			return []*net.UDPAddr{privateaddr}
		}

		if loopbackaddr != nil {
			return []*net.UDPAddr{loopbackaddr}
		}
		return []*net.UDPAddr{}
	}
	return public_addresses
}
