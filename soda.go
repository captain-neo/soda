package soda

import (
	"context"
	"log"
	"regexp"
	"strings"
	"sync"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
)

type Options struct {
	SwaggerPath         *string
	RapiDocPath         *string
	RedocPath           *string
	OpenAPISpecJSONPath *string
	Validator           *validator.Validate
}
type Option func(o *Options)

func WithOpenAPISpec(path string) Option {
	return func(o *Options) {
		o.OpenAPISpecJSONPath = &path
	}
}

func WithSwagger(path string) Option {
	return func(o *Options) {
		o.SwaggerPath = &path
	}
}

func WithRedoc(path string) Option {
	return func(o *Options) {
		o.RedocPath = &path
	}
}

func WithRapiDoc(path string) Option {
	return func(o *Options) {
		o.RapiDocPath = &path
	}
}

func EnableValidateRequest(v ...*validator.Validate) Option {
	var validate *validator.Validate
	if len(v) == 0 {
		validate = validator.New()
	} else {
		validate = v[0]
	}
	return func(o *Options) {
		o.Validator = validate
	}
}

type Soda struct {
	spec         []byte
	specOnce     sync.Once
	oaiGenerator *oaiGenerator

	Options *Options
	*fiber.App
}

func (s *Soda) GetOpenAPIJSON() []byte {
	s.specOnce.Do(func() {
		if err := s.oaiGenerator.openapi.Validate(context.TODO()); err != nil {
			log.Fatalln(err)
		}
		spec, err := s.oaiGenerator.openapi.MarshalJSON()
		if err != nil {
			log.Fatalln(err)
		}
		s.spec = spec
	})
	return s.spec
}

func (s *Soda) GetOpenAPI() *openapi3.T {
	return s.oaiGenerator.openapi
}

func New(title, version string, fconf fiber.Config, options ...Option) *Soda { //nolint
	opt := &Options{}
	for _, option := range options {
		option(opt)
	}

	s := &Soda{
		oaiGenerator: newGenerator(&openapi3.Info{Title: title, Version: version}),
		App:          fiber.New(fconf),
		Options:      opt,
	}

	if opt.OpenAPISpecJSONPath != nil {
		s.App.Get(*opt.OpenAPISpecJSONPath, func(ctx *fiber.Ctx) error {
			ctx.Set(fiber.HeaderContentType, fiber.MIMEApplicationJSONCharsetUTF8)
			return ctx.Send(s.GetOpenAPIJSON())
		})
	}

	if opt.RapiDocPath != nil {
		s.App.Get(*opt.RapiDocPath, func(ctx *fiber.Ctx) error {
			ctx.Set(fiber.HeaderContentType, fiber.MIMETextHTML)
			return ctx.SendString(s.RapiDoc())
		})
	}

	if opt.SwaggerPath != nil {
		s.App.Get(*opt.SwaggerPath, func(ctx *fiber.Ctx) error {
			ctx.Set(fiber.HeaderContentType, fiber.MIMETextHTML)
			return ctx.SendString(s.Swagger())
		})
	}

	if opt.RedocPath != nil {
		s.App.Get(*opt.RedocPath, func(ctx *fiber.Ctx) error {
			ctx.Set(fiber.HeaderContentType, fiber.MIMETextHTML)
			return ctx.SendString(s.Redoc())
		})
	}
	return s
}

func (s *Soda) newOperation(path, method string, handlers ...fiber.Handler) *Operation {
	operation := openapi3.NewOperation()
	operation.AddResponse(0, openapi3.NewResponse().WithDescription("OK"))
	op := &Operation{
		Operation:    operation,
		Path:         path,
		Method:       method,
		TParameters:  nil,
		TRequestBody: nil,
		Soda:         s,
		handlers:     handlers,
	}
	return op
}

func (s *Soda) Get(path string, handlers ...fiber.Handler) *Operation {
	return s.Handle(path, "GET", handlers...)
}
func (s *Soda) Post(path string, handlers ...fiber.Handler) *Operation {
	return s.Handle(path, "POST", handlers...)
}
func (s *Soda) Put(path string, handlers ...fiber.Handler) *Operation {
	return s.Handle(path, "PUT", handlers...)
}
func (s *Soda) Patch(path string, handlers ...fiber.Handler) *Operation {
	return s.Handle(path, "PATCH", handlers...)
}
func (s *Soda) Delete(path string, handlers ...fiber.Handler) *Operation {
	return s.Handle(path, "DELETE", handlers...)
}
func (s *Soda) Handle(path, method string, handlers ...fiber.Handler) *Operation {
	summary := method + " " + path
	operationID := strings.Builder{}
	for _, p := range strings.Split(path, "/") {
		if p != "" {
			operationID.WriteString(strings.Title(p))
		}
	}
	operationID.WriteString(method)
	return s.newOperation(fixPath(path), method, handlers...).SetSummary(summary).SetOperationID(operationID.String())
}

var fixPathReg = regexp.MustCompile("/:([0-9a-zA-Z]+)")

func fixPath(path string) string {
	return fixPathReg.ReplaceAllString(path, "/{${1}}")
}
