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

func newFieldResolver(f *reflect.StructField) *fieldResolver {
	resolver := &fieldResolver{
		f:        f,
		ignored:  false,
		tagPairs: nil,
	}
	if oaiTags, oaiOK := f.Tag.Lookup(OpenAPITag); oaiOK {
		tags := strings.Split(oaiTags, SeparatorProp)
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
	case TypeString:
		s.resolveString(schema)
	case TypeNumber, TypeInteger:
		s.resolveNumeric(schema)
	case TypeArray:
		s.resolveArray(schema)
	}
}

func (s fieldResolver) required() bool {
	required := true
	if s.f.Type.Kind() == reflect.Ptr {
		required = false
	}
	if v, ok := s.tagPairs[PropRequired]; ok {
		required = toBool(v)
	}
	return required
}

func (s fieldResolver) name() string {
	if s.f.Name != "" {
		return s.f.Name
	}
	if name := s.f.Tag.Get("json"); name != "" {
		return strings.Split(name, ",")[0]
	}
	return s.f.Name
}

func (s fieldResolver) shouldEmbed() bool {
	return s.f.Anonymous && !s.ignored
}

func (s *fieldResolver) resolveGeneric(schema *openapi3.Schema) {
	for tag, val := range s.tagPairs {
		switch tag {
		case PropTitle:
			schema.Title = val
		case PropDescription:
			schema.Description = val
		case PropType:
			schema.Type = val
		case PropDeprecated:
			schema.Deprecated = toBool(val)
		case PropAllowEmptyValue:
			schema.AllowEmptyValue = toBool(val)
		case PropNullable:
			schema.Nullable = toBool(val)
		case PropWriteOnly:
			schema.WriteOnly = toBool(val)
		case PropReadOnly:
			schema.ReadOnly = toBool(val)
		}
	}
}

// read struct tags for string type keywords.
func (s *fieldResolver) resolveString(schema *openapi3.Schema) {
	for tag, val := range s.tagPairs {
		switch tag {
		case PropMinLength:
			schema.MinLength = toUint(val)
		case PropMaxLength:
			schema.MaxLength = openapi3.Uint64Ptr(toUint(val))
		case PropPattern:
			schema.Pattern = val
		case PropFormat:
			switch val {
			case "date-time", "date", "email", "hostname", "ipv4", "ipv6", "uri":
				schema.Format = val
			}
		case PropEnum:
			for _, item := range strings.Split(val, SeparatorPropItem) {
				schema.Enum = append(schema.Enum, item)
			}
		case PropDefault:
			schema.Default = val
		case PropExample:
			schema.Example = val
		}
	}
}

// read struct tags for numeric type keywords.
func (s *fieldResolver) resolveNumeric(schema *openapi3.Schema) { //nolint
	for tag, val := range s.tagPairs {
		switch tag {
		case PropMultipleOf:
			schema.MultipleOf = openapi3.Float64Ptr(toFloat(val))
		case PropMinimum:
			schema.Min = openapi3.Float64Ptr(toFloat(val))
		case PropMaximum:
			schema.Max = openapi3.Float64Ptr(toFloat(val))
		case PropExclusiveMaximum:
			schema.ExclusiveMax = toBool(val)
		case PropExclusiveMinimum:
			schema.ExclusiveMin = toBool(val)
		case PropDefault:
			switch schema.Type {
			case TypeInteger:
				schema.Default = toInt(val)
			case TypeNumber:
				schema.Default = toFloat(val)
			}
		case PropExample:
			switch schema.Type {
			case TypeInteger:
				schema.Example = toInt(val)
			case TypeNumber:
				schema.Example = toFloat(val)
			}
		case PropEnum:
			items := strings.Split(val, SeparatorPropItem)
			switch schema.Type {
			case TypeInteger:
				for _, item := range items {
					schema.Enum = append(schema.Enum, toInt(item))
				}
			case TypeNumber:
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
		case PropMinItems:
			schema.MinItems = toUint(val)
		case PropMaxItems:
			schema.MaxItems = openapi3.Uint64Ptr(toUint(val))
		case PropUniqueItems:
			schema.UniqueItems = toBool(val)
		case PropDefault, PropEnum, PropExample:
			var items interface{}
			switch schema.Items.Value.Type {
			case TypeString:
				items = parseStringSlice(val)
			case TypeInteger:
				items = parseIntSlice(val)
			case TypeNumber:
				items = parseFloatSlice(val)
			}

			switch tag {
			case PropDefault:
				schema.Default = items
			case PropExample:
				schema.Example = items
			case PropEnum:
				schema.Enum = []interface{}{items}
			}
		}
	}
}
