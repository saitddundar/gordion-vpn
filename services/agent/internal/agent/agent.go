package agent

import (
	"context"
	"fmt"
	"math"
	"net"
	"strings"
	"sync"
	"time"

	pkglogger "github.com/saitddundar/gordion-vpn/pkg/logger"
	"github.com/saitddundar/gordion-vpn/services/agent/internal/client"
	"github.com/saitddundar/gordion-vpn/services/agent/internal/config"
	"github.com/saitddundar/gordion-vpn/services/agent/internal/p2p"
	"github.com/saitddundar/gordion-vpn/services/agent/internal/wireguard"
)

type Agent struct {
	cfg     *config.Config
	client  *client.Client
	wg_mgr  *wireguard.Manager
	p2p_mgr *p2p.Manager
	bridge  *p2p.Bridge
	logger  pkglogger.Logger

	nodeID    string
	vpnIP     string
	publicKey string

	tokenMu   sync.RWMutex
	token     string
	expiresAt int64

	peersMu     sync.RWMutex
	activePeers map[string]string

	cancel context.CancelFunc
	wg     sync.WaitGroup
}

func New(cfg *config.Config, logger pkglogger.Logger) (*Agent, error) {
	c, err := client.New(cfg.IdentityAddr, cfg.DiscoveryAddr, cfg.ConfigAddr, cfg.TLSCACert)
	if err != nil {
		return nil, err
	}

	wgMgr := wireguard.NewManager(logger, *cfg.DryRun)

	return &Agent{
		cfg:         cfg,
		client:      c,
		wg_mgr:      wgMgr,
		logger:      logger,
		activePeers: make(map[string]string),
	}, nil
}

func (a *Agent) Start(ctx context.Context) error {
	ctx, a.cancel = context.WithCancel(ctx)

	a.logger.Info("Starting P2P Host...")
	p2pMgr, err := p2p.New(ctx, a.logger, a.cfg.P2PPort)
	if err != nil {
		return fmt.Errorf("failed to start p2p host: %w", err)
	}
	a.p2p_mgr = p2pMgr

	bridge, err := a.p2p_mgr.NewBridge(a.cfg.WireGuardPort, a.cfg.WireGuardPort+100)
	if err != nil {
		return fmt.Errorf("failed to create bridge: %w", err)
	}
	a.bridge = bridge
	a.bridge.RegisterIncoming()

	a.logger.Info("Generating WireGuard keypair...")
	keyPair, err := wireguard.GenerateKeyPair()
	if err != nil {
		return err
	}
	a.publicKey = keyPair.PublicKey
	a.logger.Infof("Public key: %s", keyPair.PublicKey[:16]+"...")

	a.logger.Info("Registering with Identity Service...")
	nodeID, token, expiresAt, err := a.retryRegister(ctx, keyPair.PublicKey)
	if err != nil {
		return err
	}
	a.nodeID = nodeID
	a.setToken(token, expiresAt)
	a.logger.Infof("Registered as %s (token expires: %s)", a.nodeID,
		time.Unix(expiresAt, 0).Format("15:04:05"))

	a.logger.Info("Fetching network config...")
	netCfg, err := a.client.GetNetworkConfig(ctx, a.token)
	if err != nil {
		return err
	}
	a.logger.Infof("Network: %s, MTU: %d, DNS: %v", netCfg.NetworkCidr, netCfg.Mtu, netCfg.DnsServers)

	a.logger.Info("Requesting VPN IP...")
	ip, subnet, gw, err := a.client.RequestIP(ctx, a.token, a.nodeID)
	if err != nil {
		return err
	}
	a.vpnIP = ip
	a.logger.Infof("VPN IP: %s, Subnet: %s, Gateway: %s", ip, subnet, gw)

	a.logger.Info("Announcing to Discovery Service...")
	if err := a.client.RegisterPeer(ctx, a.token, a.vpnIP, int32(a.cfg.WireGuardPort), a.p2p_mgr.PeerID(), a.p2p_mgr.Multiaddrs()); err != nil {
		return err
	}
	a.logger.Info("Peer registered")

	peers, err := a.client.DiscoverPeers(ctx, a.getToken(), 10)
	if err != nil {
		a.logger.Warnf("Initial peer discovery failed: %v", err)
	} else {
		a.logger.Infof("Found %d peers", len(peers))
		for _, p := range peers {
			a.logger.Infof("  Peer: %s (P2P ID: %s)", p.NodeId, p.PeerId)

			if p.NodeId != a.nodeID && len(p.P2PAddrs) > 0 {
				go func(nodeID string, addrs []string) {
					pInfo, err := a.p2p_mgr.GetPeerInfo(addrs[0])
					if err != nil || pInfo == nil {
						return
					}
					pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
					defer cancel()
					if err := a.p2p_mgr.ConnectAndPing(pingCtx, *pInfo); err != nil {
						return
					}
					_, _ = a.bridge.AddPeer(pingCtx, pInfo.ID)
				}(p.NodeId, p.P2PAddrs)
			}
		}
	}

	dns := ""
	if len(netCfg.DnsServers) > 0 {
		dns = strings.Join(netCfg.DnsServers, ", ")
	}

	wgCfg := &wireguard.Config{
		PrivateKey: keyPair.PrivateKey,
		ListenPort: a.cfg.WireGuardPort,
		Address:    fmt.Sprintf("%s/%d", ip, maskToCIDR(subnet)),
		MTU:        netCfg.Mtu,
		DNS:        dns,
	}

	for _, p := range peers {
		if p.NodeId == a.nodeID {
			continue
		}
		peerKey, err := a.client.GetPeerPublicKey(ctx, p.NodeId)
		if err != nil {
			a.logger.Warnf("Failed to get public key for %s: %v", p.NodeId, err)
			continue
		}

		// Use the per-peer proxy port from bridge
		peerIDStr := p.PeerId
		pInfo, _ := a.p2p_mgr.GetPeerInfo(p.P2PAddrs[0])
		var endpoint string
		if pInfo != nil {
			if port, ok := a.bridge.GetProxyPort(pInfo.ID); ok {
				endpoint = fmt.Sprintf("127.0.0.1:%d", port)
			}
		}
		if endpoint == "" {
			a.logger.Warnf("No bridge proxy for %s, skipping WG peer", peerIDStr)
			continue
		}

		wgCfg.Peers = append(wgCfg.Peers, wireguard.PeerConfig{
			PublicKey:  peerKey,
			Endpoint:   endpoint,
			AllowedIPs: p.IpAddress + "/32", // host-only route, not entire subnet
		})
		a.logger.Infof("  Added peer %s @ %s", p.NodeId, endpoint)

		a.peersMu.Lock()
		a.activePeers[p.NodeId] = peerKey
		a.peersMu.Unlock()
	}

	if err := a.wg_mgr.Configure(wgCfg); err != nil {
		a.logger.Errorf("WireGuard config failed: %v", err)
	}

	a.wg.Add(3)
	go a.heartbeatLoop(ctx)
	go a.tokenRefreshLoop(ctx)
	go a.peerSyncLoop(ctx, netCfg.NetworkCidr)

	a.logger.Info("Agent is running")
	return nil
}

func (a *Agent) Stop() {
	a.logger.Info("Stopping agent...")

	if a.cancel != nil {
		a.cancel()
	}
	a.wg.Wait()

	if tok := a.getToken(); a.vpnIP != "" && tok != "" {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := a.client.ReleaseIP(ctx, tok, a.nodeID, a.vpnIP); err != nil {
			a.logger.Errorf("Failed to release IP: %v", err)
		} else {
			a.logger.Infof("Released IP %s", a.vpnIP)
		}
	}

	if err := a.wg_mgr.Down(); err != nil {
		a.logger.Errorf("WireGuard down failed: %v", err)
	}

	if a.bridge != nil {
		a.bridge.Close()
	}

	if a.p2p_mgr != nil {
		if err := a.p2p_mgr.Close(); err != nil {
			a.logger.Errorf("P2P down failed: %v", err)
		}
	}

	a.client.Close()
	a.logger.Info("Agent stopped")
}

func (a *Agent) getToken() string {
	a.tokenMu.RLock()
	defer a.tokenMu.RUnlock()
	return a.token
}

func (a *Agent) getExpiresAt() int64 {
	a.tokenMu.RLock()
	defer a.tokenMu.RUnlock()
	return a.expiresAt
}

func (a *Agent) setToken(token string, expiresAt int64) {
	a.tokenMu.Lock()
	defer a.tokenMu.Unlock()
	a.token = token
	a.expiresAt = expiresAt
}

func (a *Agent) heartbeatLoop(ctx context.Context) {
	defer a.wg.Done()

	ticker := time.NewTicker(time.Duration(a.cfg.Heartbeat) * time.Second)
	defer ticker.Stop()

	failCount := 0
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := a.client.Heartbeat(ctx, a.getToken()); err != nil {
				failCount++
				backoff := time.Duration(math.Min(float64(failCount*failCount), 60)) * time.Second
				a.logger.Errorf("Heartbeat failed (%d): %v, retry in %s", failCount, err, backoff)
				time.Sleep(backoff)
			} else {
				failCount = 0
				a.logger.Debug("Heartbeat sent")
			}
		}
	}
}

func (a *Agent) tokenRefreshLoop(ctx context.Context) {
	defer a.wg.Done()

	for {
		remaining := time.Until(time.Unix(a.getExpiresAt(), 0))
		refreshIn := time.Duration(float64(remaining) * 0.8)
		if refreshIn < 30*time.Second {
			refreshIn = 30 * time.Second
		}

		a.logger.Infof("Token refresh scheduled in %s", refreshIn.Round(time.Second))

		select {
		case <-ctx.Done():
			return
		case <-time.After(refreshIn):
			a.logger.Info("Refreshing token...")
			_, token, expiresAt, err := a.retryRegister(ctx, a.publicKey)
			if err != nil {
				a.logger.Errorf("Token refresh failed: %v", err)
				continue
			}
			a.setToken(token, expiresAt)
			a.logger.Infof("Token refreshed (expires: %s)",
				time.Unix(expiresAt, 0).Format("15:04:05"))
		}
	}
}

func (a *Agent) peerSyncLoop(ctx context.Context, networkCIDR string) {
	defer a.wg.Done()

	interval := time.Duration(a.cfg.PeerSyncInterval) * time.Second
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	a.logger.Infof("Peer sync loop started (interval: %s)", interval)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			a.syncPeers(ctx, networkCIDR)
		}
	}
}

func (a *Agent) syncPeers(ctx context.Context, networkCIDR string) {
	peers, err := a.client.DiscoverPeers(ctx, a.getToken(), 50)
	if err != nil {
		a.logger.Warnf("Peer sync: discovery failed: %v", err)
		return
	}

	// Build the set of currently online peers (excluding self).
	online := make(map[string]string) // nodeID → publicKey
	for _, p := range peers {
		if p.NodeId == a.nodeID {
			continue
		}
		key, err := a.client.GetPeerPublicKey(ctx, p.NodeId)
		if err != nil {
			a.logger.Warnf("Peer sync: get public key for %s failed: %v", p.NodeId, err)
			continue
		}
		online[p.NodeId] = key
	}

	a.peersMu.Lock()
	defer a.peersMu.Unlock()

	for nodeID, pubKey := range online {
		if _, exists := a.activePeers[nodeID]; exists {
			continue
		}

		var p2pAddrs []string
		var vpnIP string
		for _, p := range peers {
			if p.NodeId == nodeID {
				p2pAddrs = p.P2PAddrs
				vpnIP = p.IpAddress
				break
			}
		}

		var proxyPort int
		if len(p2pAddrs) > 0 {
			pInfo, err := a.p2p_mgr.GetPeerInfo(p2pAddrs[0])
			if err == nil && pInfo != nil {
				pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
				if err := a.p2p_mgr.ConnectAndPing(pingCtx, *pInfo); err == nil {
					proxyPort, _ = a.bridge.AddPeer(pingCtx, pInfo.ID)
				}
				cancel()
			}
		}

		if proxyPort == 0 {
			a.logger.Warnf("Peer sync: no bridge for %s, skipping", nodeID)
			continue
		}

		endpoint := fmt.Sprintf("127.0.0.1:%d", proxyPort)
		allowedIPs := vpnIP + "/32"
		if vpnIP == "" {
			allowedIPs = networkCIDR // fallback
		}
		if err := a.wg_mgr.AddPeer(wireguard.PeerConfig{
			PublicKey:  pubKey,
			Endpoint:   endpoint,
			AllowedIPs: allowedIPs,
		}); err != nil {
			a.logger.Warnf("Peer sync: add peer %s failed: %v", nodeID, err)
			continue
		}
		a.activePeers[nodeID] = pubKey
		a.logger.Infof("Peer sync: added peer %s via bridge @ %s", nodeID, endpoint)
	}

	// Remove peers that are no longer online.
	for nodeID, pubKey := range a.activePeers {
		if _, exists := online[nodeID]; exists {
			continue
		}
		if err := a.wg_mgr.RemovePeer(pubKey); err != nil {
			a.logger.Warnf("Peer sync: remove peer %s failed: %v", nodeID, err)
			continue
		}
		delete(a.activePeers, nodeID)
		a.logger.Infof("Peer sync: removed stale peer %s", nodeID)
	}
}

func (a *Agent) retryRegister(ctx context.Context, publicKey string) (string, string, int64, error) {
	maxRetries := 5
	for i := 0; i < maxRetries; i++ {
		nodeID, token, expiresAt, err := a.client.Register(ctx, publicKey, a.p2p_mgr.PeerID())
		if err == nil {
			return nodeID, token, expiresAt, nil
		}

		if i == maxRetries-1 {
			return "", "", 0, fmt.Errorf("registration failed after %d attempts: %w", maxRetries, err)
		}

		backoff := time.Duration(math.Pow(2, float64(i))) * time.Second
		a.logger.Warnf("Register attempt %d failed: %v, retrying in %s", i+1, err, backoff)

		select {
		case <-ctx.Done():
			return "", "", 0, ctx.Err()
		case <-time.After(backoff):
		}
	}
	return "", "", 0, fmt.Errorf("unreachable")
}

func maskToCIDR(mask string) int {
	ip := net.ParseIP(mask)
	if ip == nil {
		return 24
	}
	ones, _ := net.IPMask(ip.To4()).Size()
	return ones
}
