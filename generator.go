package soda

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/gofiber/fiber/v2/utils"
)

type OAIGenerator struct {
	openapi *openapi3.T
}

func NewGenerator(info *openapi3.Info) *OAIGenerator {
	return &OAIGenerator{
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

func (g *OAIGenerator) GenerateJSONRequestBody(operationID string, model reflect.Type) *openapi3.RequestBodyRef {
	schema := g.getSchemaRef(model)
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

func (g *OAIGenerator) GenerateResponse(operationID string, status int, model reflect.Type) *openapi3.ResponseRef {
	ref := g.getSchemaRef(model)
	responseName := fmt.Sprintf("%s%s", toCamelCase(operationID), strings.ReplaceAll(utils.StatusMessage(status), " ", ""))
	response := openapi3.NewResponse().WithJSONSchemaRef(ref).WithDescription(utils.StatusMessage(status))

	// TODO: check if has a duplicate name
	g.openapi.Components.Responses[responseName] = &openapi3.ResponseRef{Value: response}

	return &openapi3.ResponseRef{Ref: fmt.Sprintf("#/components/responses/%s", responseName), Value: response}
}

func (g *OAIGenerator) GenerateParameters(model reflect.Type) openapi3.Parameters {
	parameters := openapi3.NewParameters()
	g.generateParameters(&parameters, model)
	return parameters
}

func (g *OAIGenerator) generateParameters(parameters *openapi3.Parameters, t reflect.Type) {
	if t.Kind() != reflect.Struct {
		return
	}

	handleField := func(f reflect.StructField) {
		field := newFieldResolver(f)
		if field.shouldEmbed() {
			g.generateParameters(parameters, f.Type)
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
