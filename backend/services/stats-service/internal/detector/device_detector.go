package detector

import "github.com/mileusna/useragent"

func DetectDevice(ua string) string {
	device := useragent.Parse(ua)

	if device.Mobile {
		return "mobile"
	}

	if device.Desktop {
		return "desktop"
	}

	if device.Tablet {
		return "tablet"
	}

	if device.Bot {
		return "unknown"
	}

	return "unknown"
}
