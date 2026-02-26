package handlers

import (
	"encoding/binary"
	"fmt"
	"net"
	"net/http"
	"time"
	"wiretify/internal/database"
	"wiretify/internal/models"
	"wiretify/internal/services"

	"github.com/labstack/echo/v4"
)

type PeerHandler struct {
	wgSvc *services.WGService
}

func RegisterRoutes(e *echo.Group, wgSvc *services.WGService) {
	h := &PeerHandler{wgSvc: wgSvc}

	e.GET("/peers", h.ListPeers)
	e.POST("/peers", h.CreatePeer)
	e.GET("/peers/:id/config", h.GetPeerConfig)
	e.DELETE("/peers/:id", h.DeletePeer)
}

func (h *PeerHandler) ListPeers(c echo.Context) error {
	var peers []models.Peer
	if err := database.DB.Find(&peers).Error; err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	// Fetch active stats from WireGuard kernel
	wgPeers, err := h.wgSvc.GetDevicePeers()
	if err == nil {
		for i := range peers {
			if wgp, ok := wgPeers[peers[i].PublicKey]; ok {
				peers[i].RxBytes = wgp.ReceiveBytes
				peers[i].TxBytes = wgp.TransmitBytes
				peers[i].LastHandshake = wgp.LastHandshakeTime

				// A peer is considered "Connected" if the last handshake was within the last 3 minutes
				if !wgp.LastHandshakeTime.IsZero() && time.Since(wgp.LastHandshakeTime) < 3*time.Minute {
					peers[i].Connected = true
				}
			}
		}
	}

	return c.JSON(http.StatusOK, peers)
}

func (h *PeerHandler) CreatePeer(c echo.Context) error {
	var req struct {
		Name          string `json:"name"`
		UseAsExitNode bool   `json:"use_as_exit_node"`
	}
	if err := c.Bind(&req); err != nil {
		return err
	}

	// Calculate next available IP
	var allPeers []models.Peer
	database.DB.Find(&allPeers)

	serverAddr := h.wgSvc.GetServerAddress()
	nextIP, err := allocateNextIP(serverAddr, allPeers)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "IP Allocation failed: " + err.Error()})
	}

	// Generate keys if not provided (Simplification: app always gens for convenience)
	priv, pub, err := h.wgSvc.GenerateKeyPair()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to generate keys"})
	}

	peer := models.Peer{
		Name:          req.Name,
		PublicKey:     pub,
		PrivateKey:    priv,
		AllowedIPs:    nextIP,
		UseAsExitNode: req.UseAsExitNode,
		Enabled:       true,
	}

	if err := database.DB.Create(&peer).Error; err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	// Sync to kernel (reload peers from DB to include the new one)
	database.DB.Find(&allPeers)
	h.wgSvc.SyncPeers(allPeers)

	return c.JSON(http.StatusCreated, peer)
}

func (h *PeerHandler) DeletePeer(c echo.Context) error {
	id := c.Param("id")
	if err := database.DB.Delete(&models.Peer{}, id).Error; err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	// Sync to kernel
	var allPeers []models.Peer
	database.DB.Find(&allPeers)
	h.wgSvc.SyncPeers(allPeers)

	return c.NoContent(http.StatusNoContent)
}

func (h *PeerHandler) GetPeerConfig(c echo.Context) error {
	id := c.Param("id")
	var peer models.Peer
	if err := database.DB.First(&peer, id).Error; err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "Peer not found"})
	}

	serverPubKey, endpoint, port := h.wgSvc.GetServerConfig()

	// Logic chia /32 thành IP Address cho Interface
	// Nếu AllowedIPs là `10.8.0.2/32`, ta có thể dùng trực tiếp hoặc chuyển thành `/24` tùy network design.
	// Ở đây WireGuard client cài đặt Address cũng dùng dạng CIDR, nên dùng trực tiếp AllowedIPs là ok.

	// Routing AllowedIPs for client config
	allowedIPsClient := "0.0.0.0/0, ::/0"
	if peer.UseAsExitNode {
		// Use server IP network as subnet, e.g 10.8.0.0/24
		serverAddr := h.wgSvc.GetServerAddress()
		ip, ipnet, err := net.ParseCIDR(serverAddr)
		if err == nil {
			allowedIPsClient = ipnet.String()
		} else {
			// Fallback
			allowedIPsClient = fmt.Sprintf("%s/24", ip.String())
		}
	}

	configTpl := `[Interface]
PrivateKey = %s
Address = %s

[Peer]
PublicKey = %s
Endpoint = %s:%d
AllowedIPs = %s
PersistentKeepalive = 25
`
	confStr := fmt.Sprintf(configTpl, peer.PrivateKey, peer.AllowedIPs, serverPubKey, endpoint, port, allowedIPsClient)

	// Set header for file download
	c.Response().Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s.conf", peer.Name))
	c.Response().Header().Set("Content-Type", "application/x-wireguard-profile")

	return c.String(http.StatusOK, confStr)
}

// allocateNextIP tìm IP khả dụng tiếp theo trong subnet của server
func allocateNextIP(baseCIDR string, peers []models.Peer) (string, error) {
	ip, ipnet, err := net.ParseCIDR(baseCIDR)
	if err != nil {
		return "", err
	}

	ip = ip.To4()
	if ip == nil {
		return "", fmt.Errorf("only IPv4 is supported")
	}

	startIP := binary.BigEndian.Uint32(ip)
	mask := binary.BigEndian.Uint32(ipnet.Mask)

	usedIPs := make(map[uint32]bool)
	usedIPs[startIP] = true // Đánh dấu Server IP đã sử dụng

	for _, p := range peers {
		pip, _, err := net.ParseCIDR(p.AllowedIPs)
		if err == nil {
			pip = pip.To4()
			if pip != nil {
				usedIPs[binary.BigEndian.Uint32(pip)] = true
			}
		}
	}

	network := startIP & mask
	broadcast := network | ^mask

	for i := network + 1; i < broadcast; i++ {
		if !usedIPs[i] {
			nextIP := make(net.IP, 4)
			binary.BigEndian.PutUint32(nextIP, i)
			return fmt.Sprintf("%s/32", nextIP.String()), nil
		}
	}

	return "", fmt.Errorf("no available IPs in subnet")
}
