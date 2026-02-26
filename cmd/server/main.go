package main

import (
	"log"
	"wiretify/internal/config"
	"wiretify/internal/database"
	"wiretify/internal/handlers"
	"wiretify/internal/models"
	"wiretify/internal/services"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

func main() {
	// 1. Load Config
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// 2. Init DB
	if err := database.InitDB(cfg.DatabasePath); err != nil {
		log.Fatalf("Failed to init DB: %v", err)
	}

	// 3. Setup Network (Interface & NAT)
	netSvc := services.NewNetworkService(cfg)
	if err := netSvc.SetupInterface(); err != nil {
		log.Printf("Warning: Interface setup failed: %v (May require root/NET_ADMIN)", err)
	}
	if err := netSvc.SetupFirewall(); err != nil {
		log.Printf("Warning: Firewall setup failed: %v", err)
	}

	// Restore Port Forwards from DB
	var activePortForwards []models.PortForward
	database.DB.Find(&activePortForwards)
	for _, pf := range activePortForwards {
		if err := netSvc.AddPortForward(pf.PublicPort, pf.TargetNode, pf.TargetPort, pf.Protocol); err != nil {
			log.Printf("Warning: Failed to restore port forward %d -> %s:%d : %v", pf.PublicPort, pf.TargetNode, pf.TargetPort, err)
		}
	}

	// 4. WG Sync
	wgSvc, err := services.NewWGService(cfg)
	if err != nil {
		log.Printf("Warning: WireGuard controller failed to init: %v", err)
	} else {
		defer wgSvc.Close()
		// Sync initial peers from DB
		var peers []models.Peer
		database.DB.Find(&peers)
		if err := wgSvc.SyncPeers(peers); err != nil {
			log.Printf("Warning: Failed to sync initial peers to wg: %v", err)
		}
	}

	// 5. API Server
	e := echo.New()
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	// Static files for Web UI
	e.Static("/", "web")

	// API Routes
	api := e.Group("/api")
	handlers.RegisterRoutes(api, wgSvc, netSvc)

	log.Printf("Wiretify starting on :8080...")
	e.Logger.Fatal(e.Start(":8080"))
}
