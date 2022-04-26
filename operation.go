package soda

import (
	"context"
	"log"
	"net/http"
	"reflect"
	"regexp"
	"strconv"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/gofiber/fiber/v2"
)

type Operation struct {
	Operation    *openapi3.Operation
	Path         string
	Method       string
	TParameters  reflect.Type
	TRequestBody reflect.Type
	Soda         *Soda

	securityHandlers []fiber.Handler
	handlers         []fiber.Handler
}

func (op *Operation) SetDescription(desc string) *Operation {
	op.Operation.Description = desc
	return op
}

func (op *Operation) SetSummary(summary string) *Operation {
	op.Operation.Summary = summary
	return op
}

func (op *Operation) SetOperationID(id string) *Operation {
	op.Operation.OperationID = id
	return op
}

func (op *Operation) SetParameters(model interface{}) *Operation {
	op.TParameters = reflect.TypeOf(model)
	op.Operation.Parameters = op.Soda.oaiGenerator.GenerateParameters(op.TParameters)
	return op
}

func (op *Operation) AddJWTSecurity(validators ...fiber.Handler) *Operation {
	op.securityHandlers = append(op.securityHandlers, validators...)
	if len(op.Soda.oaiGenerator.openapi.Components.SecuritySchemes) == 0 {
		op.Soda.oaiGenerator.openapi.Components.SecuritySchemes = make(map[string]*openapi3.SecuritySchemeRef, 1)
	}
	op.Soda.oaiGenerator.openapi.Components.SecuritySchemes["JWTAuth"] = &openapi3.SecuritySchemeRef{Value: openapi3.NewJWTSecurityScheme()}
	if op.Operation.Security == nil {
		op.Operation.Security = openapi3.NewSecurityRequirements()
	}
	require := openapi3.NewSecurityRequirement().Authenticate("JWTAuth")
	op.Operation.Security.With(require)
	return op
}

func (op *Operation) SetJSONRequestBody(model interface{}) *Operation {
	op.TRequestBody = reflect.TypeOf(model)
	op.Operation.RequestBody = op.Soda.oaiGenerator.GenerateJSONRequestBody(op.Operation.OperationID, op.TRequestBody)
	return op
}

func (op *Operation) AddJSONResponse(status int, model interface{}) *Operation {
	if len(op.Operation.Responses) == 0 {
		op.Operation.Responses = make(openapi3.Responses)
	}
	if model != nil {
		ref := op.Soda.oaiGenerator.GenerateResponse(op.Operation.OperationID, status, reflect.TypeOf(model), "json")
		op.Operation.Responses[strconv.Itoa(status)] = ref
	} else {
		op.Operation.AddResponse(status, openapi3.NewResponse().WithDescription(http.StatusText(status)))
	}
	return op
}

func (op *Operation) AddResponseWithContentType(status int, contentType string) *Operation {
	if len(op.Operation.Responses) == 0 {
		op.Operation.Responses = make(openapi3.Responses)
	}
	ct := openapi3.NewContentWithSchema(openapi3.NewSchema(), []string{contentType})
	op.Operation.AddResponse(status, openapi3.NewResponse().WithContent(ct).WithDescription(http.StatusText(status)))
	return op
}

func (op *Operation) AddTags(tags ...string) *Operation {
	op.Operation.Tags = append(op.Operation.Tags, tags...)
	return op
}

func (op *Operation) SetDeprecated(deprecated bool) *Operation {
	op.Operation.Deprecated = deprecated
	return op
}

func (op *Operation) OK() *Operation {
	if err := op.Operation.Validate(context.TODO()); err != nil {
		log.Fatalln(err)
	}

	op.Soda.oaiGenerator.openapi.AddOperation(fixPath(op.Path), op.Method, op.Operation)
	if err := op.Soda.oaiGenerator.openapi.Validate(context.TODO()); err != nil {
		log.Fatalln(err)
	}
	op.handlers = append(op.handlers[:len(op.handlers)-1], BindData(op), op.handlers[len(op.handlers)-1])
	op.Soda.Add(op.Method, op.Path, op.handlers...)
	return op
}

func (op *Operation) parameterParsers() []parserFunc {
	if op.Operation.Parameters == nil {
		return nil
	}
	set := make(map[string]struct{})
	for _, p := range op.Operation.Parameters {
		set[p.Value.In] = struct{}{}
	}
	funcs := make([]parserFunc, 0, len(set))
	for k := range set {
		if fn, ok := parameterParsers[k]; ok {
			funcs = append(funcs, fn)
		}
	}
	return funcs
}

func BindData(op *Operation) fiber.Handler {
	return func(c *fiber.Ctx) error {
		for _, secHandler := range op.securityHandlers {
			if err := secHandler(c); err != nil {
				return err
			}
		}

		v := op.Soda.Options.validator
		if v == nil {
			return c.Next()
		}

		if op.TParameters != nil && op.TParameters.Kind() == reflect.Struct {
			parameters := reflect.New(op.TParameters).Interface()
			for _, parser := range op.parameterParsers() {
				if err := parser(c, parameters); err != nil {
					return err
				}
			}
			if err := v.StructCtx(c.Context(), parameters); err != nil {
				return err
			}
			c.Locals(KeyParameter, parameters)
		}

		// validate request body
		if op.TRequestBody != nil && op.TRequestBody.Kind() == reflect.Struct {
			requestBody := reflect.New(op.TRequestBody).Interface()
			if err := c.BodyParser(&requestBody); err != nil {
				return err
			}
			if err := v.StructCtx(c.Context(), requestBody); err != nil {
				return err
			}
			c.Locals(KeyRequestBody, requestBody)
		}
		// TODO: validate response also?
		return c.Next()
	}
}

var fixPathReg = regexp.MustCompile("/:([0-9a-zA-Z_]+)")

func fixPath(path string) string {
	return fixPathReg.ReplaceAllString(path, "/{${1}}")
}
