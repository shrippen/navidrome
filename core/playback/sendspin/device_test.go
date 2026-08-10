package sendspin

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("IsDevice", func() {
	It("matches the sendspin device name case-insensitively", func() {
		Expect(IsDevice("sendspin")).To(BeTrue())
		Expect(IsDevice("Sendspin")).To(BeTrue())
		Expect(IsDevice(" sendspin ")).To(BeTrue())
	})

	It("rejects other device names", func() {
		Expect(IsDevice("auto")).To(BeFalse())
		Expect(IsDevice("alsa/hdmi")).To(BeFalse())
		Expect(IsDevice("")).To(BeFalse())
	})
})
