package cloudnative_test

import (
	"os"
	"testing"

	"github.com/cloudfoundry/cnb2cf/cloudnative"
	"github.com/sclevine/spec"

	. "github.com/onsi/gomega"
)

func testEnvironment(t *testing.T, when spec.G, it spec.S) {
	var (
		Expect func(interface{}, ...interface{}) Assertion

		environment cloudnative.Environment
	)

	it.Before(func() {
		Expect = NewWithT(t).Expect

		environment = cloudnative.NewEnvironment()
	})

	when("Services", func() {
		it.Before(func() {
			os.Setenv("VCAP_SERVICES", `{"some-key": "some-value"}`)
		})

		it("retrieves VCAP_SERVICES env var", func() {
			Expect(environment.Services()).To(Equal(`{"some-key": "some-value"}`))
		})

		when("No Services", func() {
			it.Before(func() {
				os.Unsetenv("VCAP_SERVICES")
			})
			it("returns empty json", func() {
				Expect(environment.Services()).To(Equal(`{}`))
			})
		})
	})

	when("Stack", func() {
		it.Before(func() {
			os.Setenv("CF_STACK", "some-stack")
		})

		it("return CF_STACK env var", func() {
			Expect(environment.Stack()).To(Equal("some-stack"))
		})
	})
}
