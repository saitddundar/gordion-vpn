package p2p

import (
	"context"
	"fmt"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/p2p/protocol/ping"
	"github.com/multiformats/go-multiaddr"
	pkglogger "github.com/saitddundar/gordion-vpn/pkg/logger"
)

type Manager struct {
	host   host.Host
	logger pkglogger.Logger
}

func New(ctx context.Context, logger pkglogger.Logger, listenPort int) (*Manager, error) {
	// Listen on all interfaces on the specified port
	listenAddr := fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", listenPort)

	h, err := libp2p.New(
		libp2p.ListenAddrStrings(listenAddr),
		// Noise for security, Yamux for multiplexing
		libp2p.DefaultTransports,
		libp2p.DefaultSecurity,
		libp2p.DefaultMuxers,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create libp2p host: %w", err)
	}

	logger.Infof("libp2p host started with ID: %s", h.ID())
	for _, addr := range h.Addrs() {
		logger.Infof("  Listening on: %s/p2p/%s", addr, h.ID())
	}

	// Start listening to incoming PING requests
	pingService := ping.NewPingService(h)
	_ = pingService

	return &Manager{
		host:   h,
		logger: logger,
	}, nil
}

func (m *Manager) Host() host.Host {
	return m.host
}

func (m *Manager) PeerID() string {
	return m.host.ID().String()
}

// returns the host's listening addresses
func (m *Manager) Multiaddrs() []string {
	addrs := m.host.Addrs()
	res := make([]string, len(addrs))
	for i, addr := range addrs {
		res[i] = fmt.Sprintf("%s/p2p/%s", addr, m.host.ID())
	}
	return res
}

func (m *Manager) Close() error {
	m.logger.Info("Stopping libp2p host...")
	return m.host.Close()
}

func (m *Manager) GetPeerInfo(addrStr string) (*peer.AddrInfo, error) {
	addr, err := multiaddr.NewMultiaddr(addrStr)
	if err != nil {
		return nil, err
	}
	return peer.AddrInfoFromP2pAddr(addr)
}

// tests connection to another peer via libp2p
func (m *Manager) ConnectAndPing(ctx context.Context, pInfo peer.AddrInfo) error {
	m.logger.Infof("P2P: Attempting to connect to peer %s...", pInfo.ID.ShortString())

	if err := m.host.Connect(ctx, pInfo); err != nil {
		return fmt.Errorf("p2p connect failed: %w", err)
	}

	m.logger.Infof("P2P: Connected to %s. Sending Ping...", pInfo.ID.ShortString())
	ch := ping.Ping(ctx, m.host, pInfo.ID)

	res := <-ch
	if res.Error != nil {
		return fmt.Errorf("p2p ping failed: %w", res.Error)
	}

	m.logger.Infof("P2P: Ping successful to %s! RTT: %s", pInfo.ID.ShortString(), res.RTT)
	return nil
}
