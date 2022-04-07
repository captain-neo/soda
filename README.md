# Soda [WIP]

soda := [OpenAPI3.0](https://swagger.io/specification) + [fiber](https://github.com/gofiber/fiber)

> inspired on [kin-openapi3](https://github.com/getkin/kin-openapi) and [fizz](https://github.com/wI2L/fizz)


### Example
```go
package main

import (
	"log"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/requestid"
	"github.com/panicneo/soda"
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
	Limit  int `oai:"in=query,default=10"`
	Offset int `oai:"in=query,default=1"`
}

type ExampleResponse struct {
	Int             int   `json:"int,omitempty"`
	IntDefault      int   `json:"int_default,omitempty"`
	IntSlice        []int `json:"int_slice,omitempty"`
	IntSliceDefault []int
	String          string   `json:"string,omitempty"`
	StringSlice     []string `json:"string_slice,omitempty"`
}
type ErrorResponse struct{}

func exampleHandler(c *fiber.Ctx) error {
  // get parameter values
	parameters := c.Locals(soda.KeyParameter).(*ExampleParameters)
  // get request body values
	body := c.Locals(soda.KeyRequestBody).(*ExampleRequestBody)
	return nil
}

func main() {
	f := fiber.New(fiber.Config{})
	f.Use(logger.New(), requestid.New())
	app := soda.NewSodaWithFiber(f, &soda.Info{
		Title:          "Example Soda APP",
		Description:    "an example of soda app",
		TermsOfService: "",
		Contact: &soda.Contact{
			Name:  "admin",
			Email: "admin@example.com",
		},
		License: &soda.License{
			Name: "MIT",
		},
		Version: "1.0.0",
	})
	app.NewOperation("/path", "POST", exampleHandler).
		SetOperationID("example-handler").
		SetJSONRequestBody(ExampleResponse{}).
		SetParameters(ExampleParameters{}).
		AddJSONResponse(200, ExampleResponse{}).
		AddJSONResponse(400, ErrorResponse{}).Mount()

	app.App.Listen(":8080")
}
```

check your openapi3 spec file at http://localhost:8080/openapi.json

and embed openapi3 renderer
- redocly: http://localhost:8080/redoc
- swagger: http://localhost:8080/swagger


### TODO:
 - [ ] more tests
 - [ ] more example && examples
