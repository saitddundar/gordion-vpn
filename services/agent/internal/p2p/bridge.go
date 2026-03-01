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

// Bridge relays UDP packets between local WireGuard and a remote peer via libp2p.
//
// Outgoing: WG sends to 127.0.0.1:proxyPort → Bridge reads → libp2p stream → remote peer
// Incoming: libp2p stream → Bridge reads → sends to 127.0.0.1:wgPort → WG receives
type Bridge struct {
	manager   *Manager
	proxyConn *net.UDPConn
	proxyPort int
	wgPort    int

	mu      sync.Mutex
	streams map[peer.ID]network.Stream
}

func (m *Manager) NewBridge(proxyPort, wgPort int) (*Bridge, error) {
	addr := &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: proxyPort}
	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		return nil, fmt.Errorf("bridge: failed to listen on UDP %d: %w", proxyPort, err)
	}

	b := &Bridge{
		manager:   m,
		proxyConn: conn,
		proxyPort: proxyPort,
		wgPort:    wgPort,
		streams:   make(map[peer.ID]network.Stream),
	}

	m.logger.Infof("WG Bridge: proxy UDP on 127.0.0.1:%d → WG on 127.0.0.1:%d", proxyPort, wgPort)
	return b, nil
}

// RegisterIncoming handles streams opened by remote peers.
func (b *Bridge) RegisterIncoming() {
	b.manager.host.SetStreamHandler(ProtocolWG, func(s network.Stream) {
		remote := s.Conn().RemotePeer()
		b.manager.logger.Infof("WG Bridge: incoming stream from %s", remote.ShortString())

		b.mu.Lock()
		if old, ok := b.streams[remote]; ok {
			old.Reset()
		}
		b.streams[remote] = s
		b.mu.Unlock()

		b.streamToUDP(s, remote)
	})
}

// ConnectToPeer opens a WG stream and starts relaying incoming packets.
func (b *Bridge) ConnectToPeer(ctx context.Context, peerID peer.ID) error {
	stream, err := b.manager.host.NewStream(ctx, peerID, ProtocolWG)
	if err != nil {
		return fmt.Errorf("bridge: stream to %s failed: %w", peerID.ShortString(), err)
	}

	b.mu.Lock()
	if old, ok := b.streams[peerID]; ok {
		old.Reset()
	}
	b.streams[peerID] = stream
	b.mu.Unlock()

	b.manager.logger.Infof("WG Bridge: stream opened to %s", peerID.ShortString())

	go b.streamToUDP(stream, peerID)
	return nil
}

// StartUDPRelay reads from proxy UDP socket and forwards to all connected streams.
func (b *Bridge) StartUDPRelay(ctx context.Context) {
	buf := make([]byte, 65535)
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		n, _, err := b.proxyConn.ReadFromUDP(buf)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			b.manager.logger.Warnf("WG Bridge: UDP read error: %v", err)
			continue
		}

		b.mu.Lock()
		for pid, stream := range b.streams {
			if err := writePacket(stream, buf[:n]); err != nil {
				b.manager.logger.Warnf("WG Bridge: write to %s failed: %v", pid.ShortString(), err)
				stream.Reset()
				delete(b.streams, pid)
			}
		}
		b.mu.Unlock()
	}
}

// streamToUDP reads length-prefixed packets from libp2p and writes to local WG port.
func (b *Bridge) streamToUDP(s network.Stream, remote peer.ID) {
	defer func() {
		b.mu.Lock()
		delete(b.streams, remote)
		b.mu.Unlock()
		s.Close()
	}()

	wgAddr := &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: b.wgPort}

	for {
		pkt, err := readPacket(s)
		if err != nil {
			if err != io.EOF {
				b.manager.logger.Warnf("WG Bridge: stream read from %s: %v", remote.ShortString(), err)
			}
			return
		}

		if _, err := b.proxyConn.WriteToUDP(pkt, wgAddr); err != nil {
			b.manager.logger.Warnf("WG Bridge: UDP write error: %v", err)
		}
	}
}

func (b *Bridge) Close() error {
	b.mu.Lock()
	for _, s := range b.streams {
		s.Close()
	}
	b.streams = make(map[peer.ID]network.Stream)
	b.mu.Unlock()
	return b.proxyConn.Close()
}

// Length-prefixed framing: [2 bytes big-endian length][payload]

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
