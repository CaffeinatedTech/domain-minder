# Phase 5: WHOIS Service

## What Should Already Exist

- `/home/adam/projects/domain-minder/internal/services/whois.go` with skeleton WHOISService
- `/home/adam/projects/domain-minder/internal/handlers/domains.go` using the WHOIS service

## Context

This phase implements the WHOIS lookup service that queries WHOIS servers to extract domain registration information including expiry date and registrar. The service handles various WHOIS response formats and includes error handling for common issues like timeouts and rate limiting.

## Agent Rules

1. Create file ONLY in `/home/adam/projects/domain-minder/internal/services/whois.go`
2. Use standard `net` package for TCP connections to WHOIS servers
3. Handle multiple date formats commonly found in WHOIS responses
4. Include proper timeout handling (use context)
5. Handle errors gracefully: domain not found, timeout, rate limiting
6. Return registrar and expiry date when available
7. Cache WHOIS results to avoid excessive queries
8. Do NOT add new dependencies
9. Do NOT create handlers - only the WHOIS service

## Tasks

### 5.1 Implement WHOIS Service

Update `/home/adam/projects/domain-minder/internal/services/whois.go`:

```go
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
    cache      map[string]*cacheEntry
    cacheMu    sync.RWMutex
    cacheTTL   time.Duration
}

type cacheEntry struct {
    result *WHOISResult
    expiry time.Time
}

func NewWHOISService() *WHOISService {
    return &WHOISService{
        cache:      make(map[string]*cacheEntry),
        cacheTTL:   24 * time.Hour,
    }
}

func (s *WHOISService) Lookup(domain string) (*WHOISResult, result *WHOISResult
    domain = strings.ToLower(domain)

    // Check cache first
    if cached := s.getFromCache(domain); cached != nil {
        return cached
    }

    // Find appropriate WHOIS server
    server, err := s.findWhoisServer(domain)
    if err != nil {
        result = &WHOISResult{
            DomainName: domain,
            Error:      fmt.Errorf("failed to find WHOIS server: %w", err),
        }
        s.setCache(domain, result)
        return result
    }

    // Perform WHOIS lookup
    result, err = s.queryWhoisServer(ctx, domain, server)
    if err != nil {
        result = &WHOISResult{
            DomainName: domain,
            Error:      err,
        }
    }

    s.setCache(domain, result)
    return result
}

func (s *WHOISService) findWhoisServer(domain string) (string, error) {
    // Query IANA to find appropriate WHOIS server
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    conn, err := net.DialContext(ctx, "tcp", "whois.iana.org:43")
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
        // Fallback to common TLD servers
        return s.getDefaultServer(domain), nil
    }

    return server, nil
}

func (s *WHOISService) getDefaultServer(domain string) string {
    parts := strings.Split(domain, ".")
    tld := parts[len(parts)-1]

    defaultServers := map[string]string{
        "com":   "whois.verisign-grs.com",
        "net":   "whois.verisign-grs.com",
        "org":   "whois.pir.org",
        "info":  "whois.afilias.net",
        "io":    "whois.nic.io",
        "co":    "whois.nic.co",
        "me":    "whois.nic.me",
        "us":    "whois.nic.us",
        "uk":    "whois.nic.uk",
        "de":    "whois.denic.de",
        "fr":    "whois.nic.fr",
        "eu":    "whois.eu",
        "au":    "whois.auda.org.au",
        "ca":    "whois.cira.ca",
        "jp":    "whois.jprs.jp",
        "cn":    "whois.cnnic.cn",
        "ru":    "whois.tcinet.ru",
        "br":    "whois.registro.br",
        "mx":    "whois.mx",
        "es":    "whois.red.es",
        "nl":    "whois.nl",
        "ch":    "whois.nic.ch",
        "at":    "whois.nic.at",
        "be":    "whois.dns.be",
        "se":    "whois.iis.se",
        "no":    "whois.norid.no",
        "dk":    "whois.dk-hostmaster.dk",
        "fi":    "whois.fi",
        "pl":    "whois.dns.pl",
        "cz":    "whois.nic.cz",
        "hu":    "whois.nic.hu",
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

    // Set deadline
    conn.SetDeadline(time.Now().Add(10 * time.Second))

    // Send query - some servers need domain only, others need "domain=" prefix
    query := domain
    if strings.Contains(server, "verisign") {
        query = "=" + domain
    }

    if _, err := fmt.Fprintf(conn, "%s\r\n", query); err != nil {
        return nil, fmt.Errorf("write failed: %w", err)
    }

    // Read response
    var response strings.Builder
    scanner := bufio.NewScanner(conn)
    for scanner.Scan() {
        line := scanner.Text()
        response.WriteString(line)
        response.WriteString("\r\n")
        // Some servers indicate end with ">>>" or "--"
        if strings.HasPrefix(line, ">>>") || strings.HasPrefix(line, "--") {
            break
        }
    }

    if err := scanner.Err(); err != nil {
        return nil, fmt.Errorf("read failed: %w", err)
    }

    raw := response.String()

    // Parse response
    result := &WHOISResult{
        DomainName: domain,
        WHOISRaw:   raw,
    }

    // Extract registrar
    result.Registrar = s.extractRegistrar(raw)

    // Extract expiry date
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
    }

    for _, pattern := range patterns {
        re := regexp.MustCompile(`(?i)` + pattern)
        matches := re.FindStringSubmatch(raw)
        if len(matches) > 1 {
            dateStr := strings.TrimSpace(matches[1])
            if dateStr == "" {
                continue
            }

            // Try various date formats
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
```

Note: Fix the typo in the Lookup method - remove extra "result *WHOISResult" from the signature.

## Deliverables

- [ ] `/home/adam/projects/domain-minder/internal/services/whois.go` fully implemented
- [ ] WHOIS lookup finds appropriate WHOIS server for TLD
- [ ] Extracts registrar from WHOIS response
- [ ] Extracts expiry date from multiple date formats
- [ ] Caches results for 24 hours
- [ ] Handles timeouts and errors gracefully
- [ ] Returns WHOISRaw for debugging

## Verification

```bash
cd /home/adam/projects/domain-minder

# Build
go build -o domain-minder ./cmd/server/

# Start server
./domain-minder &
sleep 2

# Login
curl -X POST http://localhost:9000/login \
  -d "email=test@example.com&password=test1234" \
  -c cookies.txt -b cookies.txt -L

# Add a domain (WHOIS lookup should work)
curl -X POST http://localhost:9000/domains \
  -d "name=example.com" \
  -b cookies.txt -L

# Check domain was added with registrar and expiry
curl http://localhost:9000/domains -b cookies.txt

# Test another TLD
curl -X POST http://localhost:9000/domains \
  -d "name=google.com" \
  -b cookies.txt -L

# Cleanup
pkill -f domain-minder
rm -f domain_minder.db cookies.txt
```

## Next Phase

After completing verification, proceed to **Phase 6: Dashboard & UI**.
