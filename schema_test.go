package soda_test

import (
	"math"
	"reflect"
	"testing"

	"github.com/go-playground/assert"
	"github.com/panicneo/soda"
)

func TestOAIGenerator_GenerateJSONRequestBody(t *testing.T) {

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

	assert.Equal(t, schema.Required, []string{"int", "int32", "uint32", "string", "float", "bool", "string_array"})
	assert.Equal(t, int(*schema.Properties["int32"].Value.Max), math.MaxInt32)
	assert.Equal(t, int(*schema.Properties["int32"].Value.Min), math.MinInt32)
	assert.Equal(t, int(*schema.Properties["uint32"].Value.Max), math.MaxUint32)
	assert.NotEqual(t, int(*schema.Properties["uint32"].Value.Min), 0)

}
