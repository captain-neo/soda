package soda

import (
	"context"
	"log"
	"sync"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/utils"
)

type Info = openapi3.Info
type Contact = openapi3.Contact
type License = openapi3.License

type Soda struct {
	fiber *fiber.App

	spec         []byte
	specOnce     sync.Once
	oaiGenerator *OAIGenerator
}

func NewSodaWithFiber(f *fiber.App, info *Info) *Soda {
	soda := &Soda{
		fiber:        f,
		oaiGenerator: NewGenerator(info),
	}
	soda.fiber.Get("/openapi.json", func(ctx *fiber.Ctx) error {
		ctx.Set(fiber.HeaderContentType, fiber.MIMEApplicationJSONCharsetUTF8)
		return ctx.Send(soda.spec)
	})
	soda.fiber.Get("/redoc", func(ctx *fiber.Ctx) error {
		ctx.Set(fiber.HeaderContentType, fiber.MIMETextHTML)
		return ctx.SendString(redocHTML)
	})
	soda.fiber.Get("/swagger", func(ctx *fiber.Ctx) error {
		ctx.Set(fiber.HeaderContentType, fiber.MIMETextHTML)
		return ctx.SendString(swaggerHTML)
	})
	return soda
}

func NewSoda(info *Info) *Soda {
	return NewSodaWithFiber(fiber.New(), info)
}

func (s *Soda) Handle(path, method string, handlers ...fiber.Handler) *Operation {
	if len(handlers) == 0 {
		panic("empty handlers")
	}
	handler := handlers[len(handlers)-1]
	op := &Operation{
		soda:        s,
		path:        path,
		method:      utils.ToUpper(method),
		operation:   openapi3.NewOperation(),
		handler:     handler,
		middlewares: handlers[:len(handlers)-1],
	}
	op.operation.OperationID = toKebabCase(utils.ToLower(method) + getHandlerName(handler))
	return op
}

func (s *Soda) GET(path string, handlers ...fiber.Handler) *Operation {
	return s.Handle(path, "GET", handlers...)
}

func (s *Soda) POST(path string, handlers ...fiber.Handler) *Operation {
	return s.Handle(path, "POST", handlers...)
}

func (s *Soda) PUT(path string, handlers ...fiber.Handler) *Operation {
	return s.Handle(path, "PUT", handlers...)
}

func (s *Soda) PATCH(path string, handlers ...fiber.Handler) *Operation {
	return s.Handle(path, "PATCH", handlers...)
}

func (s *Soda) DELETE(path string, handlers ...fiber.Handler) *Operation {
	return s.Handle(path, "PATCH", handlers...)
}

func (s *Soda) Fiber() *fiber.App {
	return s.fiber
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
