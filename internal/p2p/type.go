package p2p

import (
	"bytes"
	"encoding/json"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
)

type Net struct {
	Host  host.Host
	Peers map[string]*peer.AddrInfo
}

type MessageHeader struct {
	Type         int    `json:"type"`
	Size         int    `json:"size"`
	Session      string `json:"session"`
	Status       int    `json:"status"`
	ErrorMessage string `json:"error_message"`
	IsBroadcast  bool   `json:"is_broadcast"`
}

type Message struct {
	Header MessageHeader
	Body   []byte
}

// Marsall converts a message header to a byte array
func (m *MessageHeader) Marshal() ([]byte, error) {
	var b bytes.Buffer
	err := json.NewEncoder(&b).Encode(m)
	if err != nil {
		return nil, err
	}

	return b.Bytes(), nil
}

// Unmarshal converts a byte array to a message header
func (m *MessageHeader) Unmarshal(b []byte) error {
	return json.NewDecoder(bytes.NewReader(b)).Decode(m)
}
