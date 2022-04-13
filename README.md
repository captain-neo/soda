# Soda

soda := [OpenAPI3.0](https://swagger.io/specification) + [fiber](https://github.com/gofiber/fiber)

> inspired on [kin-openapi3](https://github.com/getkin/kin-openapi) and [fizz](https://github.com/wI2L/fizz)


### Example
```go
package main

import (
	"github.com/captain-neo/soda"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/monitor"
	"github.com/gofiber/fiber/v2/middleware/requestid"
)

type ExampleRequestBody struct {
	Int             int       `json:"int"`
	String          string
	StringSlice     []string
}

type Auth struct {
  Token string `header:"Authorization" oai:"description=some JWT Token"`
}

type ExampleParameters struct {
  Auth
	Limit  int      `query:"limit" oai:"description=blabla"`
	Offset int      `query:"offset"`
  Q      []string `query:"q" oai:"description=support list parameters"`
}

type ExampleResponse struct {
	Parameters  *ExampleParameters  `json:"parameters"`
	RequestBody *ExampleRequestBody `json:"request_body"`
}
type ErrorResponse struct{}

func exampleHandler(c *fiber.Ctx) error {
	// get parameter values
	params := c.Locals(soda.KeyParameter).(*ExampleParameters)
  fmt.Println(params.Authorization, params.Limit, params.Offset, params.Q)
	// get request body values
	body := c.Locals(soda.KeyRequestBody).(*ExampleRequestBody)
  fmt.Println(body.Int)
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
```

check your openapi3 spec file at http://localhost:8080/openapi.json

and embed openapi3 renderer
- redocly: http://localhost:8080/redoc
- swagger: http://localhost:8080/swagger
- rapidoc: http://localhost:8080/rapidoc


### TODO:
 - [ ] need add more examples to cover all the features
 - [ ] support app.Group() or app.Use() maybe? need design
