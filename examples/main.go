package main

import (
	"fmt"

	"github.com/captain-neo/soda"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/requestid"
)

type Auth struct {
	Token string `header:"Authorization" oai:"description=some JWT Token"`
}

type ExParameters struct {
	Auth
	ID     int      `json:"id"     validate:"required" oai:"description=some id"                 path:"id"`
	Q      []string `json:"q"      validate:"required" oai:"description=support list parameters"           query:"q"`
	Limit  int      `json:"limit"  validate:"required" oai:"description=limit params"                      query:"limit"`
	Offset int8     `json:"offset" validate:"required" oai:"description=offset params"                     query:"offset"`
	// Cookie string   `validate:"required" oai:"description=i am the cookie"                                  cookie:"cookie"`
}

type ExBody struct {
	ID     int      `json:"id"     validate:"required" oai:"description=我是结构体里面的ID"`
	Q      []string `json:"q"      validate:"required" oai:"description=这是一个支持模糊查询的字符串列表"`
	Limit  int      `json:"limit"  validate:"required" oai:"description=limit;maximum=20"`
	Offset int      `json:"offset" validate:"required" oai:"description=offset"`
}

type ExampleResponse struct {
	Parameters  *ExParameters `json:"parameters"`
	RequestBody *ExBody       `json:"request_body"`
}
type ErrorResponse struct{}

func exampleGet(c *fiber.Ctx) error {
	params := c.Locals(soda.KeyParameter).(*ExParameters)
	fmt.Println(params.Token, params.Limit, params.Offset, params.Q)
	return c.Status(200).JSON(ExampleResponse{
		Parameters: params,
	})
}

func examplePost(c *fiber.Ctx) error {
	body := c.Locals(soda.KeyRequestBody).(*ExBody)
	fmt.Println(body.Limit, body.Offset, body.Q)
	return c.Status(200).JSON(ExampleResponse{
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
	app.Get("/test/:id", exampleGet).
		SetParameters(ExParameters{}).
		AddJSONResponse(200, ExampleResponse{}).
		AddJSONResponse(400, ErrorResponse{}).OK()

	app.Post("/test", examplePost).
		SetJSONRequestBody(ExBody{}).
		AddJSONResponse(200, ExampleResponse{}).
		AddJSONResponse(400, ErrorResponse{}).OK()
	_ = app.Listen(":8080")
}
