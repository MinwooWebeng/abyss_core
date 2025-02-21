package net_service

import "net"

type AddressSelector struct {
	local_privateaddr net.IP
	local_publicaddr  net.IP //may not be available
}

func (s *AddressSelector) FilterAddressCandidates(addresses []*net.UDPAddr) []*net.UDPAddr {
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

		if address.IP.Equal(s.local_publicaddr) {
			continue //ignore same public address
		}

		public_addresses = append(public_addresses, address)
	}

	if len(public_addresses) == 0 { //no public address found
		if privateaddr != nil && !privateaddr.IP.Equal(s.local_privateaddr) {
			return []*net.UDPAddr{privateaddr}
		}

		if loopbackaddr != nil {
			return []*net.UDPAddr{loopbackaddr}
		}
		return []*net.UDPAddr{}
	}
	return public_addresses
}
