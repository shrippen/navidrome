package playback_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/conf/configtest"
	"github.com/navidrome/navidrome/core/playback"
)

var _ = Describe("SendspinUIEnabled", func() {
	var restore func()

	BeforeEach(func() {
		restore = configtest.SetupConfig()
	})
	AfterEach(func() {
		restore()
	})

	It("is false when jukebox is disabled", func() {
		conf.Server.Jukebox.Enabled = false
		conf.Server.Jukebox.Sendspin.Enabled = true
		Expect(playback.SendspinUIEnabled()).To(BeFalse())
	})

	It("is true when Sendspin is enabled under jukebox", func() {
		conf.Server.Jukebox.Enabled = true
		conf.Server.Jukebox.Sendspin.Enabled = true
		Expect(playback.SendspinUIEnabled()).To(BeTrue())
	})

	It("is true when a sendspin device is configured", func() {
		conf.Server.Jukebox.Enabled = true
		conf.Server.Jukebox.Sendspin.Enabled = false
		conf.Server.Jukebox.Devices = []conf.AudioDeviceDefinition{{"House", "sendspin"}}
		Expect(playback.SendspinUIEnabled()).To(BeTrue())
	})
})
