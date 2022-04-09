package soda

import (
	"context"
	"fmt"
	"net/http"
	"reflect"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
)

type oaiGenerator struct {
	openapi *openapi3.T
}

func newGenerator(info *openapi3.Info) *oaiGenerator {
	return &oaiGenerator{
		openapi: &openapi3.T{
			OpenAPI: "3.0.3",
			Info:    info,
			Components: openapi3.Components{
				Schemas:       make(openapi3.Schemas),
				Responses:     make(openapi3.Responses),
				RequestBodies: make(openapi3.RequestBodies),
			},
		},
	}
}

func (g *oaiGenerator) GenerateJSONRequestBody(operationID string, model reflect.Type) *openapi3.RequestBodyRef {
	schema := g.getSchemaRef(model, "json")
	requestBody := openapi3.NewRequestBody().WithJSONSchemaRef(schema).WithRequired(true)
	requestName := toCamelCase(operationID)

	// TODO: check if duplicate name
	g.openapi.Components.RequestBodies[requestName] = &openapi3.RequestBodyRef{
		Value: requestBody,
	}
	return &openapi3.RequestBodyRef{
		Ref:   fmt.Sprintf("#/components/requestBodies/%s", requestName),
		Value: requestBody,
	}
}

func (g *oaiGenerator) GenerateResponse(operationID string, status int, model reflect.Type, typ string) *openapi3.ResponseRef {
	ref := g.getSchemaRef(model, typ)
	responseName := fmt.Sprintf("%s%s", toCamelCase(operationID), strings.ReplaceAll(http.StatusText(status), " ", ""))
	response := openapi3.NewResponse().WithJSONSchemaRef(ref).WithDescription(http.StatusText(status))

	// TODO: check if has a duplicate name
	g.openapi.Components.Responses[responseName] = &openapi3.ResponseRef{Value: response}

	return &openapi3.ResponseRef{Ref: fmt.Sprintf("#/components/responses/%s", responseName), Value: response}
}

func (g *oaiGenerator) GenerateParameters(model reflect.Type) openapi3.Parameters {
	parameters := openapi3.NewParameters()
	g.generateParameters(&parameters, model)
	return parameters
}

func (g *oaiGenerator) generateParameters(parameters *openapi3.Parameters, t reflect.Type) {
	if t.Kind() != reflect.Struct {
		return
	}

	handleField := func(f *reflect.StructField) {
		var typ string
		for _, pos := range []string{openapi3.ParameterInQuery, openapi3.ParameterInHeader, openapi3.ParameterInPath, openapi3.ParameterInCookie} {
			if name := f.Tag.Get(pos); name != "" {
				typ = pos
				break
			}
		}
		field := newFieldResolver(f)
		if field.shouldEmbed() {
			g.generateParameters(parameters, f.Type)
			return
		}
		if field.ignored {
			return
		}
		if typ == "" {
			panic(fmt.Sprintf("field %q's parameter type is unknown", f.Name))
		}
		fieldSchema, _ := g.genSchema(nil, f.Type, typ)
		field.injectOAITags(fieldSchema.Value)
		param := &openapi3.Parameter{
			Required:    field.required(),
			Description: fieldSchema.Value.Description,
			Example:     fieldSchema.Value.Example,
			Deprecated:  fieldSchema.Value.Deprecated,
			Schema:      fieldSchema.Value.NewRef(),
		}
		param.In = typ
		param.Name = field.name(typ)

		if v, ok := field.tagPairs[PropExplode]; ok {
			param.Explode = openapi3.BoolPtr(toBool(v))
		}
		if v, ok := field.tagPairs[PropStyle]; ok {
			param.Style = v
		}
		if err := param.Validate(context.TODO()); err != nil {
			panic(err)
		}
		*parameters = append(*parameters, &openapi3.ParameterRef{Value: param})
	}
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		handleField(&f)
	}
}
