package agent

import (
	"context"
	"sync"
	"time"

	pkglogger "github.com/saitddundar/gordion-vpn/pkg/logger"
	"github.com/saitddundar/gordion-vpn/services/agent/internal/client"
	"github.com/saitddundar/gordion-vpn/services/agent/internal/config"
)

type Agent struct {
	cfg    *config.Config
	client *client.Client
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

	return &Agent{
		cfg:    cfg,
		client: c,
		logger: logger,
	}, nil
}

func (a *Agent) Start(ctx context.Context) error {
	ctx, a.cancel = context.WithCancel(ctx)

	// Step 1: Register with Identity Service
	a.logger.Info("Registering with Identity Service...")
	nodeID, token, err := a.client.Register(ctx, a.cfg.PublicKey)
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
	if err := a.client.RegisterPeer(ctx, a.token, a.vpnIP, 51820); err != nil {
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

	// Step 6: Start heartbeat loop
	a.wg.Add(1)
	go a.heartbeatLoop(ctx)

	// TODO: Step 7: Configure WireGuard tunnel

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
