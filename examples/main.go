package main

import (
	"fmt"

	"github.com/captain-neo/soda"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/monitor"
	"github.com/gofiber/fiber/v2/middleware/requestid"
)

type ExampleRequestBody struct {
	String      string
	StringSlice []string
	Int         int `json:"int"`
}

type Auth struct {
	Token string `header:"Authorization" oai:"description=some JWT Token"`
}

type ExampleParameters struct {
	Auth
	Q      []string `query:"q" oai:"description=support list parameters"`
	Limit  int      `query:"limit" oai:"description=blabla"`
	Offset int      `query:"offset"`
}

type ExampleResponse struct {
	Parameters  *ExampleParameters  `json:"parameters"`
	RequestBody *ExampleRequestBody `json:"request_body"`
}
type ErrorResponse struct{}

func exampleHandler(c *fiber.Ctx) error {
	// get parameter values
	params := c.Locals(soda.KeyParameter).(*ExampleParameters)
	fmt.Println(params.Token, params.Limit, params.Offset, params.Q)
	// get request body values
	body := c.Locals(soda.KeyRequestBody).(*ExampleRequestBody)
	fmt.Println(body.Int)
	return c.Status(200).JSON(ExampleResponse{
		Parameters:  params,
		RequestBody: body,
	})
}

func main() {
	app := soda.New("soda_fiber", "0.1",
		soda.WithOpenAPISpec("/openapi.json"),
		soda.WithRapiDoc("/rapidoc"),
		soda.WithSwagger("/swagger"),
		soda.WithRedoc("/redoc"),
		soda.EnableValidateRequest(),
	)
	app.Use(logger.New(), requestid.New())
	app.Get("/monitor", monitor.New()).SetSummary("it's a monitor").OK()
	app.Post("/path", exampleHandler).
		SetOperationID("example-handler").
		SetJSONRequestBody(ExampleResponse{}).
		SetParameters(ExampleParameters{}).
		AddJSONResponse(200, ExampleResponse{}).
		AddJSONResponse(400, ErrorResponse{}).OK()

	_ = app.Listen(":8080")
}
