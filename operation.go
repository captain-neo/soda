package soda

import (
	"context"
	"errors"
	"log"
	"reflect"
	"strconv"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/utils"
	"github.com/mitchellh/mapstructure"
)

type Operation struct {
	operation        *openapi3.Operation
	handler          fiber.Handler
	path             string
	method           string
	parametersNameKV map[string]string
	tParameters      reflect.Type
	tRequestBody     reflect.Type
	soda             *Soda
}

func (s *Soda) NewOperation(path, method string, handler fiber.Handler) *Operation {
	op := &Operation{
		soda:      s,
		path:      path,
		method:    utils.ToUpper(method),
		operation: openapi3.NewOperation(),
		handler:   handler,
	}
	op.operation.OperationID = toKebabCase(utils.ToLower(method) + getHandlerName(handler))
	return op
}

func (op *Operation) SetDescription(desc string) *Operation {
	op.operation.Description = desc
	return op
}

func (op *Operation) SetSummary(summary string) *Operation {
	op.operation.Summary = summary
	return op
}

func (op *Operation) SetOperationID(id string) *Operation {
	op.operation.OperationID = id
	return op
}

func (op *Operation) SetParameters(model interface{}) *Operation {
	op.tParameters = reflect.TypeOf(model)
	op.operation.Parameters = op.soda.oaiGenerator.ResolveParameters(op.tParameters)
	op.parametersNameKV = make(map[string]string, len(op.operation.Parameters))
	for i := 0; i < op.tParameters.NumField(); i++ {
		f := op.tParameters.Field(i)
		op.parametersNameKV[newFieldResolver(f).name()] = f.Name
	}
	return op
}

func (op *Operation) SetJSONRequestBody(model interface{}) *Operation {
	op.tRequestBody = reflect.TypeOf(model)
	op.operation.RequestBody = op.soda.oaiGenerator.GenerateJSONRequestBody(op.operation.OperationID, op.tRequestBody)
	return op
}

func (op *Operation) AddJSONResponse(status int, model interface{}) *Operation {
	if len(op.operation.Responses) == 0 {
		op.operation.Responses = make(openapi3.Responses)
	}
	if model != nil {
		ref := op.soda.oaiGenerator.GenerateResponse(op.operation.OperationID, status, reflect.TypeOf(model))
		op.operation.Responses[strconv.Itoa(status)] = ref
	} else {
		op.operation.AddResponse(status, openapi3.NewResponse().WithDescription(utils.StatusMessage(status)))
	}
	return op
}

func (op *Operation) AddTags(tags ...string) *Operation {
	op.operation.Tags = append(op.operation.Tags, tags...)
	return op
}

func (op *Operation) SetDeprecated(deprecated bool) *Operation {
	op.operation.Deprecated = deprecated
	return op
}

func (op *Operation) Mount() {
	// TODO: check operationID duplicate
	// validate the operation
	if err := op.operation.Validate(context.TODO()); err != nil {
		log.Fatalln(err)
	}
	op.soda.oaiGenerator.openapi.AddOperation(op.path, op.method, op.operation)
	op.soda.App.Add(op.method, op.path, op.ValidationHandler(), op.handler)
}

func (op *Operation) ValidationHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		// validate parameters
		values, err := op.validateParameters(c)
		if err != nil {
			return err
		}
		c.Locals(KeyParameter, values)

		// validate request body
		body, err := op.validateRequestBody(c)
		if err != nil {
			return err
		}
		c.Locals(KeyRequestBody, body)
		// TODO: check response also?
		return c.Next()
	}
}

func (op *Operation) validateParameters(c *fiber.Ctx) (interface{}, error) {
	m := make(map[string]interface{}, len(op.operation.Parameters))
	for _, param := range op.operation.Parameters {
		v, err := op.validateParameter(c, param.Value)
		if err != nil {
			return nil, err
		}
		m[op.parametersNameKV[param.Value.Name]] = v
	}
	// here we use mapstructure to transform an interface{} to struct
	v := reflect.New(op.tParameters).Interface()
	if err := mapstructure.WeakDecode(m, v); err != nil {
		return v, err
	}
	return v, nil
}

func (op *Operation) validateParameter(c *fiber.Ctx, parameter *openapi3.Parameter) (interface{}, error) {
	if parameter.Schema == nil && parameter.Content == nil {
		// We have no schema for the parameter. Assume that everything passes
		// a schema-less check, but this could also be an error. The OpenAPI
		// validation allows this to happen.
		return nil, nil
	}
	var schema *openapi3.Schema
	var value interface{}
	var err error
	// ValidationHandler will ensure that we either have content or schema.
	if parameter.Content != nil {
		value, schema, err = decodeContentParameter(parameter, c)
		if err != nil {
			return nil, ValidationError{
				Field:    parameter.Name,
				Position: parameter.In,
				Reason:   err.Error(),
			}
		}
	} else {
		value, err = decodeStyledParameter(parameter, c)
		if err != nil {
			return nil, ValidationError{
				Field:    parameter.Name,
				Position: parameter.In,
				Reason:   err.Error(),
			}
		}
		schema = parameter.Schema.Value
	}
	if v := reflect.ValueOf(value); !v.IsValid() || v.IsZero() {
		if parameter.Schema.Value.Default == nil && parameter.Required {
			return nil, ValidationError{
				Field:    parameter.Name,
				Position: parameter.In,
				Reason:   "field is required",
			}
		}
		return parameter.Schema.Value.Default, nil
	}
	if schema == nil {
		// A parameter's schema is not defined so skip validation of a parameter's value.
		return nil, nil
	}

	opts := []openapi3.SchemaValidationOption{openapi3.VisitAsRequest()}
	if err = schema.VisitJSON(value, opts...); err != nil {
		var e *openapi3.SchemaError
		if ok := errors.Is(err, e); ok {
			return nil, ValidationError{
				Field:    parameter.Name,
				Position: parameter.In,
				Reason:   e.Reason,
			}
		}
		return value, err
	}

	return value, nil
}

//nolint:funlen
func (op *Operation) validateRequestBody(c *fiber.Ctx) (interface{}, error) {
	if op.operation.RequestBody == nil {
		return nil, nil
	}
	requestBody := op.operation.RequestBody.Value

	if len(c.Body()) == 0 {
		if requestBody.Required {
			return nil, ValidationError{
				Position: "request body",
				Reason:   "request body is empty",
			}
		}
		return nil, nil
	}

	content := requestBody.Content
	if len(content) == 0 {
		// A request's body does not have declared content, so skip validation.
		return nil, nil
	}

	cType := utils.ToLower(utils.UnsafeString(c.Request().Header.ContentType()))
	cType = utils.ParseVendorSpecificContentType(cType)
	contentType := requestBody.Content.Get(cType)
	if contentType == nil {
		return nil, ValidationError{
			Field:    "ContentType",
			Position: "header",
			Reason:   "invalid content type",
		}
	}

	if contentType.Schema == nil {
		// A JSON schema that describes the received data is not declared, so skip validation.
		return nil, nil
	}
	value := make(map[string]interface{})
	// value := reflect.New(op.tRequestBody).Interface()
	if err := c.BodyParser(&value); err != nil {
		return value, ValidationError{
			Field:    "",
			Position: "request body",
			Reason:   "parse request body failed",
		}
	}

	opts := make([]openapi3.SchemaValidationOption, 0, 1) // 2 potential opts here
	opts = append(opts, openapi3.VisitAsRequest())

	// Validate JSON with the schema
	if err := contentType.Schema.Value.VisitJSON(value, opts...); err != nil {
		// var e *openapi3.SchemaError
		if e, ok := err.(*openapi3.SchemaError); ok {
			return nil, ValidationError{
				Field:    e.SchemaField,
				Position: "request body",
				Reason:   e.Reason,
			}
		}
		if err != nil {
			return nil, ValidationError{
				Position: "request body",
				Reason:   err.Error(),
			}
		}
	}
	ret := reflect.New(op.tRequestBody).Interface()
	if err := mapstructure.WeakDecode(value, &ret); err != nil{
		return nil, ValidationError{
			Position: "request body",
			Reason:   err.Error(),
		}
	}
	return ret, nil
}
