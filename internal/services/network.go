package services

import (
	"fmt"
	"log"
	"os/exec"
	"wiretify/internal/config"

	"github.com/coreos/go-iptables/iptables"
	"github.com/vishvananda/netlink"
)

type NetworkService struct {
	cfg *config.Config
}

func NewNetworkService(cfg *config.Config) *NetworkService {
	return &NetworkService{cfg: cfg}
}

func (s *NetworkService) SetupInterface() error {
	linkName := s.cfg.InterfaceName
	la := netlink.NewLinkAttrs()
	la.Name = linkName

	// Kiểm tra xem interface đã tồn tại chưa
	if link, err := netlink.LinkByName(linkName); err == nil {
		log.Printf("Interface %s already exists, deleting for clean state...", linkName)
		if err := netlink.LinkDel(link); err != nil {
			return fmt.Errorf("failed to delete existing interface: %v", err)
		}
	}

	// Tạo interface loại wireguard
	wgLink := &netlink.GenericLink{
		LinkAttrs: la,
		LinkType:  "wireguard",
	}

	if err := netlink.LinkAdd(wgLink); err != nil {
		return fmt.Errorf("failed to add wireguard interface: %v", err)
	}

	// Gán IP
	addr, err := netlink.ParseAddr(s.cfg.Address)
	if err != nil {
		return fmt.Errorf("invalid address %s: %v", s.cfg.Address, err)
	}

	if err := netlink.AddrAdd(wgLink, addr); err != nil {
		return fmt.Errorf("failed to add address to %s: %v", linkName, err)
	}

	// Bring up
	if err := netlink.LinkSetUp(wgLink); err != nil {
		return fmt.Errorf("failed to set %s UP: %v", linkName, err)
	}

	log.Printf("Interface %s initialized with address %s", linkName, s.cfg.Address)
	return nil
}

func (s *NetworkService) SetupFirewall() error {
	ipt, err := iptables.New()
	if err != nil {
		return err
	}

	// Bật IP forwarding
	cmd := exec.Command("sysctl", "-w", "net.ipv4.ip_forward=1")
	if err := cmd.Run(); err != nil {
		log.Printf("Warning: failed to enable IP forwarding: %v", err)
	}

	// NAT Masquerade (ví dụ cho eth0, bạn nên detect interface mạng chính)
	// Để đơn giản, ta apply cho toàn bộ traffic từ VPN pool
	err = ipt.AppendUnique("nat", "POSTROUTING", "-s", s.cfg.Address, "-j", "MASQUERADE")
	if err != nil {
		return fmt.Errorf("failed to setup NAT: %v", err)
	}

	log.Println("Firewall rules (NAT) applied")
	return nil
}
