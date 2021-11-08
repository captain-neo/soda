package soda_test

import (
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/panicneo/soda"
)

func TestSodaApp(t *testing.T) {
	CreateSoda := func(c *fiber.Ctx) error {
		return c.SendString("hello")
	}
	UpdateSoda := func(c *fiber.Ctx) error {
		return c.SendString("hello")
	}

	type Parameters struct {
		Int             int      `oai:"int,in=query,name=int" json:"int,omitempty"`
		IntDefault      int      `oai:"int,in=query,name=int_default,default=3" json:"int_default,omitempty"`
		IntSlice        []int    `oai:"int,in=query,name=int_slice,explode=0" json:"int_slice,omitempty"`
		IntSliceDefault []int    `oai:"int,in=query,enum=1 2 3"`
		String          string   `oai:"int,in=query,name=string" json:"string,omitempty"`
		StringSlice     []string `oai:"int,in=query,name=string_slice,explode=f" json:"string_slice,omitempty"`
	}

	type EmbedStruct struct {
		A string
	}

	type Body struct {
		EmbedStruct
		B *Body
	}
	s := soda.NewSoda(&soda.Info{
		Title:       "Title",
		Description: "Desc",
		Version:     "1.0.0",
	})
	s.POST("/soda", CreateSoda).
		SetOperationID("create-soda").
		SetParameters(Parameters{}).
		SetJSONRequestBody(Body{}).
		AddJSONResponse(201, Body{}).
		AddJSONResponse(204, nil).Mount()
	s.PUT("/soda", UpdateSoda).
		SetOperationID("update-soda").
		SetParameters(Parameters{}).
		SetJSONRequestBody(Body{}).
		AddJSONResponse(200, Body{}).Mount()
}
