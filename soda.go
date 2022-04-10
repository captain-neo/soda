package soda

import (
	"context"
	"log"
	"strings"
	"sync"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

type Options struct {
	swaggerPath         *string
	rapiDocPath         *string
	redocPath           *string
	openAPISpecJSONPath *string
	fiberConfig         []fiber.Config
	validator           *validator.Validate
}
type Option func(o *Options)

func WithOpenAPISpec(path string) Option {
	return func(o *Options) {
		o.openAPISpecJSONPath = &path
	}
}

func WithSwagger(path string) Option {
	return func(o *Options) {
		o.swaggerPath = &path
	}
}

func WithRedoc(path string) Option {
	return func(o *Options) {
		o.redocPath = &path
	}
}

func WithRapiDoc(path string) Option {
	return func(o *Options) {
		o.rapiDocPath = &path
	}
}

func WithFiberConfig(config ...fiber.Config) Option {
	return func(o *Options) {
		o.fiberConfig = config
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
		o.validator = validate
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

func (s *Soda) OpenAPI() *openapi3.T {
	return s.oaiGenerator.openapi
}

func New(title, version string, options ...Option) *Soda {
	opt := &Options{}
	for _, option := range options {
		option(opt)
	}

	s := &Soda{
		oaiGenerator: newGenerator(&openapi3.Info{Title: title, Version: version}),
		App:          fiber.New(opt.fiberConfig...),
		Options:      opt,
	}

	if opt.openAPISpecJSONPath != nil {
		s.App.Get(*opt.openAPISpecJSONPath, func(ctx *fiber.Ctx) error {
			ctx.Set(fiber.HeaderContentType, fiber.MIMEApplicationJSONCharsetUTF8)
			return ctx.Send(s.GetOpenAPIJSON())
		})
	}

	if opt.rapiDocPath != nil {
		s.App.Get(*opt.rapiDocPath, func(ctx *fiber.Ctx) error {
			ctx.Set(fiber.HeaderContentType, fiber.MIMETextHTML)
			return ctx.SendString(s.RapiDoc())
		})
	}

	if opt.swaggerPath != nil {
		s.App.Get(*opt.swaggerPath, func(ctx *fiber.Ctx) error {
			ctx.Set(fiber.HeaderContentType, fiber.MIMETextHTML)
			return ctx.SendString(s.Swagger())
		})
	}

	if opt.redocPath != nil {
		s.App.Get(*opt.redocPath, func(ctx *fiber.Ctx) error {
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
	idBuilder := strings.Builder{}
	for _, p := range strings.Split(path, "/") {
		if p != "" {
			idBuilder.WriteString(cases.Title(language.English).String(p))
		}
	}
	idBuilder.WriteString(method)
	id := strings.ReplaceAll(idBuilder.String(), ":", "")
	return s.newOperation(path, method, handlers...).SetSummary(summary).SetOperationID(id)
}
