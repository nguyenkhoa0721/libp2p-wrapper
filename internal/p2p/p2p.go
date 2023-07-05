package p2p

import (
	"context"
	"encoding/hex"
	"fmt"
	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/peerstore"
	"github.com/multiformats/go-multiaddr"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"time"
)

// NewNet @Description: Net is a wrapper of libp2p host
// @param port: port of this node
// @param privKeyStr: private key of this node
// @return *Net: a wrapper of libp2p host
func NewNet(_addr string, privKeyStr string) (*Net, error) {
	// @Description: decode private key
	privKeyHex, _ := hex.DecodeString(privKeyStr)
	privKey, err := crypto.UnmarshalSecp256k1PrivateKey(privKeyHex)
	if err != nil {
		return nil, err
	}

	// @Description: create libp2p host
	addr, err := multiaddr.NewMultiaddr(_addr)
	if err != nil {
		return nil, err
	}

	// @Description: create libp2p host
	host, err := libp2p.New(
		libp2p.ListenAddrs(addr),
		libp2p.Identity(privKey),
	)
	if err != nil {
		return nil, err
	}

	return &Net{
		Host:  host,
		Peers: make(map[string]*peer.AddrInfo),
	}, nil
}

// HandleStream @Description: HandleStream handles incoming stream
// @param s: incoming stream
// @return *Message: message received
func HandleStream(s network.Stream) (*Message, error) {
	// @Description: read header
	header := make([]byte, 1028)
	_, err := s.Read(header)
	if err != nil {
		SendErrorResponse(INTERNAL_ERROR_CODE, err.Error(), "", s)
		return nil, err
	}

	var msgHeader MessageHeader
	if err = msgHeader.Unmarshal(header); err != nil {
		SendErrorResponse(INTERNAL_ERROR_CODE, err.Error(), "", s)
		return nil, err
	}

	if msgHeader.Size == 0 {
		return &Message{
			Header: msgHeader,
			Body:   []byte{},
		}, nil
	}

	// @Description: read body
	body := make([]byte, msgHeader.Size)
	buffer := make([]byte, 1028)
	readSize := 0
	for readSize < msgHeader.Size {
		n, err := s.Read(buffer)
		if err != nil {
			SendErrorResponse(INTERNAL_ERROR_CODE, err.Error(), msgHeader.Session, s)
			return nil, err
		}

		copy(body[readSize:], buffer[:n])
		readSize += n
	}

	return &Message{
		Header: msgHeader,
		Body:   body,
	}, nil
}

func SendErrorResponse(
	statusCode int,
	errorMessage string,
	session string,
	s network.Stream,
) {
	header := MessageHeader{
		Type:         RES_TYPE,
		Size:         0,
		Session:      session,
		Status:       statusCode,
		ErrorMessage: errorMessage,
		IsBroadcast:  false,
	}

	headerBytes, _ := header.Marshal()

	s.Write(headerBytes)
	s.Close()
}

func SendSuccessResponse(
	session string,
	s network.Stream,
) {
	header := MessageHeader{
		Type:         RES_TYPE,
		Size:         0,
		Session:      session,
		Status:       SUSSCESS_CODE,
		ErrorMessage: "",
		IsBroadcast:  false,
	}

	headerBytes, _ := header.Marshal()

	s.Write(headerBytes)
	s.Close()
}

// ConnectToPeer @Description: ConnectToPeer connects to a peer
// @param peerId: id of the peer
// @param dest: addr of the peer
// @return string: id of the peer
func (net Net) ConnectToPeer(dest string) string {
	// @Description: create multiaddr from dest
	maddr, err := multiaddr.NewMultiaddr(dest)
	if err != nil {
		logrus.Errorf("error creating multiaddr from %s: %v", dest, err)
	}

	info, err := peer.AddrInfoFromP2pAddr(maddr)
	if err != nil {
		logrus.Errorf("error extracting peer ID from multiaddr: %v", err)
	}

	// @Description: connect to peer
	// Retry until success, add peer to peerstore and save peer info in net.Peers
	for {
		net.Host.Peerstore().AddAddrs(info.ID, info.Addrs, peerstore.PermanentAddrTTL)

		_, err := net.Host.NewStream(context.Background(), info.ID, "/")
		if err != nil {
			logrus.Errorf("Error creating stream: %v. Retrying...", err)
			time.Sleep(RETRY * time.Second)
			continue
		}

		net.Peers[info.ID.String()] = info
		logrus.Infof("Connected to peer %s", info.ID.String())
		return info.ID.String()
	}
}

// SendRequest @Description: SendRequest sends a request to a peer
// @param session: session of the request
// @param peerId: id of the peer
// @param message: message to send
func (net Net) SendRequest(session string, to string, message Message) error {
	// @Description: get peer info
	info := net.Peers[to]
	if info == nil {
		return errors.New(fmt.Sprintf("peer %d not found", to))
	}

	// @Description: create header
	message.Header.Size = len(message.Body)
	message.Header.Status = 0
	message.Header.ErrorMessage = ""
	message.Header.Session = session

	headerBytes, err := message.Header.Marshal()
	if err != nil {
		return fmt.Errorf("error marshalling message header: %v", err)
	}

	// @Description: create stream
	s, err := net.Host.NewStream(context.Background(), info.ID, "/")
	if err != nil {
		return fmt.Errorf("Error creating stream: %v", err)
	}

	// @Description: send header and body
	_, err = s.Write(headerBytes)
	if err != nil {
		s.Close()
		return fmt.Errorf("error sending message header: %v", err)
	}

	_, err = s.Write(message.Body)
	if err != nil {
		s.Close()
		return fmt.Errorf("error sending message body: %v", err)
	}

	// @Description: receive response and close stream
	err = net.receiveResponse(s)
	s.Close()
	return err
}

// receiveResponse @Description: receiveResponse receives response from a peer
func (net Net) receiveResponse(s network.Stream) error {
	// @Description: read header
	header := make([]byte, 1028)
	if _, err := s.Read(header); err != nil {
		return err
	}

	var msgHeader MessageHeader
	if err := msgHeader.Unmarshal(header); err != nil {
		return err
	}

	// @Description: Handle error
	if msgHeader.Status != SUSSCESS_CODE {
		return errors.New(msgHeader.ErrorMessage)
	}

	return nil
}

// Broadcast @Description: Broadcast sends a message to all peers
func (net Net) Broadcast(session string, message Message) error {
	errs := make([]error, 0, len(net.Peers))

	message.Header.IsBroadcast = true
	for peerId, _ := range net.Peers {
		err := net.SendRequest(session, peerId, message)
		if err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("failed to broadcast message to %d peers: %v", len(errs), errs)
	}

	return nil
}
