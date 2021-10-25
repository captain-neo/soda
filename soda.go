package soda

import (
	"context"
	"log"
	"sync"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/gofiber/fiber/v2"
)

type Info = openapi3.Info
type Contact = openapi3.Contact
type License = openapi3.License

type Soda struct {
	App *fiber.App

	spec         []byte
	specOnce     sync.Once
	oaiGenerator *OAIGenerator
}

func NewSodaWithFiber(f *fiber.App, info *Info) *Soda {
	soda := &Soda{
		App:          f,
		oaiGenerator: NewGenerator(info),
	}
	soda.App.Get("/openapi.json", func(ctx *fiber.Ctx) error {
		soda.specOnce.Do(func() {
			if err := soda.oaiGenerator.openapi.Validate(context.TODO()); err != nil {
				log.Fatalln(err)
			}
			spec, err := soda.oaiGenerator.openapi.MarshalJSON()
			if err != nil {
				log.Fatalln(err)
			}
			soda.spec = spec
		})
		ctx.Response().Header.SetContentType(fiber.MIMEApplicationJSON)
		return ctx.Send(soda.spec)
	})
	soda.App.Get("/redoc", func(ctx *fiber.Ctx) error {
		ctx.Response().Header.SetContentType(fiber.MIMETextHTML)
		return ctx.SendString(redocHTML)
	})
	soda.App.Get("/swagger", func(ctx *fiber.Ctx) error {
		ctx.Response().Header.SetContentType(fiber.MIMETextHTML)
		return ctx.SendString(swaggerHTML)
	})
	return soda
}

func NewSoda(info *Info) *Soda {
	return NewSodaWithFiber(fiber.New(), info)
}
