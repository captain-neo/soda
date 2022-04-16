package soda

import (
	"encoding/json"
	"math"
	"net"
	"net/url"
	"reflect"
	"strings"
	"time"

	"github.com/getkin/kin-openapi/openapi3"
)

var (
	timeType       = reflect.TypeOf(time.Time{})       // date-time RFC section 7.3.1
	ipType         = reflect.TypeOf(net.IP{})          // ipv4 and ipv6 RFC section 7.3.4, 7.3.5
	uriType        = reflect.TypeOf(url.URL{})         // uri RFC section 7.3.6
	byteSliceType  = reflect.TypeOf([]byte(nil))       // Byte slices will be encoded as base64
	rawMessageType = reflect.TypeOf(json.RawMessage{}) // Except for json.RawMessage
)

type getOAISchema interface {
	OAISchema() *openapi3.Schema
}

var getOAISchemaFunc = reflect.TypeOf((*getOAISchema)(nil)).Elem()

func (g *oaiGenerator) getSchemaName(rf reflect.Type) string {
	for rf.Kind() == reflect.Ptr {
		rf = rf.Elem()
	}
	name := rf.String()
	return strings.ReplaceAll(name, ".", "")
}

func (g *oaiGenerator) getSchemaRef(rf reflect.Type, typ string) *openapi3.SchemaRef {
	ref, _ := g.genSchema(nil, rf, typ)
	schemaName := g.getSchemaName(rf)
	g.openapi.Components.Schemas[schemaName] = ref
	return openapi3.NewSchemaRef("#/components/schemas/"+schemaName, ref.Value)
}

func (g *oaiGenerator) generateCycleSchemaRef(t reflect.Type, schema *openapi3.Schema) *openapi3.SchemaRef {
	switch t.Kind() {
	case reflect.Ptr:
		return g.generateCycleSchemaRef(t.Elem(), schema)
	case reflect.Slice:
		ref := g.generateCycleSchemaRef(t.Elem(), schema)
		g.openapi.Components.Schemas[g.getSchemaName(t.Elem())] = openapi3.NewSchemaRef("", ref.Value)
		sliceSchema := openapi3.NewArraySchema()
		sliceSchema.Items = ref
		return openapi3.NewSchemaRef("", sliceSchema)
	case reflect.Map:
		ref := g.generateCycleSchemaRef(t.Elem(), schema)
		g.openapi.Components.Schemas[g.getSchemaName(t.Elem())] = openapi3.NewSchemaRef("", ref.Value)
		mapSchema := openapi3.NewObjectSchema()
		mapSchema.AdditionalProperties = ref
		return openapi3.NewSchemaRef("", mapSchema)
	}

	return openapi3.NewSchemaRef("#/components/schemas/"+g.getSchemaName(t), schema)
}

func (g *oaiGenerator) genSchema(parents []reflect.Type, t reflect.Type, nameTag string) (*openapi3.SchemaRef, bool) { //nolint
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	for _, parent := range parents {
		if parent == t {
			return nil, true
		}
	}

	parents = append(parents, t)

	switch t.Kind() {
	case reflect.Struct:
		switch t {
		case timeType:
			return openapi3.NewDateTimeSchema().NewRef(), false
		case uriType:
			return openapi3.NewStringSchema().WithFormat("uri").NewRef(), false
		case ipType:
			return openapi3.NewStringSchema().WithFormat("ipv4").NewRef(), false
		default:
			if t.Implements(getOAISchemaFunc) {
				return reflect.New(t).Interface().(getOAISchema).OAISchema().NewRef(), false
			}

			schema := openapi3.NewObjectSchema().WithAnyAdditionalProperties()
			schema.AdditionalPropertiesAllowed = openapi3.BoolPtr(false)

			for i := 0; i < t.NumField(); i++ {
				f := t.Field(i)
				field := newFieldResolver(&f)
				var fieldSchemaRef *openapi3.SchemaRef
				if field.ignored {
					break
				}
				if field.shouldEmbed() {
					if ref, cycle := g.genSchema(parents, f.Type, nameTag); cycle {
						fieldSchemaRef = g.generateCycleSchemaRef(f.Type, schema)
					} else {
						fieldSchemaRef = ref
					}
					for k, v := range fieldSchemaRef.Value.Properties {
						schema.Properties[k] = v
					}
					continue
				}
				if ref, cycle := g.genSchema(parents, f.Type, nameTag); cycle {
					fieldSchemaRef = g.generateCycleSchemaRef(f.Type, schema)
				} else {
					fieldSchemaRef = ref
				}

				field.injectOAITags(fieldSchemaRef.Value)
				if fieldSchemaRef.Value.Nullable {
					nullSchema := openapi3.NewSchema()
					nullSchema.Type = "null"
					fieldSchemaRef.Value = &openapi3.Schema{
						OneOf: openapi3.SchemaRefs{
							fieldSchemaRef,
							nullSchema.NewRef(),
						},
					}
				}
				schema.Properties[field.name(nameTag)] = fieldSchemaRef
				if field.required() {
					schema.Required = append(schema.Required, field.name(nameTag))
				}
			}

			return schema.NewRef(), false
		}
	case reflect.Map:
		schema := openapi3.NewObjectSchema()
		additionalProperties, cycle := g.genSchema(parents, t.Elem(), nameTag)
		if cycle {
			additionalProperties = g.generateCycleSchemaRef(t.Elem(), schema)
		}
		schema.AdditionalProperties = additionalProperties
		return schema.NewRef(), false

	case reflect.Slice, reflect.Array:
		if t == rawMessageType {
			return openapi3.NewBytesSchema().NewRef(), false
		}
		if t.Kind() == reflect.Slice && t.Elem() == byteSliceType.Elem() {
			return openapi3.NewBytesSchema().NewRef(), false
		}
		schema := openapi3.NewArraySchema()
		if t.Kind() == reflect.Array {
			schema.MinItems = uint64(t.Len())
			schema.MaxItems = &schema.MinItems
		}
		if ref, cycle := g.genSchema(parents, t.Elem(), nameTag); cycle {
			schema.Items = g.generateCycleSchemaRef(t.Elem(), schema)
		} else {
			schema.Items = ref
		}
		return schema.NewRef(), false

	case reflect.Interface:
		return openapi3.NewSchema().WithAnyAdditionalProperties().NewRef(), false
	case reflect.Int:
		return openapi3.NewIntegerSchema().NewRef(), false
	case reflect.Uint:
		return openapi3.NewIntegerSchema().WithMin(0).WithMax(math.MaxUint).NewRef(), false
	case reflect.Int8:
		return openapi3.NewIntegerSchema().WithMin(math.MinInt8).WithMax(math.MaxInt8).NewRef(), false
	case reflect.Uint8:
		return openapi3.NewIntegerSchema().WithMin(0).WithMax(math.MaxUint8).NewRef(), false

	case reflect.Int16:
		return openapi3.NewIntegerSchema().WithMin(math.MinInt16).WithMax(math.MaxInt16).NewRef(), false
	case reflect.Uint16:
		return openapi3.NewIntegerSchema().WithMin(0).WithMax(math.MaxUint16).NewRef(), false

	case reflect.Int32:
		return openapi3.NewInt32Schema().WithMin(math.MinInt32).WithMax(math.MaxInt32).NewRef(), false
	case reflect.Uint32:
		return openapi3.NewInt32Schema().WithMin(0).WithMax(math.MaxUint32).NewRef(), false

	case reflect.Int64:
		return openapi3.NewInt64Schema().NewRef(), false
	case reflect.Uint64:
		return openapi3.NewInt64Schema().WithMin(0).NewRef(), false

	case reflect.Float32:
		return openapi3.NewFloat64Schema().WithFormat("float").NewRef(), false
	case reflect.Float64:
		return openapi3.NewFloat64Schema().WithFormat("double").NewRef(), false

	case reflect.Bool:
		return openapi3.NewBoolSchema().NewRef(), false
	case reflect.String:
		return openapi3.NewStringSchema().NewRef(), false
	default:
		panic("unsupported type " + t.String())
	}
}
