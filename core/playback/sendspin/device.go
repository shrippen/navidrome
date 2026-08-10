package sendspin

import "strings"

// DeviceName is the Jukebox.Devices device identifier that selects the
// Sendspin playback backend instead of MPV.
const DeviceName = "sendspin"

// IsDevice reports whether the jukebox device name should use Sendspin.
func IsDevice(deviceName string) bool {
	return strings.EqualFold(strings.TrimSpace(deviceName), DeviceName)
}
