package services

import (
	"context"
	"fmt"
	"net"
	"strings"
	"time"

	"wiretify/internal/config"
)

type DomainService struct {
	cfg *config.Config
}

func NewDomainService(cfg *config.Config) *DomainService {
	return &DomainService{cfg: cfg}
}

// VerifyDomainChecks thực hiện kiểm tra bản ghi A wildcard và TXT verification
func (s *DomainService) VerifyDomainChecks(domain, token string) (bool, string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resolver := &net.Resolver{}

	// 1. Kiểm tra bản ghi TXT: _acme-challenge.domain -> token
	txtHost := fmt.Sprintf("_acme-challenge.%s", domain)
	txtRecords, err := resolver.LookupTXT(ctx, txtHost)
	if err != nil {
		return false, "TXT record not found for " + txtHost, nil
	}
	
	foundTXT := false
	for _, rec := range txtRecords {
		if strings.TrimSpace(rec) == token {
			foundTXT = true
			break
		}
	}
	if !foundTXT {
		return false, fmt.Sprintf("TXT record found but value does not match (Found: %s, Expected: %s)", strings.Join(txtRecords, ", "), token), nil
	}

	// 2. Kiểm tra bản ghi A wildcard: *.domain -> VPS IP
	// Ta sẽ resolve chính subdomain wildcard thử nghiệm: wiretify-test.domain
	wildcardSub := fmt.Sprintf("wiretify-test.%s", domain)
	ips, err := resolver.LookupIPAddr(ctx, wildcardSub)
	if err != nil {
		return false, "A record (wildcard) not found for " + wildcardSub, nil
	}

	foundIP := false
	for _, ip := range ips {
		// So sánh với server endpoint (IP VPS)
		if ip.String() == s.cfg.ServerEndpoint {
			foundIP = true
			break
		}
	}

	if !foundIP {
		return false, fmt.Sprintf("Wildcard A record points to wrong IP (Found: %v, Expected: %s)", ips, s.cfg.ServerEndpoint), nil
	}

	return true, "Domain verified successfully!", nil
}
