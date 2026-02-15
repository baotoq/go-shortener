package enrichment

import (
	"net"

	geoip2 "github.com/oschwald/geoip2-golang"
)

// GeoIPResolver resolves IP addresses to country codes using GeoIP2 database.
type GeoIPResolver struct {
	db *geoip2.Reader
}

// NewGeoIPResolver creates a new GeoIPResolver.
// Returns error if the database file cannot be opened or is corrupt.
func NewGeoIPResolver(dbPath string) (*GeoIPResolver, error) {
	db, err := geoip2.Open(dbPath)
	if err != nil {
		return nil, err
	}
	return &GeoIPResolver{db: db}, nil
}

// Close closes the GeoIP database reader.
func (g *GeoIPResolver) Close() error {
	return g.db.Close()
}

// ResolveCountry returns the ISO country code for the given IP address.
// Returns "Unknown" for private IPs, invalid IPs, or lookup failures.
func (g *GeoIPResolver) ResolveCountry(ipStr string) string {
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return "Unknown"
	}

	record, err := g.db.Country(ip)
	if err != nil {
		return "Unknown"
	}

	if record.Country.IsoCode == "" {
		return "Unknown"
	}

	return record.Country.IsoCode
}
