package ahmp

import (
	"abyss_neighbor_discovery/aurl"
	abyss "abyss_neighbor_discovery/interfaces"
	"abyss_neighbor_discovery/tools/functional"
	"errors"

	"github.com/google/uuid"
)

type IDFrame struct { //delivers real identity -
	Payload []byte //certificate - local id -> what? metadata?
}

type DummyAuth struct {
	Name string
}

type SessionInfoText struct {
	AURL      string
	SessionID string
}

const (
	JN_T int = iota
	JOK_T
	JDN_T
	JNI_T
	MEM_T
	SNB_T
	CRR_T
	RST_T

	SOA_T
	SOD_T
)

type RawJN struct {
	SenderSessionID string
	Text            string
}

func (r *RawJN) TryParse() (*JN, error) {
	ssid, err := uuid.Parse(r.SenderSessionID)
	if err != nil {
		return nil, err
	}
	return &JN{ssid, r.Text}, nil
}

type RawJOK struct {
	SenderSessionID string
	RecverSessionID string
	Neighbors       []SessionInfoText
	Text            string
}

func (r *RawJOK) TryParse() (*JOK, error) {
	ssid, err := uuid.Parse(r.SenderSessionID)
	if err != nil {
		return nil, err
	}
	rsid, err := uuid.Parse(r.RecverSessionID)
	if err != nil {
		return nil, err
	}
	neig, ok := functional.Filter_strict_ok(r.Neighbors, func(i SessionInfoText) (abyss.ANDPeerSessionInfo, bool) {
		abyss_url, err := aurl.ParseAURL(i.AURL)
		if err != nil {
			return abyss.ANDPeerSessionInfo{}, false
		}
		psid, err := uuid.Parse(i.SessionID)
		if err != nil {
			return abyss.ANDPeerSessionInfo{}, false
		}
		return abyss.ANDPeerSessionInfo{AURL: abyss_url, SessionID: psid}, true
	})
	if !ok {
		return nil, errors.New("failed to parse session information")
	}
	return &JOK{ssid, rsid, neig, r.Text}, nil
}

type RawJDN struct {
	RecverSessionID string
	Text            string
	Code            int
}

func (r *RawJDN) TryParse() (*JDN, error) {
	rsid, err := uuid.Parse(r.RecverSessionID)
	if err != nil {
		return nil, err
	}
	return &JDN{rsid, r.Text, r.Code}, nil
}

type RawJNI struct {
	SenderSessionID string
	RecverSessionID string
	Neighbor        SessionInfoText
}

func (r *RawJNI) TryParse() (*JNI, error) {
	ssid, err := uuid.Parse(r.SenderSessionID)
	if err != nil {
		return nil, err
	}
	rsid, err := uuid.Parse(r.RecverSessionID)
	if err != nil {
		return nil, err
	}

	abyss_url, err := aurl.ParseAURL(r.Neighbor.AURL)
	if err != nil {
		return nil, err
	}
	psid, err := uuid.Parse(r.Neighbor.SessionID)
	if err != nil {
		return nil, err
	}
	return &JNI{ssid, rsid, abyss.ANDPeerSessionInfo{AURL: abyss_url, SessionID: psid}}, nil
}

type RawMEM struct {
	SenderSessionID string
	RecverSessionID string
}

func (r *RawMEM) TryParse() (*MEM, error) {
	ssid, err := uuid.Parse(r.SenderSessionID)
	if err != nil {
		return nil, err
	}
	rsid, err := uuid.Parse(r.RecverSessionID)
	if err != nil {
		return nil, err
	}
	return &MEM{ssid, rsid}, nil
}

type RawSNB struct {
	SenderSessionID string
	RecverSessionID string
	Hashes          []string
}

func (r *RawSNB) TryParse() (*SNB, error) {
	ssid, err := uuid.Parse(r.SenderSessionID)
	if err != nil {
		return nil, err
	}
	rsid, err := uuid.Parse(r.RecverSessionID)
	if err != nil {
		return nil, err
	}
	return &SNB{ssid, rsid, r.Hashes}, nil
}

type RawCRR struct {
	SenderSessionID string
	RecverSessionID string
	Hashes          []string
}

func (r *RawCRR) TryParse() (*CRR, error) {
	ssid, err := uuid.Parse(r.SenderSessionID)
	if err != nil {
		return nil, err
	}
	rsid, err := uuid.Parse(r.RecverSessionID)
	if err != nil {
		return nil, err
	}
	return &CRR{ssid, rsid, r.Hashes}, nil
}

type RawRST struct {
	SenderSessionID string
	RecverSessionID string
}

func (r *RawRST) TryParse() (*RST, error) {
	ssid, err := uuid.Parse(r.SenderSessionID)
	if err != nil {
		return nil, err
	}
	rsid, err := uuid.Parse(r.RecverSessionID)
	if err != nil {
		return nil, err
	}
	return &RST{ssid, rsid}, nil
}

type RawSOA struct {
	SenderSessionID string
	RecverSessionID string
	//TODO
}

func (r *RawSOA) TryParse() (*SOA, error) {
	ssid, err := uuid.Parse(r.SenderSessionID)
	if err != nil {
		return nil, err
	}
	rsid, err := uuid.Parse(r.RecverSessionID)
	if err != nil {
		return nil, err
	}
	return &SOA{ssid, rsid}, nil
}

type RawSOD struct {
	SenderSessionID string
	RecverSessionID string
	//TODO
}

func (r *RawSOD) TryParse() (*SOD, error) {
	ssid, err := uuid.Parse(r.SenderSessionID)
	if err != nil {
		return nil, err
	}
	rsid, err := uuid.Parse(r.RecverSessionID)
	if err != nil {
		return nil, err
	}
	return &SOD{ssid, rsid}, nil
}
