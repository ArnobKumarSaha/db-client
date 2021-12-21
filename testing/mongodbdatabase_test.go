package schema_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"kubedb.dev/schema-manager/controllers/schema/framework"
)

var _ = Describe("Mongodbdatabase", func() {
	var (
		x *framework.Invocation
	)
	BeforeEach(func() {
		x = framework.NewInvocation().Invoke()
	})
	Describe("Categorizing book length", func() {
		Context("With more than 300 pages", func() {
			It("should be a novel", func() {
				//Expect(x.Invoke()).To(Equal("NOVEL"))
				Expect(x.Checker()).To(Equal("true"))
			})
		})
	})
})
