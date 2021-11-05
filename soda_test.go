package soda_test

import (
	"math"
	"reflect"
	"testing"

	"github.com/gofiber/fiber/v2"
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

func TestSodaApp(t *testing.T) {
	CreateSoda := func(c *fiber.Ctx) error {
		return c.SendString("hello")
	}
	UpdateSoda := func(c *fiber.Ctx) error {
		return c.SendString("hello")
	}

	type Parameters struct {
		Int             int      `oai:"int,in=query,name=int" json:"int,omitempty"`
		IntDefault      int      `oai:"int,in=query,name=int_default,default=3" json:"int_default,omitempty"`
		IntSlice        []int    `oai:"int,in=query,name=int_slice,explode=0" json:"int_slice,omitempty"`
		IntSliceDefault []int    `oai:"int,in=query,enum=1 2 3"`
		String          string   `oai:"int,in=query,name=string" json:"string,omitempty"`
		StringSlice     []string `oai:"int,in=query,name=string_slice,explode=f" json:"string_slice,omitempty"`
	}

	type EmbedStruct struct {
		A string
	}

	type Body struct {
		EmbedStruct
		B *Body
	}
	s := soda.NewSoda(&soda.Info{
		Title:       "Title",
		Description: "Desc",
		Version:     "1.0.0",
	})
	s.POST("/soda", CreateSoda).
		SetOperationID("create-soda").
		SetParameters(Parameters{}).
		SetJSONRequestBody(Body{}).
		AddJSONResponse(201, Body{}).
		AddJSONResponse(204, nil).Mount()
	s.PUT("/soda", UpdateSoda).
		SetOperationID("update-soda").
		SetParameters(Parameters{}).
		SetJSONRequestBody(Body{}).
		AddJSONResponse(200, Body{}).Mount()
}
