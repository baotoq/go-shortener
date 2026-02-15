package enrichment

import (
	ua "github.com/mileusna/useragent"
)

// DeviceDetector detects device type from User-Agent strings.
type DeviceDetector struct{}

// NewDeviceDetector creates a new DeviceDetector.
func NewDeviceDetector() *DeviceDetector {
	return &DeviceDetector{}
}

// DetectDevice returns the device type from a User-Agent string.
// Returns "Desktop", "Mobile", "Tablet", "Bot", or "Unknown".
func (d *DeviceDetector) DetectDevice(uaString string) string {
	if uaString == "" {
		return "Unknown"
	}

	parsed := ua.Parse(uaString)

	// Check bot first (per user decision: bots tracked separately)
	if parsed.Bot {
		return "Bot"
	}

	if parsed.Tablet {
		return "Tablet"
	}

	if parsed.Mobile {
		return "Mobile"
	}

	if parsed.Desktop {
		return "Desktop"
	}

	return "Unknown"
}
