package sendspin

import (
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// fixturePath returns a path under tests/fixtures. tests.Init chdirs to the
// module root, so paths must be rooted there (not relative to this package).
func fixturePath(name string) string {
	GinkgoHelper()
	path, err := filepath.Abs(filepath.Join("tests", "fixtures", name))
	Expect(err).ToNot(HaveOccurred())
	Expect(path).To(BeARegularFile())
	return path
}
