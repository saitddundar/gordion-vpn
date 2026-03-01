package p2p

import (
	"bufio"
	"context"
	"fmt"

	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
)

const ProtocolWG = protocol.ID("/gordion/wg/1.0.0")

func (m *Manager) RegisterWGProtocol() {
	m.host.SetStreamHandler(ProtocolWG, func(s network.Stream) {
		remotePeer := s.Conn().RemotePeer()
		m.logger.Infof("WG-Stream: Incoming from %s", remotePeer.ShortString())

		reader := bufio.NewReader(s)
		line, err := reader.ReadString('\n')
		if err != nil {
			m.logger.Warnf("WG-Stream: Read error: %v", err)
			s.Reset()
			return
		}

		m.logger.Infof("WG-Stream: Received: %s", line)
		fmt.Fprintf(s, "ACK: %s\n", line)
		s.Close()
	})

	m.logger.Info("WG Protocol registered: /gordion/wg/1.0.0")
}

func (m *Manager) OpenWGStream(ctx context.Context, peerID peer.ID) error {
	m.logger.Infof("WG-Stream: Opening to %s...", peerID.ShortString())

	stream, err := m.host.NewStream(ctx, peerID, ProtocolWG)
	if err != nil {
		return fmt.Errorf("failed to open WG stream: %w", err)
	}
	defer stream.Close()

	msg := fmt.Sprintf("HELLO from %s", m.host.ID().ShortString())
	fmt.Fprintf(stream, "%s\n", msg)
	m.logger.Infof("WG-Stream: Sent: %s", msg)

	reader := bufio.NewReader(stream)
	ack, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("WG-Stream ACK failed: %w", err)
	}

	m.logger.Infof("WG-Stream: ACK: %s", ack)
	return nil
}
