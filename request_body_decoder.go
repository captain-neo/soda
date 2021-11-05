package soda

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/gofiber/fiber/v2"
	"gopkg.in/yaml.v2"
)

// EncodingFn is a function that returns an encoding of a request body's part.
type EncodingFn func(partName string) *openapi3.Encoding

// BodyDecoder is an interface to decode a body of a request or response.
// An implementation must return a value that is a primitive, []interface{}, or map[string]interface{}.
type BodyDecoder func(*fiber.Ctx, *openapi3.SchemaRef, EncodingFn) (interface{}, error)

// bodyDecoders contains decoders for supported content types of a body.
// By default, there is content type "application/json" is supported only.
var bodyDecoders = make(map[string]BodyDecoder)

// RegisteredBodyDecoder returns the registered body decoder for the given content type.
//
// If no decoder was registered for the given content type, nil is returned.
// This call is not thread-safe: body decoders should not be created/destroyed by multiple goroutines.
func RegisteredBodyDecoder(contentType string) BodyDecoder {
	return bodyDecoders[contentType]
}

// RegisterBodyDecoder registers a request body's decoder for a content type.
//
// If a decoder for the specified content type already exists, the function replaces
// it with the specified decoder.
// This call is not thread-safe: body decoders should not be created/destroyed by multiple goroutines.
func RegisterBodyDecoder(contentType string, decoder BodyDecoder) {
	if contentType == "" {
		panic("contentType is empty")
	}
	if decoder == nil {
		panic("decoder is not defined")
	}
	bodyDecoders[contentType] = decoder
}

// UnregisterBodyDecoder dissociates a body decoder from a content type.
//
// Decoding this content type will result in an error.
// This call is not thread-safe: body decoders should not be created/destroyed by multiple goroutines.
func UnregisterBodyDecoder(contentType string) {
	if contentType == "" {
		panic("contentType is empty")
	}
	delete(bodyDecoders, contentType)
}

const prefixUnsupportedCT = "unsupported content type"

func parseMediaType(contentType string) string {
	i := strings.IndexByte(contentType, ';')
	if i < 0 {
		return contentType
	}
	return contentType[:i]
}

// decodeBody returns a decoded body.
// The function returns ParseError when a body is invalid.
func decodeBody(c *fiber.Ctx, schema *openapi3.SchemaRef, encFn EncodingFn) (interface{}, error) {
	contentType := c.Get(fiber.HeaderContentType)
	// if contentType == "" {
	// 	if _, ok := body.(*multipart.Part); ok {
	// 		contentType = "text/plain"
	// 	}
	// }
	mediaType := parseMediaType(contentType)
	decoder, ok := bodyDecoders[mediaType]
	if !ok {
		return nil, &ParseError{
			Kind:   KindUnsupportedFormat,
			Reason: fmt.Sprintf("%s %q", prefixUnsupportedCT, mediaType),
		}
	}
	value, err := decoder(c, schema, encFn)
	if err != nil {
		return nil, err
	}
	return value, nil
}

func init() {
	RegisterBodyDecoder("text/plain", plainBodyDecoder)
	RegisterBodyDecoder("application/json", jsonBodyDecoder)
	RegisterBodyDecoder("application/x-yaml", yamlBodyDecoder)
	RegisterBodyDecoder("application/yaml", yamlBodyDecoder)
	RegisterBodyDecoder("application/problem+json", jsonBodyDecoder)
	RegisterBodyDecoder("application/x-www-form-urlencoded", urlencodedBodyDecoder)
	RegisterBodyDecoder("multipart/form-data", multipartBodyDecoder)
	RegisterBodyDecoder("application/octet-stream", FileBodyDecoder)
}

func plainBodyDecoder(c *fiber.Ctx, schema *openapi3.SchemaRef, encFn EncodingFn) (interface{}, error) {
	return string(c.Body()), nil
}

func jsonBodyDecoder(c *fiber.Ctx, schema *openapi3.SchemaRef, encFn EncodingFn) (interface{}, error) {
	var value interface{}
	if err := json.Unmarshal(c.Body(), &value); err != nil {
		return nil, &ParseError{Kind: KindInvalidFormat, Cause: err}
	}
	return value, nil
}

func yamlBodyDecoder(c *fiber.Ctx, schema *openapi3.SchemaRef, encFn EncodingFn) (interface{}, error) {
	var value interface{}
	if err := yaml.Unmarshal(c.Body(), &value); err != nil {
		return nil, &ParseError{Kind: KindInvalidFormat, Cause: err}
	}
	return value, nil
}

func urlencodedBodyDecoder(c *fiber.Ctx, schema *openapi3.SchemaRef, encFn EncodingFn) (interface{}, error) {
	// Validate schema of request body.
	// By the OpenAPI 3 specification request body's schema must have type "object".
	// Properties of the schema describes individual parts of request body.
	if schema.Value.Type != "object" {
		return nil, errors.New("unsupported schema of request body")
	}
	for propName, propSchema := range schema.Value.Properties {
		switch propSchema.Value.Type {
		case "object":
			return nil, fmt.Errorf("unsupported schema of request body's property %q", propName)
		case "array":
			items := propSchema.Value.Items.Value
			if items.Type != "string" && items.Type != "integer" && items.Type != "number" && items.Type != "boolean" {
				return nil, fmt.Errorf("unsupported schema of request body's property %q", propName)
			}
		}
	}

	// Make an object value from form values.
	obj := make(map[string]interface{})
	dec := &urlValuesDecoder{values: c.Request().PostArgs()}
	for name, prop := range schema.Value.Properties {
		var (
			value interface{}
			enc   *openapi3.Encoding
		)
		if encFn != nil {
			enc = encFn(name)
		}
		sm := enc.SerializationMethod()

		value, err := decodeValue(dec, name, sm, prop, false)
		if err != nil {
			return nil, err
		}
		obj[name] = value
	}

	return obj, nil
}

func multipartBodyDecoder(c *fiber.Ctx, schema *openapi3.SchemaRef, encFn EncodingFn) (interface{}, error) {
	if schema.Value.Type != "object" {
		return nil, errors.New("unsupported schema of request body")
	}

	// Parse form.
	form, err := c.MultipartForm()
	if err != nil {
		return nil, err
	}
	values := make(map[string][]interface{}, len(form.Value))

	for name := range form.Value {
		var enc *openapi3.Encoding
		if encFn != nil {
			enc = encFn(name)
		}

		subEncFn := func(string) *openapi3.Encoding { return enc }
		// If the property's schema has type "array" it is means that the form contains a few parts with the same name.
		// Every such part has a type that is defined by an items schema in the property's schema.
		var valueSchema *openapi3.SchemaRef
		var exists bool
		// mr := multipart.NewReader(body, params["boundary"])
		// p, _ := mr.NextPart()
		// p.FormName()
		valueSchema, exists = schema.Value.Properties[name]
		if !exists {
			anyProperties := schema.Value.AdditionalPropertiesAllowed
			if anyProperties != nil {
				switch *anyProperties {
				case true:
					//additionalProperties: true
					continue
				default:
					//additionalProperties: false
					return nil, &ParseError{Kind: KindOther, Cause: fmt.Errorf("part %s: undefined", name)}
				}
			}
			if schema.Value.AdditionalProperties == nil {
				return nil, &ParseError{Kind: KindOther, Cause: fmt.Errorf("part %s: undefined", name)}
			}
			valueSchema, exists = schema.Value.AdditionalProperties.Value.Properties[name]
			if !exists {
				return nil, &ParseError{Kind: KindOther, Cause: fmt.Errorf("part %s: undefined", name)}
			}
		}
		if valueSchema.Value.Type == "array" {
			valueSchema = valueSchema.Value.Items
		}

		var value interface{}
		if value, err = decodeBody(c, valueSchema, subEncFn); err != nil {
			if v, ok := err.(*ParseError); ok {
				return nil, &ParseError{path: []interface{}{name}, Cause: v}
			}
			return nil, fmt.Errorf("part %s: %s", name, err)
		}
		values[name] = append(values[name], value)
	}

	allTheProperties := make(map[string]*openapi3.SchemaRef)
	for k, v := range schema.Value.Properties {
		allTheProperties[k] = v
	}
	if schema.Value.AdditionalProperties != nil {
		for k, v := range schema.Value.AdditionalProperties.Value.Properties {
			allTheProperties[k] = v
		}
	}
	// Make an object value from form values.
	obj := make(map[string]interface{})
	for name, prop := range allTheProperties {
		vv := values[name]
		if len(vv) == 0 {
			continue
		}
		if prop.Value.Type == "array" {
			obj[name] = vv
		} else {
			obj[name] = vv[0]
		}
	}

	return obj, nil
}

// FileBodyDecoder is a body decoder that decodes a file body to a string.
func FileBodyDecoder(c *fiber.Ctx, schema *openapi3.SchemaRef, encFn EncodingFn) (interface{}, error) {
	return string(c.Body()), nil
}
