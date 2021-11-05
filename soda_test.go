package soda_test

import (
	"math"
	"reflect"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/panicneo/soda"
)

var _ = Describe("Soda", func() {
	Describe("the soda package", func() {
		Context("soda.NewSoda()", func() {
			When("pass ok cases", func() {
				It("should generate correct schema", func() {
					type requestBodyOk struct {
						Int                 int       `oai:"name=int"`
						OptionalInt         *int      `oai:"name=optional_int"`
						Int32               int32     `oai:"name=int32"`
						OptionalInt32       *int32    `oai:"name=optional_int32"`
						UInt32              uint32    `oai:"name=uint32"`
						OptionalUInt32      *uint32   `oai:"name=optional_uint32"`
						String              string    `oai:"name=string"`
						OptionalString      *string   `oai:"name=optional_string"`
						Float               float64   `oai:"name=float"`
						OptionalFloat       *float64  `oai:"name=optional_float"`
						Bool                bool      `oai:"name=bool"`
						OptionalBool        *bool     `oai:"name=optional_bool"`
						StringArray         []string  `oai:"name=string_array"`
						OptionalStringArray *[]string `oai:"name=optional_string_array"`
					}
					generator := soda.NewGenerator(nil)
					ref := generator.GenerateJSONRequestBody("test-ok", reflect.TypeOf(requestBodyOk{}))
					schema := ref.Value.Content["application/json"].Schema.Value

					Expect(schema.Required).To(BeEquivalentTo([]string{"int", "int32", "uint32", "string", "float", "bool", "string_array"}))
					Expect(int(*schema.Properties["int32"].Value.Max)).To(BeEquivalentTo(math.MaxInt32))
					Expect(int(*schema.Properties["int32"].Value.Min)).To(BeEquivalentTo(math.MinInt32))
					Expect(int(*schema.Properties["uint32"].Value.Max)).To(BeEquivalentTo(math.MaxUint32))
					Expect(int(*schema.Properties["uint32"].Value.Min)).To(BeEquivalentTo(0))
				})
			})
		})
	})
})
