package services

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"regexp"
	"strings"
	"sync"
	"time"
)

type WHOISResult struct {
	DomainName string
	Registrar  string
	ExpiryDate time.Time
	WHOISRaw   string
	Error      error
}

type WHOISService struct {
	cache    map[string]*cacheEntry
	cacheMu  sync.RWMutex
	cacheTTL time.Duration
}

type cacheEntry struct {
	result *WHOISResult
	expiry time.Time
}

func NewWHOISService() *WHOISService {
	return &WHOISService{
		cache:    make(map[string]*cacheEntry),
		cacheTTL: 24 * time.Hour,
	}
}

func (s *WHOISService) Lookup(ctx context.Context, domain string) (*WHOISResult, error) {
	domain = strings.ToLower(domain)

	if cached := s.getFromCache(domain); cached != nil {
		return cached, nil
	}

	server, err := s.findWhoisServer(domain)
	if err != nil {
		result := &WHOISResult{
			DomainName: domain,
			Error:      fmt.Errorf("failed to find WHOIS server: %w", err),
		}
		s.setCache(domain, result)
		return result, nil
	}

	result, err := s.queryWhoisServer(ctx, domain, server)
	if err != nil {
		result = &WHOISResult{
			DomainName: domain,
			Error:      err,
		}
	}

	s.setCache(domain, result)
	return result, nil
}

func (s *WHOISService) findWhoisServer(domain string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	dialer := &net.Dialer{}
	conn, err := dialer.DialContext(ctx, "tcp", "whois.iana.org:43")
	if err != nil {
		return "", err
	}
	defer conn.Close()

	fmt.Fprintf(conn, "%s\r\n", domain)
	scanner := bufio.NewScanner(conn)
	server := ""
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "whois:") {
			server = strings.TrimSpace(strings.TrimPrefix(line, "whois:"))
			break
		}
	}
	if err := scanner.Err(); err != nil {
		return "", err
	}

	if server == "" {
		return s.getDefaultServer(domain), nil
	}

	return server, nil
}

func (s *WHOISService) getDefaultServer(domain string) string {
	parts := strings.Split(domain, ".")
	tld := parts[len(parts)-1]

	defaultServers := map[string]string{
		"com":  "whois.verisign-grs.com",
		"net":  "whois.verisign-grs.com",
		"org":  "whois.pir.org",
		"info": "whois.afilias.net",
		"io":   "whois.nic.io",
		"co":   "whois.nic.co",
		"me":   "whois.nic.me",
		"us":   "whois.nic.us",
		"uk":   "whois.nic.uk",
		"de":   "whois.denic.de",
		"fr":   "whois.nic.fr",
		"eu":   "whois.eu",
		"au":   "whois.auda.org.au",
		"ca":   "whois.cira.ca",
		"jp":   "whois.jprs.jp",
		"cn":   "whois.cnnic.cn",
		"ru":   "whois.tcinet.ru",
		"br":   "whois.registro.br",
		"mx":   "whois.mx",
		"es":   "whois.red.es",
		"nl":   "whois.nl",
		"ch":   "whois.nic.ch",
		"at":   "whois.nic.at",
		"be":   "whois.dns.be",
		"se":   "whois.iis.se",
		"no":   "whois.norid.no",
		"dk":   "whois.dk-hostmaster.dk",
		"fi":   "whois.fi",
		"pl":   "whois.dns.pl",
		"cz":   "whois.nic.cz",
		"hu":   "whois.nic.hu",
	}

	if server, ok := defaultServers[tld]; ok {
		return server
	}

	return "whois.iana.org"
}

func (s *WHOISService) queryWhoisServer(ctx context.Context, domain, server string) (*WHOISResult, error) {
	conn, err := net.DialTimeout("tcp", server+":43", 10*time.Second)
	if err != nil {
		return nil, fmt.Errorf("connection failed: %w", err)
	}
	defer conn.Close()

	conn.SetDeadline(time.Now().Add(10 * time.Second))

	query := domain
	if strings.Contains(server, "verisign") {
		query = "=" + domain
	}

	if _, err := fmt.Fprintf(conn, "%s\r\n", query); err != nil {
		return nil, fmt.Errorf("write failed: %w", err)
	}

	var response strings.Builder
	scanner := bufio.NewScanner(conn)
	for scanner.Scan() {
		line := scanner.Text()
		response.WriteString(line)
		response.WriteString("\r\n")
		if strings.HasPrefix(line, ">>>") || strings.HasPrefix(line, "--") {
			break
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("read failed: %w", err)
	}

	raw := response.String()

	result := &WHOISResult{
		DomainName: domain,
		WHOISRaw:   raw,
	}

	result.Registrar = s.extractRegistrar(raw)
	result.ExpiryDate = s.extractExpiryDate(raw)

	return result, nil
}

func (s *WHOISService) extractRegistrar(raw string) string {
	patterns := []string{
		`Registrar:\s*(.+)`,
		`Registrar Name:\s*(.+)`,
		`registrar:\s*(.+)`,
		`Registrar WHOIS Server:\s*(.+)`,
		`Whois Server:\s*(.+)`,
	}

	for _, pattern := range patterns {
		re := regexp.MustCompile(`(?i)` + pattern)
		matches := re.FindStringSubmatch(raw)
		if len(matches) > 1 {
			registrar := strings.TrimSpace(matches[1])
			if registrar != "" {
				return registrar
			}
		}
	}

	return ""
}

func (s *WHOISService) extractExpiryDate(raw string) time.Time {
	patterns := []string{
		`Expiry Date:\s*(\d{4}-\d{2}-\d{2})`,
		`Expiration Date:\s*(\d{4}-\d{2}-\d{2})`,
		`Domain Expiration Date:\s*(\d{4}-\d{2}-\d{2})`,
		`expires:\s*(\d{4}-\d{2}-\d{2})`,
		`Registry Expiry Date:\s*(\d{4}-\d{2}-\d{2})`,
		`Valid Until:\s*(\d{4}-\d{2}-\d{2})`,
		`Expiry date:\s*(\d{2}-\w{3}-\d{4})`,
		`Expiration Date:\s*(\d{2}-\w{3}-\d{4})`,
		`expires:\s*(\d{2}-\w{3}-\d{4})`,
		`Expiry Date:\s*(\d{2}/\d{2}/\d{4})`,
		`Expiration Date:\s*(\d{2}/\d{2}/\d{4})`,
		`Expiry Date:\s*(\d{4}\.\d{2}\.\d{2})`,
	}

	for _, pattern := range patterns {
		re := regexp.MustCompile(`(?i)` + pattern)
		matches := re.FindStringSubmatch(raw)
		if len(matches) > 1 {
			dateStr := strings.TrimSpace(matches[1])
			if dateStr == "" {
				continue
			}

			formats := []string{
				"2006-01-02",
				"02-Jan-2006",
				"02/01/2006",
				"2006.01.02",
			}

			for _, format := range formats {
				if t, err := time.Parse(format, dateStr); err == nil {
					return t
				}
			}
		}
	}

	return time.Time{}
}

func (s *WHOISService) getFromCache(domain string) *WHOISResult {
	s.cacheMu.RLock()
	defer s.cacheMu.RUnlock()

	entry, ok := s.cache[domain]
	if !ok {
		return nil
	}

	if time.Now().Before(entry.expiry) {
		return entry.result
	}

	delete(s.cache, domain)
	return nil
}

func (s *WHOISService) setCache(domain string, result *WHOISResult) {
	s.cacheMu.Lock()
	defer s.cacheMu.Unlock()

	s.cache[domain] = &cacheEntry{
		result: result,
		expiry: time.Now().Add(s.cacheTTL),
	}
}

func (s *WHOISService) ClearCache() {
	s.cacheMu.Lock()
	defer s.cacheMu.Unlock()
	s.cache = make(map[string]*cacheEntry)
}
