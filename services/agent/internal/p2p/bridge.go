package p2p

import (
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"sync"

	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
)

// Bridge relays UDP packets between local WireGuard and remote peers via libp2p.
// Each peer gets its own proxy UDP port so packets are routed correctly.
//
// Outgoing: WG sends to 127.0.0.1:peerProxyPort → relay reads → libp2p stream → remote peer
// Incoming: libp2p stream → relay reads → sends to 127.0.0.1:wgPort → WG receives
type Bridge struct {
	manager  *Manager
	wgPort   int
	nextPort int

	mu    sync.Mutex
	peers map[peer.ID]*peerRelay
}

type peerRelay struct {
	proxyConn *net.UDPConn
	proxyPort int
	stream    network.Stream
	cancel    context.CancelFunc
}

func (m *Manager) NewBridge(wgPort, baseProxyPort int) (*Bridge, error) {
	b := &Bridge{
		manager:  m,
		wgPort:   wgPort,
		nextPort: baseProxyPort,
		peers:    make(map[peer.ID]*peerRelay),
	}

	m.logger.Infof("WG Bridge: initialized (WG port: %d, proxy base: %d)", wgPort, baseProxyPort)
	return b, nil
}

func (b *Bridge) RegisterIncoming() {
	b.manager.host.SetStreamHandler(ProtocolWG, func(s network.Stream) {
		remote := s.Conn().RemotePeer()
		b.manager.logger.Infof("WG Bridge: incoming stream from %s", remote.ShortString())

		b.mu.Lock()
		relay, exists := b.peers[remote]
		if exists && relay.stream != nil {
			relay.stream.Reset()
			relay.stream = s
			b.mu.Unlock()
			b.streamToUDP(s, remote, relay.proxyConn)
			return
		}
		b.mu.Unlock()

		// No relay yet for this peer — create one
		port, err := b.allocateRelay(remote, s)
		if err != nil {
			b.manager.logger.Warnf("WG Bridge: failed to create relay for %s: %v", remote.ShortString(), err)
			s.Reset()
			return
		}
		b.manager.logger.Infof("WG Bridge: relay created for %s on proxy port %d", remote.ShortString(), port)
	})
}

func (b *Bridge) AddPeer(ctx context.Context, peerID peer.ID) (int, error) {
	stream, err := b.manager.host.NewStream(ctx, peerID, ProtocolWG)
	if err != nil {
		return 0, fmt.Errorf("bridge: stream to %s failed: %w", peerID.ShortString(), err)
	}

	port, err := b.allocateRelay(peerID, stream)
	if err != nil {
		stream.Close()
		return 0, err
	}

	b.manager.logger.Infof("WG Bridge: peer %s on proxy port %d", peerID.ShortString(), port)
	return port, nil
}

func (b *Bridge) allocateRelay(peerID peer.ID, stream network.Stream) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if relay, exists := b.peers[peerID]; exists {
		if relay.stream != nil {
			relay.stream.Reset()
		}
		relay.stream = stream
		go b.streamToUDP(stream, peerID, relay.proxyConn)
		return relay.proxyPort, nil
	}

	port := b.nextPort
	b.nextPort++

	addr := &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: port}
	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		return 0, fmt.Errorf("bridge: failed to listen on UDP %d: %w", port, err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	relay := &peerRelay{
		proxyConn: conn,
		proxyPort: port,
		stream:    stream,
		cancel:    cancel,
	}
	b.peers[peerID] = relay

	go b.udpToStream(ctx, relay, peerID)
	go b.streamToUDP(stream, peerID, conn)

	return port, nil
}

func (b *Bridge) RemovePeer(peerID peer.ID) {
	b.mu.Lock()
	relay, exists := b.peers[peerID]
	if !exists {
		b.mu.Unlock()
		return
	}
	delete(b.peers, peerID)
	b.mu.Unlock()

	relay.cancel()
	if relay.stream != nil {
		relay.stream.Close()
	}
	relay.proxyConn.Close()
	b.manager.logger.Infof("WG Bridge: removed relay for %s", peerID.ShortString())
}

func (b *Bridge) GetProxyPort(peerID peer.ID) (int, bool) {
	b.mu.Lock()
	defer b.mu.Unlock()
	relay, exists := b.peers[peerID]
	if !exists {
		return 0, false
	}
	return relay.proxyPort, true
}

func (b *Bridge) udpToStream(ctx context.Context, relay *peerRelay, peerID peer.ID) {
	buf := make([]byte, 65535)
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		n, _, err := relay.proxyConn.ReadFromUDP(buf)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			b.manager.logger.Warnf("WG Bridge: UDP read for %s: %v", peerID.ShortString(), err)
			continue
		}

		b.mu.Lock()
		s := relay.stream
		b.mu.Unlock()

		if s == nil {
			continue
		}

		if err := writePacket(s, buf[:n]); err != nil {
			b.manager.logger.Warnf("WG Bridge: write to %s failed: %v", peerID.ShortString(), err)
		}
	}
}

func (b *Bridge) streamToUDP(s network.Stream, peerID peer.ID, proxyConn *net.UDPConn) {
	wgAddr := &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: b.wgPort}

	for {
		pkt, err := readPacket(s)
		if err != nil {
			if err != io.EOF {
				b.manager.logger.Warnf("WG Bridge: stream read from %s: %v", peerID.ShortString(), err)
			}
			return
		}

		if _, err := proxyConn.WriteToUDP(pkt, wgAddr); err != nil {
			b.manager.logger.Warnf("WG Bridge: UDP write error: %v", err)
		}
	}
}

func (b *Bridge) Close() error {
	b.mu.Lock()
	peers := make(map[peer.ID]*peerRelay, len(b.peers))
	for k, v := range b.peers {
		peers[k] = v
	}
	b.peers = make(map[peer.ID]*peerRelay)
	b.mu.Unlock()

	for _, relay := range peers {
		relay.cancel()
		if relay.stream != nil {
			relay.stream.Close()
		}
		relay.proxyConn.Close()
	}
	return nil
}

func writePacket(w io.Writer, data []byte) error {
	header := make([]byte, 2)
	binary.BigEndian.PutUint16(header, uint16(len(data)))
	if _, err := w.Write(header); err != nil {
		return err
	}
	_, err := w.Write(data)
	return err
}

func readPacket(r io.Reader) ([]byte, error) {
	header := make([]byte, 2)
	if _, err := io.ReadFull(r, header); err != nil {
		return nil, err
	}
	length := binary.BigEndian.Uint16(header)
	data := make([]byte, length)
	if _, err := io.ReadFull(r, data); err != nil {
		return nil, err
	}
	return data, nil
}
