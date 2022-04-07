package soda

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"reflect"
	"strconv"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/gofiber/fiber/v2"
)

type Operations map[string]*Operation

func (ops Operations) Get(path, method string) *Operation {
	key := fmt.Sprintf("%s~~%s", path, method)
	return ops[key]
}
func (ops Operations) Set(path, method string, op *Operation) {
	key := fmt.Sprintf("%s~~%s", path, method)
	ops[key] = op
}

type Operation struct {
	Operation    *openapi3.Operation
	Path         string
	Method       string
	TParameters  reflect.Type
	TRequestBody reflect.Type
	Soda         *Soda

	handlers []fiber.Handler
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
		ref := op.Soda.oaiGenerator.GenerateResponse(op.Operation.OperationID, status, reflect.TypeOf(model))
		op.Operation.Responses[strconv.Itoa(status)] = ref
	} else {
		op.Operation.AddResponse(status, openapi3.NewResponse().WithDescription(http.StatusText(status)))
	}
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

func (op *Operation) BindData() fiber.Handler { //nolint
	return func(c *fiber.Ctx) error {
		// validate parameters
		if op.TParameters != nil { // nolint
			parameters := reflect.New(op.TParameters).Interface()
			if op.hasParameter("query") {
				if err := c.QueryParser(&parameters); err != nil {
					return err
				}
			}
			if op.hasParameter("header") {
				if err := headerParser(c, &parameters); err != nil {
					return err
				}
			}
			if op.hasParameter("path") {
				if err := pathParser(c, &parameters); err != nil {
					return err
				}
			}
			if op.hasParameter("cookie") {
				if err := cookieParser(c, &parameters); err != nil {
					return err
				}
			}

			if v := op.Soda.Options.Validator; v != nil {
				if err := v.StructCtx(c.Context(), parameters); err != nil {
					return err
				}
			}
			c.Locals(KeyParameter, parameters)
		}
		// validate request body
		if op.TRequestBody != nil {
			requestBody := reflect.New(op.TRequestBody).Interface()
			if err := c.BodyParser(&requestBody); err != nil {
				return err
			}
			if v := op.Soda.Options.Validator; v != nil {
				if err := v.StructCtx(c.Context(), requestBody); err != nil {
					return err
				}
			}
			c.Locals(KeyRequestBody, requestBody)
		}
		// TODO: validate response also?
		return c.Next()
	}
}

func (op *Operation) OK() *Operation {
	if err := op.Operation.Validate(context.TODO()); err != nil {
		log.Fatalln(err)
	}
	op.Soda.oaiGenerator.openapi.AddOperation(op.Path, op.Method, op.Operation)
	if err := op.Soda.oaiGenerator.openapi.Validate(context.TODO()); err != nil {
		panic(err)
	}
	op.handlers = append(op.handlers[:len(op.handlers)-1], op.BindData(), op.handlers[len(op.handlers)-1])
	op.Soda.Add(op.Method, op.Path, op.handlers...)
	return op
}

func (op *Operation) hasParameter(typ string) bool {
	if op.Operation.Parameters == nil {
		return false
	}
	for _, p := range op.Operation.Parameters {
		if p.Value.In == typ {
			return true
		}
	}
	return false
}
