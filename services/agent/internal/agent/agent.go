package agent

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	pkglogger "github.com/saitddundar/gordion-vpn/pkg/logger"
	"github.com/saitddundar/gordion-vpn/services/agent/internal/client"
	"github.com/saitddundar/gordion-vpn/services/agent/internal/config"
	"github.com/saitddundar/gordion-vpn/services/agent/internal/wireguard"
)

type Agent struct {
	cfg    *config.Config
	client *client.Client
	wg_mgr *wireguard.Manager
	logger pkglogger.Logger

	nodeID string
	token  string
	vpnIP  string

	cancel context.CancelFunc
	wg     sync.WaitGroup
}

func New(cfg *config.Config, logger pkglogger.Logger) (*Agent, error) {
	c, err := client.New(cfg.IdentityAddr, cfg.DiscoveryAddr, cfg.ConfigAddr)
	if err != nil {
		return nil, err
	}

	wgMgr := wireguard.NewManager(logger, *cfg.DryRun)

	return &Agent{
		cfg:    cfg,
		client: c,
		wg_mgr: wgMgr,
		logger: logger,
	}, nil
}

func (a *Agent) Start(ctx context.Context) error {
	ctx, a.cancel = context.WithCancel(ctx)

	// Step 0: Generate WireGuard keypair
	a.logger.Info("Generating WireGuard keypair...")
	keyPair, err := wireguard.GenerateKeyPair()
	if err != nil {
		return err
	}
	a.logger.Infof("Public key: %s", keyPair.PublicKey[:16]+"...")

	// Step 1: Register with Identity Service
	a.logger.Info("Registering with Identity Service...")
	nodeID, token, err := a.client.Register(ctx, keyPair.PublicKey)
	if err != nil {
		return err
	}
	a.nodeID = nodeID
	a.token = token
	a.logger.Infof("Registered as %s", a.nodeID)

	// Step 2: Get network config
	a.logger.Info("Fetching network config...")
	netCfg, err := a.client.GetNetworkConfig(ctx, a.token)
	if err != nil {
		return err
	}
	a.logger.Infof("Network: %s, MTU: %d, DNS: %v", netCfg.NetworkCidr, netCfg.Mtu, netCfg.DnsServers)

	// Step 3: Request VPN IP
	a.logger.Info("Requesting VPN IP...")
	ip, subnet, gw, err := a.client.RequestIP(ctx, a.token, a.nodeID)
	if err != nil {
		return err
	}
	a.vpnIP = ip
	a.logger.Infof("VPN IP: %s, Subnet: %s, Gateway: %s", ip, subnet, gw)

	// Step 4: Register as peer in Discovery
	a.logger.Info("Announcing to Discovery Service...")
	if err := a.client.RegisterPeer(ctx, a.token, a.vpnIP, int32(a.cfg.WireGuardPort)); err != nil {
		return err
	}
	a.logger.Info("Peer registered")

	// Step 5: Discover other peers
	peers, err := a.client.DiscoverPeers(ctx, 10)
	if err != nil {
		a.logger.Warnf("Peer discovery failed: %v", err)
	} else {
		a.logger.Infof("Found %d peers", len(peers))
		for _, p := range peers {
			a.logger.Infof("  Peer: %s", p.NodeId)
		}
	}

	// Step 6: Configure WireGuard tunnel
	a.logger.Info("Configuring WireGuard tunnel...")
	dns := ""
	if len(netCfg.DnsServers) > 0 {
		dns = strings.Join(netCfg.DnsServers, ", ")
	}

	wgCfg := &wireguard.Config{
		PrivateKey: keyPair.PrivateKey,
		Address:    fmt.Sprintf("%s/%s", ip, subnet),
		MTU:        netCfg.Mtu,
		DNS:        dns,
	}

	// Add discovered peers to WireGuard config (fetch their public keys)
	for _, p := range peers {
		if p.NodeId == a.nodeID {
			continue // skip ourselves
		}

		peerKey, err := a.client.GetPeerPublicKey(ctx, p.NodeId)
		if err != nil {
			a.logger.Warnf("Failed to get public key for %s: %v", p.NodeId, err)
			continue
		}

		endpoint := fmt.Sprintf("%s:%d", p.IpAddress, p.Port)
		wgCfg.Peers = append(wgCfg.Peers, wireguard.PeerConfig{
			PublicKey:  peerKey,
			Endpoint:   endpoint,
			AllowedIPs: netCfg.NetworkCidr,
		})
		a.logger.Infof("  Added peer %s @ %s", p.NodeId, endpoint)
	}

	if err := a.wg_mgr.Configure(wgCfg); err != nil {
		a.logger.Errorf("WireGuard config failed: %v", err)
	}

	// Step 7: Start heartbeat loop
	a.wg.Add(1)
	go a.heartbeatLoop(ctx)

	a.logger.Info("Agent is running")
	return nil
}

func (a *Agent) Stop() {
	a.logger.Info("Stopping agent...")

	if a.cancel != nil {
		a.cancel()
	}
	a.wg.Wait()

	// Release IP before exit
	if a.vpnIP != "" && a.token != "" {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := a.client.ReleaseIP(ctx, a.token, a.nodeID, a.vpnIP); err != nil {
			a.logger.Errorf("Failed to release IP: %v", err)
		} else {
			a.logger.Infof("Released IP %s", a.vpnIP)
		}
	}

	// Tear down WireGuard
	if err := a.wg_mgr.Down(); err != nil {
		a.logger.Errorf("WireGuard down failed: %v", err)
	}

	a.client.Close()
	a.logger.Info("Agent stopped")
}

func (a *Agent) heartbeatLoop(ctx context.Context) {
	defer a.wg.Done()

	ticker := time.NewTicker(time.Duration(a.cfg.Heartbeat) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := a.client.Heartbeat(ctx, a.token); err != nil {
				a.logger.Errorf("Heartbeat failed: %v", err)
			} else {
				a.logger.Debug("Heartbeat sent")
			}
		}
	}
}
