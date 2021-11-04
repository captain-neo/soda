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

type ExampleParameters struct {
	Limit  int `oai:"in=query,name=limit,default=10"`
	Offset int `oai:"in=query,name=offset,default=1"`
}

type ExampleRequestBody struct {
	Int             int      `oai:"name=int" json:"int"`
	IntDefault      int      `oai:"name=int_default" json:"int_default"`
	IntSlice        []int    `oai:"name=int_slice" json:"int_slice"`
	IntSliceDefault []int    `oai:"name=int_slice_default" json:"int_slice_default"`
	String          string   `oai:"name=string" json:"string"`
	StringSlice     []string `oai:"name=string_slice" json:"string_slice"`
}

func exampleHandler(c *fiber.Ctx) error {
	// get parameter values
	parameters := c.Locals(soda.KeyParameter).(*ExampleParameters)
	// get request body values
	body := c.Locals(soda.KeyRequestBody).(*ExampleRequestBody)

	log.Println(parameters.Limit)
	log.Println(parameters.Offset)

	return c.Status(200).JSON(body)
}

func main() {
	log.SetFlags(log.Lshortfile)
	f := fiber.New(fiber.Config{
		ErrorHandler: func(ctx *fiber.Ctx, err error) error {
			if e, ok := err.(soda.ValidationError); ok {
				return ctx.Status(422).JSON(e)
			}
			return ctx.SendStatus(500)
		},
	})
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
		SetJSONRequestBody(ExampleRequestBody{}).
		SetParameters(ExampleParameters{}).
		AddJSONResponse(200, ExampleRequestBody{}).
		AddJSONResponse(422, soda.ValidationError{}).Mount()

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