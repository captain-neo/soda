package soda

import (
	"encoding/json"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/google/uuid"
	"math"
	"net"
	"net/url"
	"reflect"
	"strconv"
	"strings"
	"time"
)

const (
	OpenAPITag = "oai"
)

var (
	timeType       = reflect.TypeOf(time.Time{})       // date-time RFC section 7.3.1
	ipType         = reflect.TypeOf(net.IP{})          // ipv4 and ipv6 RFC section 7.3.4, 7.3.5
	uriType        = reflect.TypeOf(url.URL{})         // uri RFC section 7.3.6
	byteSliceType  = reflect.TypeOf([]byte(nil))       // Byte slices will be encoded as base64
	rawMessageType = reflect.TypeOf(json.RawMessage{}) // Except for json.RawMessage
	uuidType       = reflect.TypeOf(uuid.UUID{})       // uuid
)

// all supported openapi schema props
const (
	propTitle            = "title"
	propDescription      = "description"
	propType             = "type"
	propRequired         = "required"
	propReadOnly         = "readOnly"
	propWriteOnly        = "writeOnly"
	propMultipleOf       = "multipleOf"
	propMinimum          = "minimum"
	propMaximum          = "maximum"
	propExclusiveMaximum = "exclusiveMaximum"
	propExclusiveMinimum = "exclusiveMinimum"
	propDefault          = "default"
	propExample          = "example"
	propEnum             = "enum"
	propMinLength        = "minLength"
	propMaxLength        = "maxLength"
	propPattern          = "pattern"
	propFormat           = "format"
	propMinItems         = "minItems"
	propMaxItems         = "maxItems"
	propUniqueItems      = "uniqueItems"
)
const (
	typeNumber  = "number"
	typeString  = "string"
	typeInteger = "integer"
	typeArray   = "array"
)

type getFieldDoc interface {
	GetFieldDocString(fieldName string) string
}

var getFieldDocFunc = reflect.TypeOf((*getFieldDoc)(nil)).Elem()

type getOAISchema interface {
	OAISchema() *openapi3.Schema
}

var getOAISchemaFunc = reflect.TypeOf((*getOAISchema)(nil)).Elem()

func ResolveModel(v interface{}) *openapi3.Schema {
	return resolveBasicType(reflect.TypeOf(v))
}

func resolveBasicType(t reflect.Type) *openapi3.Schema {
	switch t.Kind() {
	case reflect.Struct:
		switch t {
		case timeType: // date-time RFC section 7.3.1
			return openapi3.NewDateTimeSchema()
		case uriType: // uri RFC section 7.3.6
			return openapi3.NewStringSchema().WithFormat("uri")
		case ipType: // ipv4 RFC section 7.3.4
			return openapi3.NewStringSchema().WithFormat("ipv4")
		case uuidType: // ipv4 RFC section 7.3.4
			return openapi3.NewUUIDSchema()
		default:
			return resolveStruct(t)
		}
	case reflect.Map:
		return openapi3.NewObjectSchema().WithAnyAdditionalProperties()

	case reflect.Slice, reflect.Array:
		if t == rawMessageType {
			return openapi3.NewBytesSchema()
		}
		if t.Kind() == reflect.Slice && t.Elem() == byteSliceType.Elem() {
			return openapi3.NewBytesSchema()
		}
		schema := openapi3.NewArraySchema()
		if t.Kind() == reflect.Array {
			schema.MinItems = uint64(t.Len())
			schema.MaxItems = &schema.MinItems
		}
		schema.Items = resolveBasicType(t.Elem()).NewRef()
		return schema

	case reflect.Interface:
		return openapi3.NewBytesSchema().WithAnyAdditionalProperties()
	case reflect.Int:
		return openapi3.NewIntegerSchema().WithMin(math.MinInt).WithMax(math.MaxInt)
	case reflect.Int8:
		return openapi3.NewIntegerSchema().WithMin(math.MinInt8).WithMax(math.MaxInt8)
	case reflect.Int16:
		return openapi3.NewIntegerSchema().WithMin(math.MinInt16).WithMax(math.MaxInt16)
	case reflect.Int32:
		return openapi3.NewInt32Schema().WithMin(math.MinInt32).WithMax(math.MaxInt32)
	case reflect.Int64:
		return openapi3.NewInt64Schema().WithMin(math.MinInt64).WithMax(math.MaxInt64)
	case reflect.Uint:
		return openapi3.NewIntegerSchema().WithMin(0).WithMax(math.MaxUint)
	case reflect.Uint8:
		return openapi3.NewIntegerSchema().WithMin(0).WithMax(math.MaxUint8)
	case reflect.Uint16:
		return openapi3.NewIntegerSchema().WithMin(0).WithMax(math.MaxUint16)
	case reflect.Uint32:
		return openapi3.NewInt32Schema().WithMin(0).WithMax(math.MaxUint32)
	case reflect.Uint64:
		return openapi3.NewInt64Schema().WithMin(0).WithMax(math.MaxUint64)
	case reflect.Float32:
		return openapi3.NewFloat64Schema().WithMin(math.SmallestNonzeroFloat32).WithMax(math.MaxFloat32)
	case reflect.Float64:
		return openapi3.NewFloat64Schema().WithMin(math.SmallestNonzeroFloat64).WithMax(math.MaxFloat64)
	case reflect.Bool:
		return openapi3.NewBoolSchema()
	case reflect.String:
		return openapi3.NewStringSchema()
	case reflect.Ptr:
		return resolveBasicType(t.Elem())
	}
	panic("unsupported type " + t.String())
}

// Reflects a struct to a JSON Schema type.
func resolveStruct(t reflect.Type) *openapi3.Schema {
	if t.Implements(getOAISchemaFunc) {
		return reflect.New(t).Interface().(getOAISchema).OAISchema()
	}
	schema := openapi3.NewObjectSchema().WithAnyAdditionalProperties()
	schema.AdditionalPropertiesAllowed = openapi3.BoolPtr(false)
	resolveStructFields(schema, t)
	return schema
}

func resolveStructFields(st *openapi3.Schema, t reflect.Type) {
	if t.Kind() != reflect.Struct {
		return
	}

	handleField := func(f reflect.StructField) {
		field := newFieldResolver(f)
		if field.shouldEmbed() {
			resolveStructFields(st, f.Type)
			return
		}
		if field.ignored {
			return
		}

		field.resolveTags()
		if t.Implements(getFieldDocFunc) {
			getFieldDocString := reflect.New(t).Interface().(getFieldDoc).GetFieldDocString
			field.schema.Description = getFieldDocString(f.Name)
		}
		if field.nullable() {
			nullSchema := openapi3.NewSchema()
			nullSchema.Type = "null"
			field.schema = &openapi3.Schema{
				OneOf: openapi3.SchemaRefs{
					field.schema.NewRef(),
					nullSchema.NewRef(),
				},
			}
		}
		st.Properties[field.name()] = field.schema.NewRef()
		if field.required() {
			st.Required = append(st.Required, field.name())
		}
	}

	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		handleField(f)
	}
}

func (s *fieldResolver) resolveTags() {
	s.resolveGeneric()
	switch s.schema.Type {
	case typeString:
		s.resolveString()
	case typeNumber, typeInteger:
		s.resolveNumeric()
	case typeArray:
		s.resolveArray()
	}
}

type fieldResolver struct {
	f        *reflect.StructField
	schema   *openapi3.Schema
	ignored  bool
	oaiTags  map[string]struct{}
	jsonTags map[string]struct{}
	jsonName string
}

func newFieldResolver(f reflect.StructField) *fieldResolver {
	resolver := &fieldResolver{
		f:        &f,
		ignored:  false,
		oaiTags:  nil,
		jsonTags: nil,
		schema:   resolveBasicType(f.Type),
	}

	if jsonTags, jsonOK := f.Tag.Lookup("json"); jsonOK {
		tags := strings.Split(jsonTags, ",")
		if tags[0] == "-" {
			resolver.ignored = true
		} else {
			resolver.jsonName = tags[0]
		}
		resolver.jsonTags = make(map[string]struct{})
		for _, tag := range tags {
			resolver.jsonTags[tag] = struct{}{}
		}
	}
	if oaiTags, oaiOK := f.Tag.Lookup(OpenAPITag); oaiOK {
		tags := strings.Split(oaiTags, ",")
		if tags[0] == "-" {
			resolver.ignored = true
		}
		resolver.oaiTags = make(map[string]struct{})
		for _, tag := range tags {
			resolver.oaiTags[tag] = struct{}{}
		}
	}
	return resolver
}

func (s fieldResolver) required() bool {
	required := true
	if s.f.Type.Kind() == reflect.Ptr {
		required = false
	}
	if _, ok := s.jsonTags["omitempty"]; ok {
		required = true
	}
	if _, ok := s.oaiTags[propRequired]; ok {
		required = true
	}
	return required
}

func (s fieldResolver) name() string {
	if s.jsonName != "" {
		return s.jsonName
	}
	return s.f.Name
}

func (s fieldResolver) nullable() bool {
	nullable := false
	if s.f.Type.Kind() == reflect.Ptr {
		nullable = true
	}
	if _, ok := s.oaiTags[propRequired]; ok {
		nullable = true
	}
	return nullable
}

func (s fieldResolver) shouldEmbed() bool {
	return s.f.Anonymous && !s.ignored
}

func (s *fieldResolver) resolveGeneric() {
	for tag := range s.oaiTags {
		if tag == propWriteOnly {
			s.schema.WriteOnly = true
			continue
		}
		if tag == propReadOnly {
			s.schema.ReadOnly = true
			continue
		}
		nameValue := strings.Split(tag, "=")
		if len(nameValue) == 2 {
			name, val := nameValue[0], nameValue[1]
			switch name {
			case propTitle:
				s.schema.Title = val
			case propDescription:
				s.schema.Description = val
			case propType:
				s.schema.Type = val
			}
		}
	}
}

// read struct tags for string type keywords
func (s *fieldResolver) resolveString() {
	for tag := range s.oaiTags {
		nameValue := strings.Split(tag, "=")
		if len(nameValue) == 2 {
			name, val := nameValue[0], nameValue[1]
			switch name {
			case propMinLength:
				i, _ := strconv.ParseUint(val, 10, 64)
				s.schema.MinLength = i
			case propMaxLength:
				i, _ := strconv.ParseUint(val, 10, 64)
				s.schema.MaxLength = &i
			case propPattern:
				s.schema.Pattern = val
			case propFormat:
				switch val {
				case "date-time", "date", "email", "hostname", "ipv4", "ipv6", "uri":
					s.schema.Format = val
				}
			case propEnum:
				for _, item := range strings.Split(val, " ") {
					s.schema.Enum = append(s.schema.Enum, item)
				}
			case propDefault:
				s.schema.Default = val
			case propExample:
				s.schema.Example = val
			}
		}
	}
}

// read struct tags for numeric type keywords
func (s *fieldResolver) resolveNumeric() {
	for tag := range s.oaiTags {
		nameValue := strings.Split(tag, "=")
		if len(nameValue) == 2 {
			name, val := nameValue[0], nameValue[1]
			switch name {
			case propMultipleOf:
				i, _ := strconv.ParseFloat(val, 64)
				s.schema.MultipleOf = &i
			case propMinimum:
				i, _ := strconv.ParseFloat(val, 64)
				s.schema.Min = &i
			case propMaximum:
				i, _ := strconv.ParseFloat(val, 64)
				s.schema.Max = &i
			case propExclusiveMaximum:
				b, _ := strconv.ParseBool(val)
				s.schema.ExclusiveMax = b
			case propExclusiveMinimum:
				b, _ := strconv.ParseBool(val)
				s.schema.ExclusiveMin = b
			case propDefault:
				switch s.schema.Type {
				case typeInteger:
					if i, err := strconv.Atoi(val); err == nil {
						s.schema.Default = i
					}
				case typeNumber:
					if i, err := strconv.ParseFloat(val, 64); err == nil {
						s.schema.Default = i
					}
				}
			case propExample:
				switch s.schema.Type {
				case typeInteger:
					if i, err := strconv.Atoi(val); err == nil {
						s.schema.Example = i
					}
				case typeNumber:
					if i, err := strconv.ParseFloat(val, 64); err == nil {
						s.schema.Example = i
					}
				}
			case propEnum:
				items := strings.Split(val, " ")
				switch s.schema.Type {
				case typeInteger:
					for _, item := range items {
						i, _ := strconv.Atoi(item)
						s.schema.Enum = append(s.schema.Enum, i)
					}
				case typeNumber:
					for _, item := range items {
						i, _ := strconv.ParseFloat(item, 64)
						s.schema.Enum = append(s.schema.Enum, i)
					}
				}
			}
		}
	}
}

// read struct tags for array type keywords
func (s *fieldResolver) resolveArray() {
	for tag := range s.oaiTags {
		nameValue := strings.Split(tag, "=")
		if len(nameValue) == 2 {
			name, val := nameValue[0], nameValue[1]
			switch name {
			case propMinItems:
				if i, err := strconv.ParseUint(val, 10, 64); err == nil {
					s.schema.MinItems = i
				}
			case propMaxItems:
				if i, err := strconv.ParseUint(val, 10, 64); err == nil {
					s.schema.MaxItems = &i
				}
			case propUniqueItems:
				s.schema.UniqueItems = true
			case propDefault, propEnum, propExample:
				ss := strings.Split(val, " ")
				switch s.schema.Type {
				case typeString:
					items := parseStringSlice(ss)
					switch name {
					case propDefault:
						s.schema.Default = items
					case propExample:
						s.schema.Example = items
					case propEnum:
						s.schema.Enum = items.([]interface{})
					}
				case typeInteger:
					items := parseIntSlice(ss)
					switch name {
					case propDefault:
						s.schema.Default = items
					case propExample:
						s.schema.Example = items
					case propEnum:
						s.schema.Enum = items.([]interface{})
					}
				case typeNumber:
					items := parseFloatSlice(ss)
					switch name {
					case propDefault:
						s.schema.Default = items
					case propExample:
						s.schema.Example = items
					case propEnum:
						s.schema.Enum = items.([]interface{})
					}
				}
			}
		}
	}
}

func parseStringSlice(ss []string) interface{} {
	result := make([]interface{}, 0, len(ss))
	for _, s := range ss {
		result = append(result, s)
	}
	return result
}
func parseIntSlice(ss []string) interface{} {
	result := make([]interface{}, 0, len(ss))
	for _, s := range ss {
		i, _ := strconv.Atoi(s)
		result = append(result, i)
	}
	return result
}
func parseFloatSlice(ss []string) interface{} {
	result := make([]interface{}, 0, len(ss))
	for _, s := range ss {
		i, _ := strconv.ParseFloat(s, 64)
		result = append(result, i)
	}
	return result
}
