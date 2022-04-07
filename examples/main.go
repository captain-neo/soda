package main

import (
	"github.com/captain-neo/soda"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/monitor"
	"github.com/gofiber/fiber/v2/middleware/requestid"
)

type ExampleRequestBody struct {
	Int             int   `json:"int,omitempty"`
	IntDefault      int   `json:"int_default,omitempty"`
	IntSlice        []int `json:"int_slice,omitempty"`
	IntSliceDefault []int
	String          string
	StringSlice     []string
}

type ExampleParameters struct {
	Limit  int `oai:",default=10"`
	Offset int `oai:"in=query,default=1"`
}

type ExampleResponse struct {
	Parameters  *ExampleParameters  `json:"parameters"`
	RequestBody *ExampleRequestBody `json:"request_body"`
}
type ErrorResponse struct{}

func exampleHandler(c *fiber.Ctx) error {
	// get parameter values
	parameters := c.Locals(soda.KeyParameter).(*ExampleParameters)
	// get request body values
	body := c.Locals(soda.KeyRequestBody).(*ExampleRequestBody)
	return c.Status(200).JSON(ExampleResponse{
		Parameters:  parameters,
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

	_ = app.App.Listen(":8080")
}
