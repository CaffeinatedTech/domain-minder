package services

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/likexian/whois"
	whoisparser "github.com/likexian/whois-parser"
)

type WHOISResult struct {
	DomainName    string
	Registrar     string
	ExpiryDate    time.Time
	ExpiryMissing bool // true if expiry date couldn't be determined from WHOIS
	WHOISRaw      string
	Error         error
}

type WHOISService struct {
	cache    map[string]*cacheEntry
	cacheMu  sync.RWMutex
	cacheTTL time.Duration
	client   *http.Client
}

type cacheEntry struct {
	result *WHOISResult
	expiry time.Time
}

func NewWHOISService() *WHOISService {
	return &WHOISService{
		cache:    make(map[string]*cacheEntry),
		cacheTTL: 24 * time.Hour,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (s *WHOISService) Lookup(ctx context.Context, domain string) (*WHOISResult, error) {
	domain = strings.ToLower(domain)

	if cached := s.getFromCache(domain); cached != nil {
		return cached, nil
	}

	result := &WHOISResult{
		DomainName: domain,
	}

	parts := strings.Split(domain, ".")
	tld := parts[len(parts)-1]

	rdapData, err := s.lookupRDAP(domain, tld)
	if err == nil && rdapData != nil {
		fmt.Printf("WHOIS: Using RDAP for %s\n", domain)
		result.WHOISRaw = formatRDAPForDisplay(rdapData)
		result.Registrar = rdapData["registrar_name"]
		if exp, ok := rdapData["expiry_date"]; ok && exp != "" {
			result.ExpiryDate, _ = time.Parse("2006-01-02", exp)
		}
		s.setCache(domain, result)
		return result, nil
	}
	fmt.Printf("WHOIS: RDAP failed for %s: %v, trying WHOIS\n", domain, err)

	raw, err := s.lookupWhoisServer(domain, tld)
	if err != nil {
		result.Error = fmt.Errorf("whois lookup failed: %w", err)
		s.setCache(domain, result)
		return result, nil
	}

	result.WHOISRaw = raw

	parsed, err := whoisparser.Parse(raw)
	if err != nil {
		result.Error = fmt.Errorf("whois parse failed: %w", err)
		s.setCache(domain, result)
		return result, nil
	}

	if parsed.Registrar != nil && parsed.Registrar.Name != "" {
		result.Registrar = parsed.Registrar.Name
	}

	if parsed.Domain.ExpirationDate != "" {
		formats := []string{
			time.RFC3339,
			"2006-01-02T15:04:05Z",
			"2006-01-02 15:04:05",
			"2006-01-02",
			"02-Jan-2006",
			"January 2, 2006",
			"02/01/2006",
			"2006/01/02",
			"01/02/2006",
			"2006-01-02T15:04:05-07:00",
			"2006-01-02T15:04:05.000Z",
		}

		for _, format := range formats {
			expiry, err := time.Parse(format, parsed.Domain.ExpirationDate)
			if err == nil {
				result.ExpiryDate = expiry
				break
			}
		}
	}

	// Mark if expiry date couldn't be determined
	if result.ExpiryDate.IsZero() {
		result.ExpiryMissing = true
	}

	s.setCache(domain, result)
	return result, nil
}

func (s *WHOISService) lookupRDAP(domain, tld string) (map[string]string, error) {
	rdapURL := fmt.Sprintf("https://rdap.org/domain/%s", domain)
	fmt.Printf("WHOIS: Trying RDAP %s\n", rdapURL)

	req, err := http.NewRequest("GET", rdapURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")

	resp, err := s.client.Do(req.WithContext(context.Background()))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	fmt.Printf("WHOIS: RDAP response status: %d\n", resp.StatusCode)

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("RDAP returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var data map[string]interface{}
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, err
	}

	result := make(map[string]string)

	if registrar := extractRDAPRegistrar(data); registrar != "" {
		result["registrar_name"] = registrar
	}

	if events, ok := data["events"].([]interface{}); ok {
		for _, e := range events {
			if event, ok := e.(map[string]interface{}); ok {
				if eventName, ok := event["eventAction"].(string); ok {
					if eventDate, ok := event["eventDate"].(string); ok {
						if eventName == "expiration" {
							result["expiry_date"] = eventDate[:10]
							break
						}
					}
				}
			}
		}
	}

	if result["registrar_name"] == "" && result["expiry_date"] == "" {
		return nil, fmt.Errorf("no useful data in RDAP response")
	}

	return result, nil
}

func formatRDAPForDisplay(data map[string]string) string {
	var builder strings.Builder
	if data["registrar_name"] != "" {
		builder.WriteString("Registrar: ")
		builder.WriteString(data["registrar_name"])
		builder.WriteString("\n")
	}
	if data["expiry_date"] != "" {
		builder.WriteString("Expiry Date: ")
		builder.WriteString(data["expiry_date"])
		builder.WriteString("\n")
	}
	return builder.String()
}

func extractRDAPRegistrar(data map[string]interface{}) string {
	if entities, ok := data["entities"].([]interface{}); ok {
		for _, e := range entities {
			if ent, ok := e.(map[string]interface{}); ok {
				if roles, ok := ent["roles"].([]interface{}); ok {
					for _, role := range roles {
						if role == "registrar" {
							if vcard, ok := ent["vcardArray"].([]interface{}); ok {
								if len(vcard) > 1 {
									if items, ok := vcard[1].([]interface{}); ok {
										for _, item := range items {
											if row, ok := item.([]interface{}); ok {
												if len(row) > 3 {
													if row[0] == "fn" {
														if name, ok := row[3].(string); ok && name != "" {
															return name
														}
													}
												}
											}
										}
									}
								}
							}
						}
					}
				}
			}
		}
	}
	return ""
}

func (s *WHOISService) lookupWhoisServer(domain, tld string) (string, error) {
	fmt.Printf("WHOIS: Trying WHOIS lookup for TLD %s\n", tld)

	ext := tld

	result, err := whois.Whois(ext)
	if err != nil {
		fmt.Printf("WHOIS: IANA lookup for %s failed: %v\n", ext, err)
	} else {
		server := extractWhoisServer(result)
		fmt.Printf("WHOIS: IANA returned server: %s\n", server)
		if server != "" {
			return s.queryWithReferral(domain, server)
		}
	}

	return "", fmt.Errorf("could not find WHOIS server for %s", tld)
}

func (s *WHOISService) queryWithReferral(domain, server string) (string, error) {
	fmt.Printf("WHOIS: Querying %s on %s\n", domain, server)

	result, err := whois.Whois(domain, server)
	if err != nil {
		return "", err
	}

	refServer := extractWhoisServer(result)
	fmt.Printf("WHOIS: Referral server: %s\n", refServer)

	if refServer != "" && refServer != server {
		fmt.Printf("WHOIS: Following referral to %s\n", refServer)
		refResult, err := whois.Whois(domain, refServer)
		if err == nil {
			return refResult, nil
		}
	}

	return result, nil
}

func extractWhoisServer(data string) string {
	tokens := []string{
		"Registrar WHOIS Server: ",
		"Whois Server: ",
		"whois: ",
	}

	for _, token := range tokens {
		start := strings.Index(data, token)
		if start != -1 {
			start += len(token)
			end := strings.Index(data[start:], "\n")
			if end == -1 {
				end = len(data) - start
			}
			server := strings.TrimSpace(data[start : start+end])
			server = strings.TrimPrefix(server, "http://")
			server = strings.TrimPrefix(server, "https://")
			server = strings.TrimPrefix(server, "whois://")
			if server != "" {
				return server
			}
		}
	}

	return ""
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
