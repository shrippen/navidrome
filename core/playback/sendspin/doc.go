// Package sendspin implements a Jukebox playback backend that streams audio to
// Sendspin clients (https://www.sendspin-audio.com/) with clock-synchronized
// multi-room playback.
//
// Select this backend by setting a Jukebox device's second field to "sendspin",
// or by enabling Jukebox.Sendspin.Enabled (which auto-registers a sendspin device).
//
// Example configuration:
//
//	[Jukebox]
//	Enabled = true
//	Default = "House"
//	Devices = [["House", "sendspin"]]
//
//	[Jukebox.Sendspin]
//	Enabled = true
//	Port = 8927
//	Name = "Navidrome"
//	EnableMDNS = true
//
// Requires ffmpeg (same as the rest of Navidrome) and libopus at build time
// (via sendspin-go). Control remains on Subsonic jukeboxControl; Sendspin
// controllers map play/pause/next/previous onto the shared jukebox queue.
package sendspin
