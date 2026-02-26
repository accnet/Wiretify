package handlers

import (
	"encoding/binary"
	"fmt"
	"net"
	"net/http"
	"os"
	"strings"
	"time"
	"wiretify/internal/config"
	"wiretify/internal/database"
	"wiretify/internal/models"
	"wiretify/internal/services"

	"github.com/labstack/echo/v4"
)

type PeerHandler struct {
	wgSvc  *services.WGService
	netSvc *services.NetworkService
	domSvc *services.DomainService
	cfg    *config.Config
}

func RegisterRoutes(e *echo.Echo, api *echo.Group, wgSvc *services.WGService, netSvc *services.NetworkService, domSvc *services.DomainService, cfg *config.Config) {
	h := &PeerHandler{wgSvc: wgSvc, netSvc: netSvc, domSvc: domSvc, cfg: cfg}

	// Public Routes
	e.GET("/login", h.ShowLogin)
	e.POST("/login", h.PostLogin)
	e.GET("/logout", h.Logout)

	// Auth Middleware for all other routes
	e.Use(h.AuthMiddleware)

	// UI Routes
	e.GET("/", h.RenderDashboard)
	e.GET("/ports", h.RenderPortForwardConfig)
	e.GET("/domains", h.RenderDomains)
	e.GET("/endpoints", h.RenderEndpoints)
	e.GET("/access-control", h.RenderAccessControl)

	// API Peer routes
	api.GET("/peers", h.ListPeers)
	api.POST("/peers", h.CreatePeer)
	api.GET("/peers/:id/config", h.GetPeerConfig)
	api.DELETE("/peers/:id", h.DeletePeer)

	// API Port forward routes
	api.GET("/portforwards", h.ListPortForwards)
	api.POST("/portforwards", h.CreatePortForward)
	api.DELETE("/portforwards/:id", h.DeletePortForward)

	// API Domain routes
	api.GET("/domains", h.ListDomains)
	api.POST("/domains", h.CreateDomain)
	api.DELETE("/domains/:id", h.DeleteDomain)
	api.POST("/domains/:id/verify", h.VerifyDomain)

	// API Endpoint routes
	api.GET("/endpoints", h.ListEndpoints)
	api.POST("/endpoints", h.CreateEndpoint)
	api.DELETE("/endpoints/:id", h.DeleteEndpoint)

	// API Auth
	api.POST("/change-password", h.ChangePassword)
}

func (h *PeerHandler) AuthMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		path := c.Path()
		if path == "/login" || path == "/logout" || strings.HasPrefix(path, "/static") {
			return next(c)
		}

		cookie, err := c.Cookie("session")
		// For simplicity, we compare context to the password. 
		// In production, use a proper session store or JWT.
		if err != nil || cookie.Value != h.cfg.AdminPassword || h.cfg.AdminPassword == "" {
			if strings.HasPrefix(path, "/api/") {
				return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Unauthorized"})
			}
			return c.Redirect(http.StatusFound, "/login")
		}
		return next(c)
	}
}

func (h *PeerHandler) ShowLogin(c echo.Context) error {
	return c.Render(http.StatusOK, "login.html", nil)
}

func (h *PeerHandler) PostLogin(c echo.Context) error {
	password := c.FormValue("password")
	if password == h.cfg.AdminPassword && h.cfg.AdminPassword != "" {
		cookie := &http.Cookie{
			Name:     "session",
			Value:    password,
			Path:     "/",
			HttpOnly: true,
			Expires:  time.Now().Add(24 * time.Hour),
		}
		c.SetCookie(cookie)
		return c.Redirect(http.StatusFound, "/")
	}
	return c.Render(http.StatusUnauthorized, "login.html", map[string]interface{}{"Error": "Invalid password"})
}

func (h *PeerHandler) Logout(c echo.Context) error {
	cookie := &http.Cookie{
		Name:     "session",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		MaxAge:   -1,
	}
	c.SetCookie(cookie)
	return c.Redirect(http.StatusFound, "/login")
}

func (h *PeerHandler) ChangePassword(c echo.Context) error {
	var req struct {
		NewPassword string `json:"new_password"`
	}
	if err := c.Bind(&req); err != nil {
		return err
	}

	newPwd := strings.TrimSpace(req.NewPassword)
	if newPwd == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Password cannot be empty"})
	}

	// Update in-memory config
	h.cfg.AdminPassword = newPwd

	// Update .env file (very basic replacement)
	envPath := ".env"
	data, err := os.ReadFile(envPath)
	if err == nil {
		lines := strings.Split(string(data), "\n")
		found := false
		for i, line := range lines {
			if strings.HasPrefix(line, "ADMIN_PASSWORD=") {
				lines[i] = "ADMIN_PASSWORD=" + newPwd
				found = true
				break
			}
		}
		if !found {
			lines = append(lines, "ADMIN_PASSWORD="+newPwd)
		}
		_ = os.WriteFile(envPath, []byte(strings.Join(lines, "\n")), 0644)
	} else {
		// If .env doesn't exist, create it
		_ = os.WriteFile(envPath, []byte("ADMIN_PASSWORD="+newPwd), 0644)
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "Password updated successfully"})
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
		Icon          string `json:"icon"`
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
		Icon:          req.Icon,
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
	var peer models.Peer
	if err := database.DB.First(&peer, id).Error; err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "Peer not found"})
	}

	// 1. Get Peer IP address (internal) e.g. "10.8.0.3" from "10.8.0.3/32"
	peerIP := peer.AllowedIPs
	if host, _, err := net.SplitHostPort(peer.AllowedIPs); err == nil {
		peerIP = host
	} else if ip, _, err := net.ParseCIDR(peer.AllowedIPs); err == nil {
		peerIP = ip.String()
	}

	// 2. Find and delete all port forwards for this peer
	var pfs []models.PortForward
	database.DB.Where("target_node = ?", peerIP).Find(&pfs)
	for _, pf := range pfs {
		// Remove from kernel
		if err := h.netSvc.RemovePortForward(pf.PublicPort, pf.TargetNode, pf.TargetPort, pf.Protocol); err != nil {
			fmt.Printf("Warning: failed to remove iptables rules for port forward %d during peer deletion: %v\n", pf.PublicPort, err)
		}
		// Remove from DB
		database.DB.Delete(&pf)
	}

	// 3. Delete peer from DB
	if err := database.DB.Delete(&peer).Error; err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	// 4. Sync WireGuard kernel state (this removes the peer from wg device)
	var remainingPeers []models.Peer
	database.DB.Find(&remainingPeers)
	h.wgSvc.SyncPeers(remainingPeers)

	return c.NoContent(http.StatusNoContent)
}

// --- Domain Handlers ---

func (h *PeerHandler) ListDomains(c echo.Context) error {
	var domains []models.Domain
	database.DB.Find(&domains)
	return c.JSON(http.StatusOK, domains)
}

func (h *PeerHandler) CreateDomain(c echo.Context) error {
	var req struct {
		Name string `json:"name"`
	}
	if err := c.Bind(&req); err != nil {
		return err
	}

	domainName := strings.TrimSpace(req.Name)
	if domainName == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Domain name is required"})
	}

	// Tạo mã xác thực ngẫu nhiên
	token := fmt.Sprintf("wiretify-verification-%d", time.Now().UnixNano()/1000000)

	domain := models.Domain{
		Name:              domainName,
		VerificationToken: token,
		Status:            "Pending",
	}

	if err := database.DB.Create(&domain).Error; err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusCreated, domain)
}

func (h *PeerHandler) DeleteDomain(c echo.Context) error {
	id := c.Param("id")
	database.DB.Delete(&models.Domain{}, id)
	return c.NoContent(http.StatusNoContent)
}

func (h *PeerHandler) VerifyDomain(c echo.Context) error {
	id := c.Param("id")
	var domain models.Domain
	if err := database.DB.First(&domain, id).Error; err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "Domain not found"})
	}

	ok, msg, err := h.domSvc.VerifyDomainChecks(domain.Name, domain.VerificationToken)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	if ok {
		now := time.Now()
		domain.Status = "Active"
		domain.LastVerifiedAt = &now
		database.DB.Save(&domain)
		return c.JSON(http.StatusOK, map[string]interface{}{"success": true, "message": msg})
	}

	return c.JSON(http.StatusBadRequest, map[string]interface{}{"success": false, "message": msg})
}

// --- Endpoint Handlers ---

func (h *PeerHandler) ListEndpoints(c echo.Context) error {
	var endpoints []models.Endpoint
	// Preload Peer and Domain info
	database.DB.Preload("Peer").Preload("Domain").Find(&endpoints)
	
	// Format the full addresses for UI convenience
	type resp struct {
		ID          uint   `json:"id"`
		Subdomain   string `json:"subdomain"`
		RootDomain  string `json:"root_domain"`
		PeerName    string `json:"peer_name"`
		FullAddress string `json:"full_address"`
	}
	
	data := make([]resp, len(endpoints))
	for i, ep := range endpoints {
		data[i] = resp{
			ID:          ep.ID,
			Subdomain:   ep.Subdomain,
			RootDomain:  ep.Domain.Name,
			PeerName:    ep.Peer.Name,
			FullAddress: fmt.Sprintf("%s.%s", ep.Subdomain, ep.Domain.Name),
		}
	}
	
	return c.JSON(http.StatusOK, data)
}

func (h *PeerHandler) CreateEndpoint(c echo.Context) error {
	var req struct {
		PeerID    uint   `json:"peer_id"`
		DomainID  uint   `json:"domain_id"`
		Subdomain string `json:"subdomain"`
	}
	if err := c.Bind(&req); err != nil {
		return err
	}

	subdomain := strings.TrimSpace(req.Subdomain)
	if subdomain == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Subdomain is required"})
	}

	// 1. Kiểm tra domain xem có active chưa
	var domain models.Domain
	if err := database.DB.First(&domain, req.DomainID).Error; err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Root domain not found"})
	}
	if domain.Status != "Active" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Root domain must be Active to create endpoints"})
	}

	// 2. Kiểm tra trùng lặp subdomain trên cùng domain
	var existing models.Endpoint
	if err := database.DB.Where("domain_id = ? AND subdomain = ?", req.DomainID, subdomain).First(&existing).Error; err == nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Subdomain already exists for this domain"})
	}

	endpoint := models.Endpoint{
		PeerID:    req.PeerID,
		DomainID:  req.DomainID,
		Subdomain: subdomain,
	}

	if err := database.DB.Create(&endpoint).Error; err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusCreated, endpoint)
}

func (h *PeerHandler) DeleteEndpoint(c echo.Context) error {
	id := c.Param("id")
	database.DB.Delete(&models.Endpoint{}, id)
	return c.NoContent(http.StatusNoContent)
}

// --- Port Forwarding Handlers ---

func (h *PeerHandler) ListPortForwards(c echo.Context) error {
	var pfs []models.PortForward
	if err := database.DB.Find(&pfs).Error; err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, pfs)
}

func (h *PeerHandler) CreatePortForward(c echo.Context) error {
	var req struct {
		PublicPort int    `json:"public_port"`
		TargetNode string `json:"target_node"`
		TargetPort int    `json:"target_port"`
		Protocol   string `json:"protocol"`
	}
	if err := c.Bind(&req); err != nil {
		return err
	}

	// 1. Check if public port is already in use for this protocol
	var existing models.PortForward
	if err := database.DB.Where("public_port = ? AND protocol = ?", req.PublicPort, req.Protocol).First(&existing).Error; err == nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": fmt.Sprintf("Public port %d (%s) is already in use", req.PublicPort, req.Protocol)})
	}

	// 2. Optional: Verify target node IP exists in our peers list
	var peer models.Peer
	if err := database.DB.Where("allowed_ips LIKE ?", req.TargetNode+"/%").First(&peer).Error; err != nil {
		// Log but allow (maybe it's a manual entry)
	}

	pf := models.PortForward{
		PublicPort: req.PublicPort,
		TargetNode: req.TargetNode,
		TargetPort: req.TargetPort,
		Protocol:   req.Protocol,
	}

	if err := database.DB.Create(&pf).Error; err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	err := h.netSvc.AddPortForward(pf.PublicPort, pf.TargetNode, pf.TargetPort, pf.Protocol)
	if err != nil {
		database.DB.Unscoped().Delete(&pf) // hard delete if kernel logic fails to avoid DB pollution
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("failed to configure iptables: %v", err)})
	}

	return c.JSON(http.StatusCreated, pf)
}

func (h *PeerHandler) DeletePortForward(c echo.Context) error {
	id := c.Param("id")
	var pf models.PortForward
	if err := database.DB.First(&pf, id).Error; err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "Port forward not found"})
	}

	if err := h.netSvc.RemovePortForward(pf.PublicPort, pf.TargetNode, pf.TargetPort, pf.Protocol); err != nil {
		// Log warning but continue deletion
		fmt.Printf("Warning: failed to remove iptables rules for port forward %d: %v\n", pf.PublicPort, err)
	}

	database.DB.Delete(&pf)
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

// --- Front-end Page renderers ---

func (h *PeerHandler) RenderDashboard(c echo.Context) error {
	return h.RenderDashboardPage(c)
}

func (h *PeerHandler) RenderDashboardPage(c echo.Context) error {
	return c.Render(http.StatusOK, "dashboard.html", map[string]interface{}{
		"CurrentPage": "dashboard",
	})
}

func (h *PeerHandler) RenderPortForwardConfig(c echo.Context) error {
	_, endpoint, _ := h.wgSvc.GetServerConfig()
	return c.Render(http.StatusOK, "port_forward.html", map[string]interface{}{
		"CurrentPage": "ports",
		"PublicIP":    endpoint,
	})
}

func (h *PeerHandler) RenderDomains(c echo.Context) error {
	return c.Render(http.StatusOK, "domains.html", map[string]interface{}{
		"CurrentPage": "domains",
	})
}

func (h *PeerHandler) RenderEndpoints(c echo.Context) error {
	return c.Render(http.StatusOK, "endpoints.html", map[string]interface{}{
		"CurrentPage": "endpoints",
	})
}

func (h *PeerHandler) RenderAccessControl(c echo.Context) error {
	return c.Render(http.StatusOK, "access_control.html", map[string]interface{}{
		"CurrentPage": "access_control",
	})
}
