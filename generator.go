package soda

import "github.com/getkin/kin-openapi/openapi3"

type OAIGenerator struct {
	openapi *openapi3.T
}

func NewGenerator(info *openapi3.Info) *OAIGenerator {
	return &OAIGenerator{
		openapi: &openapi3.T{
			OpenAPI: "3.0.3",
			Info:    info,
			Components: openapi3.Components{
				Schemas:       make(openapi3.Schemas),
				Responses:     make(openapi3.Responses),
				RequestBodies: make(openapi3.RequestBodies),
			},
		},
	}
}
