package soda

import (
	"reflect"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
)

type fieldResolver struct {
	f        *reflect.StructField
	ignored  bool
	tagPairs map[string]string
}

func newFieldResolver(f reflect.StructField) *fieldResolver {
	resolver := &fieldResolver{
		f:        &f,
		ignored:  false,
		tagPairs: nil,
	}
	if oaiTags, oaiOK := f.Tag.Lookup(OpenAPITag); oaiOK {
		tags := strings.Split(oaiTags, ",")
		if tags[0] == "-" {
			resolver.ignored = true
			return resolver
		}
		resolver.tagPairs = make(map[string]string)
		for _, tag := range tags {
			tag = strings.TrimSpace(tag)
			pair := strings.Split(tag, "=")
			if len(pair) == 2 {
				resolver.tagPairs[strings.TrimSpace(pair[0])] = strings.TrimSpace(pair[1])
			} else {
				resolver.tagPairs[strings.TrimSpace(pair[0])] = ""
			}
		}
	}
	return resolver
}

func (s *fieldResolver) reflectSchemas(schema *openapi3.Schema) {
	s.resolveGeneric(schema)
	switch schema.Type {
	case typeString:
		s.resolveString(schema)
	case typeNumber, typeInteger:
		s.resolveNumeric(schema)
	case typeArray:
		s.resolveArray(schema)
	}
}

func (s fieldResolver) required() bool {
	required := true
	if s.f.Type.Kind() == reflect.Ptr {
		required = false
	}
	if v, ok := s.tagPairs[propRequired]; ok {
		required = toBool(v)
	}
	return required
}

func (s fieldResolver) name() string {
	if name, ok := s.tagPairs[propName]; ok {
		return name
	}
	return s.f.Name
}

func (s fieldResolver) shouldEmbed() bool {
	return s.f.Anonymous && !s.ignored
}

func (s *fieldResolver) resolveGeneric(schema *openapi3.Schema) {
	for tag, val := range s.tagPairs {
		switch tag {
		case propTitle:
			schema.Title = val
		case propDescription:
			schema.Description = val
		case propType:
			schema.Type = val
		case propDeprecated:
			schema.Deprecated = toBool(val)
		case propAllowEmptyValue:
			schema.AllowEmptyValue = toBool(val)
		case propNullable:
			schema.Nullable = toBool(val)
		case propWriteOnly:
			schema.WriteOnly = toBool(val)
		case propReadOnly:
			schema.ReadOnly = toBool(val)
		}
	}
}

// read struct tags for string type keywords.
func (s *fieldResolver) resolveString(schema *openapi3.Schema) {
	for tag, val := range s.tagPairs {
		switch tag {
		case propMinLength:
			schema.MinLength = toUint(val)
		case propMaxLength:
			schema.MaxLength = openapi3.Uint64Ptr(toUint(val))
		case propPattern:
			schema.Pattern = val
		case propFormat:
			switch val {
			case "date-time", "date", "email", "hostname", "ipv4", "ipv6", "uri":
				schema.Format = val
			}
		case propEnum:
			for _, item := range strings.Split(val, " ") {
				schema.Enum = append(schema.Enum, item)
			}
		case propDefault:
			schema.Default = val
		case propExample:
			schema.Example = val
		}
	}
}

// read struct tags for numeric type keywords.
func (s *fieldResolver) resolveNumeric(schema *openapi3.Schema) {
	for tag, val := range s.tagPairs {
		switch tag {
		case propMultipleOf:
			schema.MultipleOf = openapi3.Float64Ptr(toFloat(val))
		case propMinimum:
			schema.Min = openapi3.Float64Ptr(toFloat(val))
		case propMaximum:
			schema.Max = openapi3.Float64Ptr(toFloat(val))
		case propExclusiveMaximum:
			schema.ExclusiveMax = toBool(val)
		case propExclusiveMinimum:
			schema.ExclusiveMin = toBool(val)
		case propDefault:
			switch schema.Type {
			case typeInteger:
				schema.Default = toInt(val)
			case typeNumber:
				schema.Default = toFloat(val)
			}
		case propExample:
			switch schema.Type {
			case typeInteger:
				schema.Example = toInt(val)
			case typeNumber:
				schema.Example = toFloat(val)
			}
		case propEnum:
			items := strings.Split(val, " ")
			switch schema.Type {
			case typeInteger:
				for _, item := range items {
					schema.Enum = append(schema.Enum, toInt(item))
				}
			case typeNumber:
				for _, item := range items {
					schema.Enum = append(schema.Enum, toFloat(item))
				}
			}
		}
	}
}

// read struct tags for array type keywords.
func (s *fieldResolver) resolveArray(schema *openapi3.Schema) {
	for tag, val := range s.tagPairs {
		switch tag {
		case propMinItems:
			schema.MinItems = toUint(val)
		case propMaxItems:
			schema.MaxItems = openapi3.Uint64Ptr(toUint(val))
		case propUniqueItems:
			schema.UniqueItems = toBool(val)
		case propDefault, propEnum, propExample:
			var items interface{}
			switch schema.Items.Value.Type {
			case typeString:
				items = parseStringSlice(val)
			case typeInteger:
				items = parseIntSlice(val)
			case typeNumber:
				items = parseFloatSlice(val)
			}

			switch tag {
			case propDefault:
				schema.Default = items
			case propExample:
				schema.Example = items
			case propEnum:
				schema.Enum = []interface{}{items}
			}
		}
	}
}
