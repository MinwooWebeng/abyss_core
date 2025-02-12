package aurl

import (
	"net"
	"net/netip"
	"strconv"
	"strings"
)

type AURL struct {
	scheme    string //abyss(ahmp) or abyst(http3)
	hash      string
	addresses []*net.UDPAddr
	path      string
}

// abyss:abc:9.8.7.6:1605/somepath
// abyss:abc:[2001:db8:85a3:8d3:1319:8a2e:370:7348]:443|9.8.7.6:1605/somepath
// abyss:abc/somepath
// abyss:abc:9.8.7.6:1605
// abyss:abc
func (a *AURL) ToString() string {
	if len(a.addresses) == 0 {
		return "abyss:" + a.hash + a.path
	}
	candidates_string := make([]string, len(a.addresses))
	for i, c := range a.addresses {
		candidates_string[i] = c.String()
	}
	return "abyss:" + a.hash + ":" + strings.Join(candidates_string, "|") + a.path
}

func (a *AURL) Scheme() string {
	return a.scheme
}
func (a *AURL) Hash() string {
	return a.hash
}
func (a *AURL) Addresses() []*net.UDPAddr {
	return a.addresses
}
func (a *AURL) Path() string {
	return a.path
}

type AURLParseError struct {
	Code int
}

func (u *AURLParseError) Error() string {
	var msg string
	switch u.Code {
	case 100:
		msg = "unsupported protocol"
	case 101:
		msg = "invalid format"
	case 102:
		msg = "hash too short"
	case 103:
		msg = "address candidate parse fail"
	default:
		msg = "unknown error (" + strconv.Itoa(u.Code) + ")"
	}
	return "failed to parse abyssURL: " + msg
}

func ParseAURL(raw string) (*AURL, error) {
	var scheme string
	var body string
	var ok bool
	if body, ok = strings.CutPrefix(raw, "abyss:"); ok {
		scheme = "abyss"
	} else {
		if body, ok = strings.CutPrefix(raw, "abyst:"); ok {
			scheme = "abyst"
		} else {
			return nil, &AURLParseError{Code: 100}
		}
	}

	body = strings.TrimPrefix(body, "//") //for old people

	hash_endpos := strings.IndexAny(body, ":/")
	if hash_endpos == -1 {
		//no candidates, no path
		return &AURL{
			scheme:    scheme,
			hash:      body,
			addresses: []*net.UDPAddr{},
			path:      "/",
		}, nil
	}
	hash := body[:hash_endpos]
	if len(hash) < 1 {
		return nil, &AURLParseError{Code: 102}
	}

	if body[hash_endpos] == ':' {
		cand_path := body[hash_endpos+1:]

		pathpos := strings.Index(cand_path, "/")
		var candidates_str string
		var path string
		if pathpos == -1 {
			//no path
			candidates_str = cand_path
			path = "/"
		} else {
			candidates_str = cand_path[:pathpos]
			path = cand_path[pathpos:]
		}

		c_split := strings.Split(candidates_str, "|")
		candidates := make([]*net.UDPAddr, len(c_split))
		for i, candidate := range c_split {
			addrport, err := netip.ParseAddrPort(candidate)
			if err != nil {
				return nil, err
			}
			cand := net.UDPAddrFromAddrPort(addrport)
			if cand.Port == 0 {
				return nil, &AURLParseError{Code: 103}
			}
			candidates[i] = cand
		}

		return &AURL{
			scheme:    scheme,
			hash:      hash,
			addresses: candidates,
			path:      path,
		}, nil
	} else if body[hash_endpos] == '/' {
		//only path
		return &AURL{
			scheme:    scheme,
			hash:      hash,
			addresses: []*net.UDPAddr{},
			path:      body[hash_endpos:],
		}, nil
	}
	panic("ParseAbyssURL: implementation error")
}
