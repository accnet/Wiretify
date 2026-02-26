package services

import (
	"fmt"
	"log"
	"net"
	"os"
	"wiretify/internal/config"
	"wiretify/internal/models"

	"golang.zx2c4.com/wireguard/wgctrl"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
)

type WGService struct {
	client *wgctrl.Client
	cfg    *config.Config
}

func NewWGService(cfg *config.Config) (*WGService, error) {
	client, err := wgctrl.New()
	if err != nil {
		return nil, err
	}

	if cfg.PrivateKey == "" {
		priv, err := wgtypes.GeneratePrivateKey()
		if err == nil {
			cfg.PrivateKey = priv.String()
			log.Printf("Generated initial Server Private Key.")
			// Ghi vào file để dùng lại
			f, err := os.OpenFile(".env", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
			if err == nil {
				defer f.Close()
				f.WriteString(fmt.Sprintf("\nWG_PRIVATE_KEY=%s\n", cfg.PrivateKey))
			} else {
				log.Printf("Failed to write to .env: %v", err)
			}
		}
	}

	return &WGService{client: client, cfg: cfg}, nil
}

func (s *WGService) Close() {
	s.client.Close()
}

func (s *WGService) SyncPeers(peers []models.Peer) error {
	privKey, err := wgtypes.ParseKey(s.cfg.PrivateKey)
	if err != nil {
		return fmt.Errorf("failed to parse server private key: %v", err)
	}

	wgConfig := wgtypes.Config{
		PrivateKey:   &privKey,
		ListenPort:   &s.cfg.Port,
		ReplacePeers: true, // Replace all peers with the ones from DB
		Peers:        make([]wgtypes.PeerConfig, 0, len(peers)),
	}

	for _, p := range peers {
		pubKey, err := wgtypes.ParseKey(p.PublicKey)
		if err != nil {
			log.Printf("Skip invalid peer %s public key: %v", p.Name, err)
			continue
		}

		_, ipNet, err := net.ParseCIDR(p.AllowedIPs)
		if err != nil {
			log.Printf("Skip invalid peer %s allowed IPs: %v", p.Name, err)
			continue
		}

		wgConfig.Peers = append(wgConfig.Peers, wgtypes.PeerConfig{
			PublicKey:         pubKey,
			Remove:            !p.Enabled,
			ReplaceAllowedIPs: true,
			AllowedIPs:        []net.IPNet{*ipNet},
		})
	}

	if err := s.client.ConfigureDevice(s.cfg.InterfaceName, wgConfig); err != nil {
		return fmt.Errorf("failed to configure device %s: %v", s.cfg.InterfaceName, err)
	}

	log.Printf("Synchronized %d peers to %s", len(peers), s.cfg.InterfaceName)
	return nil
}

func (s *WGService) GenerateKeyPair() (string, string, error) {
	priv, err := wgtypes.GeneratePrivateKey()
	if err != nil {
		return "", "", err
	}
	return priv.String(), priv.PublicKey().String(), nil
}

func (s *WGService) GetServerConfig() (string, string, int) {
	privKey, err := wgtypes.ParseKey(s.cfg.PrivateKey)
	var pubKey string
	if err == nil {
		pubKey = privKey.PublicKey().String()
	} else {
		// Trả về tạm thời nếu server chưa sinh key
		pubKey = "SERVER_PUBLIC_KEY_NOT_SET"
	}
	return pubKey, s.cfg.ServerEndpoint, s.cfg.Port
}

func (s *WGService) GetServerAddress() string {
	return s.cfg.Address
}
