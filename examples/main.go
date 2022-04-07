package main

import (
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/monitor"
	"github.com/panicneo/soda"
)

type Item struct {
	Name string `json:"name" oai:"description=名字;example=Item 1"`
	Type string `json:"type" oai:"description=类型;"`
}

type Body struct {
	Parameters []Item `json:"parameters"`
	ReturnType string `json:"return_type"`
	Body       string `json:"body"`
}

func main() {
	app := soda.New("soda_fiber", "0.1",
		soda.WithOpenAPISpec("/openapi.json"),
		soda.WithRapiDoc("/rapidoc"),
		soda.WithSwagger("/swagger"),
		soda.WithRedoc("/redoc"),
		soda.EnableValidateRequest(),
	)
	app.Use(logger.New())
	app.Get("/monitor", monitor.New()).SetSummary("it's a monitor").OK()

	app.Post("/", TestPost).
		SetJSONRequestBody(Body{}).
		AddJSONResponse(200, Body{}).
		OK()
	_ = app.Listen(":3000")
}

func TestPost(ctx *fiber.Ctx) error {
	body := ctx.Locals(soda.KeyRequestBody).(*Body)
	return ctx.Status(200).JSON(body)
}
