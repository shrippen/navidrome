package sendspin

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("applyGain", func() {
	It("scales samples and clamps to 24-bit range", func() {
		samples := []int32{1000, -1000, 8388607}
		applyGain(samples, 0.5)
		Expect(samples[0]).To(Equal(int32(500)))
		Expect(samples[1]).To(Equal(int32(-500)))
		Expect(samples[2]).To(Equal(int32(4194303)))

		hot := []int32{8388607}
		applyGain(hot, 2.0)
		Expect(hot[0]).To(Equal(int32(8388607)))
	})
})

var _ = Describe("switchableSource", func() {
	It("returns silence while paused", func() {
		src := newSwitchableSource()
		buf := make([]int32, 8)
		n, err := src.Read(buf)
		Expect(err).ToNot(HaveOccurred())
		Expect(n).To(Equal(8))
		Expect(buf).To(Equal(make([]int32, 8)))
	})

	It("reports empty metadata without a loaded track", func() {
		src := newSwitchableSource()
		title, artist, album := src.Metadata()
		Expect(title).To(BeEmpty())
		Expect(artist).To(BeEmpty())
		Expect(album).To(BeEmpty())
	})
})
