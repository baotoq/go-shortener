package mqs

import (
	"context"
	"encoding/json"
	"net"
	"strings"
	"time"

	"go-shortener/common/events"
	"go-shortener/services/analytics-consumer/internal/svc"
	"go-shortener/services/analytics-rpc/model"

	"github.com/google/uuid"
	"github.com/mssola/useragent"
	"github.com/zeromicro/go-zero/core/logx"
)

type ClickEventConsumer struct {
	svcCtx *svc.ServiceContext
}

func NewClickEventConsumer(ctx context.Context, svcCtx *svc.ServiceContext) *ClickEventConsumer {
	return &ClickEventConsumer{
		svcCtx: svcCtx,
	}
}

func (c *ClickEventConsumer) Consume(ctx context.Context, key, val string) error {
	logx.WithContext(ctx).Infof("ClickEventConsumer received: key=%s", key)

	var event events.ClickEvent
	if err := json.Unmarshal([]byte(val), &event); err != nil {
		logx.WithContext(ctx).Errorf("failed to unmarshal click event: %v", err)
		return nil // Don't retry malformed messages
	}

	// Enrich with GeoIP
	countryCode := resolveCountry(c.svcCtx, event.IP)

	// Enrich with device type from User-Agent
	deviceType := resolveDeviceType(event.UserAgent)

	// Enrich with traffic source from Referer
	trafficSource := resolveTrafficSource(event.Referer)

	// Generate UUIDv7 for the click record
	id, err := uuid.NewV7()
	if err != nil {
		logx.WithContext(ctx).Errorf("failed to generate UUIDv7: %v", err)
		return err
	}

	clickedAt := time.Unix(event.Timestamp, 0)

	_, insertErr := c.svcCtx.ClickModel.Insert(ctx, &model.Clicks{
		Id:            id.String(),
		ShortCode:     event.ShortCode,
		ClickedAt:     clickedAt,
		CountryCode:   countryCode,
		DeviceType:    deviceType,
		TrafficSource: trafficSource,
	})
	if insertErr != nil {
		// Check for duplicate key (idempotent handling)
		if strings.Contains(insertErr.Error(), "duplicate key") {
			logx.WithContext(ctx).Infof("duplicate click event, skipping: short_code=%s", event.ShortCode)
			return nil
		}
		logx.WithContext(ctx).Errorf("failed to insert click: %v", insertErr)
		return insertErr
	}

	logx.WithContext(ctx).Infow("click event processed",
		logx.Field("short_code", event.ShortCode),
		logx.Field("country", countryCode),
		logx.Field("device", deviceType),
		logx.Field("source", trafficSource),
	)

	return nil
}

// resolveCountry looks up the country code from IP using GeoIP database.
// Falls back to "XX" if GeoIP is unavailable or lookup fails.
func resolveCountry(svcCtx *svc.ServiceContext, ip string) string {
	if svcCtx.GeoDB == nil || ip == "" {
		return "XX"
	}

	parsedIP := net.ParseIP(ip)
	if parsedIP == nil {
		return "XX"
	}

	record, err := svcCtx.GeoDB.Country(parsedIP)
	if err != nil {
		return "XX"
	}

	code := record.Country.IsoCode
	if code == "" {
		return "XX"
	}

	return code
}

// resolveDeviceType parses the User-Agent string to determine device type.
func resolveDeviceType(userAgent string) string {
	if userAgent == "" {
		return "Unknown"
	}

	ua := useragent.New(userAgent)

	if ua.Bot() {
		return "Bot"
	}

	if ua.Mobile() {
		return "Mobile"
	}

	// useragent library doesn't have Tablet detection, treat as Desktop
	return "Desktop"
}

// resolveTrafficSource categorizes the referer into traffic source types.
func resolveTrafficSource(referer string) string {
	if referer == "" {
		return "Direct"
	}

	refLower := strings.ToLower(referer)

	// Search engines
	searchEngines := []string{"google.", "bing.", "yahoo.", "duckduckgo.", "baidu.", "yandex."}
	for _, engine := range searchEngines {
		if strings.Contains(refLower, engine) {
			return "Search"
		}
	}

	// Social media
	socialNetworks := []string{"facebook.", "twitter.", "t.co", "linkedin.", "reddit.", "instagram.", "youtube.", "tiktok."}
	for _, social := range socialNetworks {
		if strings.Contains(refLower, social) {
			return "Social"
		}
	}

	return "Referral"
}
