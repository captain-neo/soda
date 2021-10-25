package soda

import (
	"context"
	"reflect"

	"github.com/getkin/kin-openapi/openapi3"
)

func (g *OAIGenerator) ResolveParameters(model reflect.Type) openapi3.Parameters {
	parameters := openapi3.NewParameters()
	g.resolveParameters(&parameters, model)
	return parameters
}

func (g *OAIGenerator) resolveParameters(parameters *openapi3.Parameters, t reflect.Type) {
	if t.Kind() != reflect.Struct {
		return
	}

	handleField := func(f reflect.StructField) {
		field := newFieldResolver(f)
		if field.shouldEmbed() {
			g.resolveParameters(parameters, f.Type)
			return
		}
		if field.ignored {
			return
		}
		fieldSchema, _ := g.getSchema(nil, f.Type)
		field.reflectSchemas(fieldSchema.Value)
		param := &openapi3.Parameter{
			Name:        field.name(),
			Required:    field.required(),
			Description: fieldSchema.Value.Description,
			Example:     fieldSchema.Value.Example,
			Deprecated:  fieldSchema.Value.Deprecated,
			Schema:      fieldSchema.Value.NewRef(),
		}
		switch field.tagPairs[propIn] {
		case openapi3.ParameterInHeader, openapi3.ParameterInPath, openapi3.ParameterInQuery, openapi3.ParameterInCookie:
			param.In = field.tagPairs[propIn]
		}
		if v, ok := field.tagPairs[propExplode]; ok {
			param.Explode = openapi3.BoolPtr(toBool(v))
		}
		if v, ok := field.tagPairs[propStyle]; ok {
			param.Style = v
		}
		if err := param.Validate(context.TODO()); err != nil {
			panic(err)
		}
		*parameters = append(*parameters, &openapi3.ParameterRef{Value: param})
	}
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		handleField(f)
	}
}
