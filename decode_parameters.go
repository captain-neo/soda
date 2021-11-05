package soda

import (
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	oai "github.com/getkin/kin-openapi/openapi3"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/utils"
	"github.com/valyala/fasthttp"
)

// Decodes a parameter defined via the content property as an object. It uses
// the user specified decoder, or our build-in decoder for application/json.
func decodeContentParameter(param *oai.Parameter, c *fiber.Ctx) (value interface{}, schema *oai.Schema, err error) {
	var paramValues []string
	var found bool
	switch param.In {
	case oai.ParameterInPath:
		if v := c.Params(param.Name); v != "" {
			found = true
			paramValues = []string{v}
		}
	case oai.ParameterInQuery:
		args := c.Request().URI().QueryArgs()
		if args.Has(param.Name) {
			bs := args.PeekMulti(param.Name)
			paramValues = make([]string, 0, len(bs))
			for _, b := range bs {
				paramValues = append(paramValues, string(b))
			}
			found = true
		}
	case oai.ParameterInHeader:
		if paramValue := c.Get(param.Name); paramValue != "" {
			paramValues = []string{paramValue}
			found = true
		}
	case oai.ParameterInCookie:
		if paramValue := c.Cookies(param.Name); paramValue != "" {
			paramValues = []string{paramValue}
			found = true
		}
	default:
		err = OpenAPISpecError{
			Field:    param.Name,
			Position: "parameter",
			Reason:   "unknown parameter.in type",
		}
		return
	}

	if !found {
		if param.Required {
			err = ValidationError{
				Field:    param.Name,
				Position: "parameter",
				Reason:   "field is required",
			}
		}
		return
	}

	value, schema, err = defaultContentParameterDecoder(param, paramValues)
	return
}

//nolint:funlen
func defaultContentParameterDecoder(param *oai.Parameter, values []string) (
	outValue interface{}, outSchema *oai.Schema, err error) {
	// Only query parameters can have multiple values.
	if len(values) > 1 && param.In != oai.ParameterInQuery {
		err = OpenAPISpecError{
			Position: "parameter",
			Field:    param.Name,
			Reason:   fmt.Sprintf("%s parameter cannot have multiple values", param.In),
		}
		return
	}

	content := param.Content
	if content == nil {
		err = OpenAPISpecError{
			Position: "parameter",
			Field:    param.Name,
			Reason:   "expected to have content",
		}
		return
	}

	// We only know how to decode a parameter if it has one content, application/json
	if len(content) != 1 {
		err = OpenAPISpecError{
			Position: "parameter",
			Field:    param.Name,
			Reason:   "multiple content cType2Schema found",
		}
		return
	}

	mt := content.Get("application/json")
	if mt == nil {
		err = OpenAPISpecError{
			Position: "parameter",
			Field:    param.Name,
			Reason:   "no content schema found",
		}
		return
	}
	outSchema = mt.Schema.Value

	if len(values) == 1 {
		if err = json.Unmarshal([]byte(values[0]), &outValue); err != nil {
			err = ValidationError{
				Position: "parameter",
				Field:    param.Name,
				Reason:   "unmarshalling failed",
			}
			return
		}
	} else {
		outArray := make([]interface{}, 0, len(values))
		for _, v := range values {
			var item interface{}
			if err = json.Unmarshal([]byte(v), &item); err != nil {
				err = ValidationError{
					Position: "parameter",
					Field:    param.Name,
					Reason:   "unmarshalling failed",
				}
				return
			}
			outArray = append(outArray, item)
		}
		outValue = outArray
	}
	return
}

type valueDecoder interface {
	DecodePrimitive(param string, sm *oai.SerializationMethod, schema *oai.SchemaRef) (interface{}, error)
	DecodeArray(param string, sm *oai.SerializationMethod, schema *oai.SchemaRef) ([]interface{}, error)
	DecodeObject(param string, sm *oai.SerializationMethod, schema *oai.SchemaRef) (map[string]interface{}, error)
}

// decodeStyledParameter returns a value of an operation's parameter from HTTP request for
// parameters defined using the style format.
// The function returns ParseError when HTTP request contains an invalid value of a parameter.
func decodeStyledParameter(param *oai.Parameter, c *fiber.Ctx) (interface{}, error) {
	sm, err := param.SerializationMethod()
	if err != nil {
		return nil, err
	}

	var dec valueDecoder
	switch param.In {
	case oai.ParameterInPath:
		if len(c.Route().Params) == 0 {
			return nil, nil
		}
		params := make(map[string]string)
		for _, p := range c.Route().Params {
			params[p] = c.Params(p)
		}
		dec = &pathParamDecoder{pathParams: params}
	case oai.ParameterInQuery:
		args := c.Request().URI().QueryArgs()
		if args.Len() == 0 {
			return nil, nil
		}
		dec = &urlValuesDecoder{values: args}
	case oai.ParameterInHeader:
		dec = &headerParamDecoder{header: &c.Request().Header}
	case oai.ParameterInCookie:
		dec = &cookieParamDecoder{header: &c.Request().Header}
	default:
		return nil, OpenAPISpecError{
			Position: "parameter",
			Field:    param.Name,
			Reason:   "unsupported parameter's 'in': " + param.In,
		}
	}

	return decodeValue(dec, param.Name, sm, param.Schema, param.Required)
}

//nolint:funlen
func decodeValue(dec valueDecoder, param string, sm *oai.SerializationMethod, schema *oai.SchemaRef, required bool) (interface{}, error) {
	var decodeFn func(param string, sm *oai.SerializationMethod, schema *oai.SchemaRef) (interface{}, error)

	if len(schema.Value.AllOf) > 0 {
		var value interface{}
		var err error
		for _, sr := range schema.Value.AllOf {
			value, err = decodeValue(dec, param, sm, sr, required)
			if value == nil || err != nil {
				return value, err
			}
		}
		return value, err
	}

	if len(schema.Value.AnyOf) > 0 {
		for _, sr := range schema.Value.AnyOf {
			value, _ := decodeValue(dec, param, sm, sr, required)
			if value != nil {
				return value, nil
			}
		}
		if required {
			return nil, ValidationError{
				Position: "parameter",
				Field:    param,
				Reason:   "decoding anyOf failed",
			}
		}
		return nil, nil
	}

	if len(schema.Value.OneOf) > 0 {
		isMatched := 0
		var value interface{}
		for _, sr := range schema.Value.OneOf {
			v, _ := decodeValue(dec, param, sm, sr, required)
			if v != nil {
				value = v
				isMatched++
			}
		}
		if isMatched == 1 {
			return value, nil
		} else if isMatched > 1 {
			return nil, ValidationError{
				Position: "parameter",
				Field:    param,
				Reason:   fmt.Sprintf("decoding oneOf failed: %d schemas matched", isMatched),
			}
		}
		if required {
			return nil, ValidationError{
				Position: "parameter",
				Field:    param,
				Reason:   "decoding oneOf failed, field is required",
			}
		}
		return nil, nil
	}

	if schema.Value.Not != nil {
		// TODO(decode not): handle decoding "not" JSON Schema
		return nil, OpenAPISpecError{
			Position: "parameter",
			Field:    param,
			Reason:   "'not' property not implemented",
		}
	}

	if schema.Value.Type != "" {
		switch schema.Value.Type {
		case typeArray:
			decodeFn = func(param string, sm *oai.SerializationMethod, schema *oai.SchemaRef) (interface{}, error) {
				return dec.DecodeArray(param, sm, schema)
			}
		case typeObject:
			decodeFn = func(param string, sm *oai.SerializationMethod, schema *oai.SchemaRef) (interface{}, error) {
				return dec.DecodeObject(param, sm, schema)
			}
		default:
			decodeFn = dec.DecodePrimitive
		}
		return decodeFn(param, sm, schema)
	}

	return nil, nil
}

// pathParamDecoder decodes values of path parameters.
type pathParamDecoder struct {
	pathParams map[string]string
}

func (d *pathParamDecoder) DecodePrimitive(param string, sm *oai.SerializationMethod, schema *oai.SchemaRef) (interface{}, error) {
	var prefix string
	switch sm.Style {
	case oai.SerializationSimple:
		// A prefix is empty for style oai.SerializationSimple:.
	case oai.SerializationLabel:
		prefix = "."
	case oai.SerializationMatrix:
		prefix = ";" + param + "="
	default:
		return nil, SerializationMethodError{
			Style:   sm.Style,
			Explode: sm.Explode,
		}
	}

	if d.pathParams == nil {
		// HTTP request does not contain a value of the target path parameter.
		return nil, nil
	}
	raw, ok := d.pathParams[param]
	if !ok || raw == "" {
		// HTTP request does not contain a value of the target path parameter.
		return nil, nil
	}
	src, err := cutPrefix(raw, prefix)
	if err != nil {
		return nil, err
	}
	return parsePrimitive(src, schema)
}

func (d *pathParamDecoder) DecodeArray(param string, sm *oai.SerializationMethod, schema *oai.SchemaRef) ([]interface{}, error) {
	var prefix, delim string
	switch {
	case sm.Style == oai.SerializationSimple:
		delim = ","
	case sm.Style == oai.SerializationLabel && !sm.Explode:
		prefix = "."
		delim = ","
	case sm.Style == oai.SerializationLabel && sm.Explode:
		prefix = "."
		delim = "."
	case sm.Style == oai.SerializationMatrix && !sm.Explode:
		prefix = ";" + param + "="
		delim = ","
	case sm.Style == oai.SerializationMatrix && sm.Explode:
		prefix = ";" + param + "="
		delim = ";" + param + "="
	default:
		return nil, SerializationMethodError{
			Style:   sm.Style,
			Explode: sm.Explode,
		}
	}

	if d.pathParams == nil {
		// HTTP request does not contain a value of the target path parameter.
		return nil, nil
	}
	raw, ok := d.pathParams[param]
	if !ok || raw == "" {
		// HTTP request does not contain a value of the target path parameter.
		return nil, nil
	}
	src, err := cutPrefix(raw, prefix)
	if err != nil {
		return nil, err
	}
	return parseArray(strings.Split(src, delim), schema)
}

func (d *pathParamDecoder) DecodeObject(param string, sm *oai.SerializationMethod, schema *oai.SchemaRef) (map[string]interface{}, error) {
	var prefix, propsDelim, valueDelim string
	switch {
	case sm.Style == oai.SerializationSimple && !sm.Explode:
		propsDelim = ","
		valueDelim = ","
	case sm.Style == oai.SerializationSimple && sm.Explode:
		propsDelim = ","
		valueDelim = "="
	case sm.Style == oai.SerializationLabel && !sm.Explode:
		prefix = "."
		propsDelim = ","
		valueDelim = ","
	case sm.Style == oai.SerializationLabel && sm.Explode:
		prefix = "."
		propsDelim = "."
		valueDelim = "="
	case sm.Style == oai.SerializationMatrix && !sm.Explode:
		prefix = ";" + param + "="
		propsDelim = ","
		valueDelim = ","
	case sm.Style == oai.SerializationMatrix && sm.Explode:
		prefix = ";"
		propsDelim = ";"
		valueDelim = "="
	default:
		return nil, SerializationMethodError{
			Style:   sm.Style,
			Explode: sm.Explode,
		}
	}

	if d.pathParams == nil {
		// HTTP request does not contain a value of the target path parameter.
		return nil, nil
	}
	raw, ok := d.pathParams[param]
	if !ok || raw == "" {
		// HTTP request does not contain a value of the target path parameter.
		return nil, nil
	}
	src, err := cutPrefix(raw, prefix)
	if err != nil {
		return nil, err
	}
	props, err := propsFromString(src, propsDelim, valueDelim)
	if err != nil {
		return nil, err
	}
	return makeObject(props, schema)
}

// cutPrefix validates that a raw value of a path parameter has the specified prefix,
// and returns a raw value without the prefix.
func cutPrefix(raw, prefix string) (string, error) {
	if prefix == "" {
		return raw, nil
	}
	if len(raw) < len(prefix) || raw[:len(prefix)] != prefix {
		return "", &ParseError{
			Kind:   KindInvalidFormat,
			Value:  raw,
			Reason: fmt.Sprintf("a value must be prefixed with %q", prefix),
		}
	}
	return raw[len(prefix):], nil
}

// urlValuesDecoder decodes values of query parameters.
type urlValuesDecoder struct {
	values *fasthttp.Args
}

func (d *urlValuesDecoder) DecodePrimitive(param string, sm *oai.SerializationMethod, schema *oai.SchemaRef) (interface{}, error) {
	if sm.Style != oai.SerializationForm {
		return nil, SerializationMethodError{
			Style:   sm.Style,
			Explode: sm.Explode,
		}
	}

	values := d.values.PeekMulti(param)
	if len(values) == 0 {
		// HTTP request does not contain a value of the target query parameter.
		return nil, nil
	}
	return parsePrimitive(string(values[0]), schema)
}

func (d *urlValuesDecoder) DecodeArray(param string, sm *oai.SerializationMethod, schema *oai.SchemaRef) ([]interface{}, error) {
	if sm.Style == oai.SerializationDeepObject {
		return nil, SerializationMethodError{
			Style:   sm.Style,
			Explode: sm.Explode,
		}
	}

	bs := d.values.PeekMulti(param)
	values := make([]string, 0, len(bs))
	for i := range bs {
		values = append(values, utils.UnsafeString(bs[i]))
	}
	if len(values) == 0 {
		// HTTP request does not contain a value of the target query parameter.
		return nil, nil
	}
	if !sm.Explode {
		var delim string
		switch sm.Style {
		case oai.SerializationForm:
			delim = ","
		case oai.SerializationSpaceDelimited:
			delim = " "
		case oai.SerializationPipeDelimited:
			delim = "|"
		}
		values = strings.Split(values[0], delim)
	}
	return parseArray(values, schema)
}

func (d *urlValuesDecoder) DecodeObject(param string, sm *oai.SerializationMethod, schema *oai.SchemaRef) (map[string]interface{}, error) {
	var propsFn func(*fasthttp.Args) (map[string]string, error)
	switch sm.Style {
	case oai.SerializationForm:
		propsFn = func(args *fasthttp.Args) (map[string]string, error) {
			if args.Len() == 0 {
				// HTTP request does not contain query parameters.
				return nil, nil
			}
			if sm.Explode {
				props := make(map[string]string)
				args.VisitAll(func(key, value []byte) {
					props[string(key)] = string(value)
				})
				return props, nil
			}
			values := args.PeekMulti(param)
			if len(values) == 0 {
				// HTTP request does not contain a value of the target query parameter.
				return nil, nil
			}
			return propsFromString(string(values[0]), ",", ",")
		}
	case oai.SerializationDeepObject:
		propsFn = func(args *fasthttp.Args) (map[string]string, error) {
			props := make(map[string]string)
			params := make(map[string][]string)
			args.VisitAll(func(k, v []byte) {
				params[utils.UnsafeString(k)] = append(params[utils.UnsafeString(k)], utils.UnsafeString(v))
			})
			for key, values := range params {
				groups := regexp.MustCompile(fmt.Sprintf("%s\\[(.+?)\\]", param)).FindAllStringSubmatch(key, -1)
				if len(groups) == 0 {
					// A query parameter's rawName does not match the required format, so skip it.
					continue
				}
				props[groups[0][1]] = values[0]
			}
			if len(props) == 0 {
				// HTTP request does not contain query parameters encoded by rules of style oai.SerializationDeepObject.
				return nil, nil
			}
			return props, nil
		}
	default:
		return nil, SerializationMethodError{
			Style:   sm.Style,
			Explode: sm.Explode,
		}
	}

	props, err := propsFn(d.values)
	if err != nil {
		return nil, err
	}
	if props == nil {
		return nil, nil
	}
	return makeObject(props, schema)
}

// headerParamDecoder decodes values of header parameters.
type headerParamDecoder struct {
	header *fasthttp.RequestHeader
}

func (d *headerParamDecoder) DecodePrimitive(param string, sm *oai.SerializationMethod, schema *oai.SchemaRef) (interface{}, error) {
	if sm.Style != oai.SerializationSimple {
		return nil, SerializationMethodError{
			Style:   sm.Style,
			Explode: sm.Explode,
		}
	}

	raw := d.header.Peek(param)
	return parsePrimitive(string(raw), schema)
}

func (d *headerParamDecoder) DecodeArray(param string, sm *oai.SerializationMethod, schema *oai.SchemaRef) ([]interface{}, error) {
	if sm.Style != oai.SerializationSimple {
		return nil, SerializationMethodError{
			Style:   sm.Style,
			Explode: sm.Explode,
		}
	}

	raw := d.header.Peek(param)
	if len(raw) == 0 {
		// HTTP request does not contains a corresponding header
		return nil, nil
	}
	return parseArray(strings.Split(string(raw), ","), schema)
}

func (d *headerParamDecoder) DecodeObject(param string, sm *oai.SerializationMethod, schema *oai.SchemaRef) (map[string]interface{}, error) {
	if sm.Style != oai.SerializationSimple {
		return nil, SerializationMethodError{
			Style:   sm.Style,
			Explode: sm.Explode,
		}
	}
	valueDelim := ","
	if sm.Explode {
		valueDelim = "="
	}

	raw := d.header.Peek(param)
	if len(raw) == 0 {
		// HTTP request does not contain a corresponding header.
		return nil, nil
	}
	props, err := propsFromString(utils.UnsafeString(raw), ",", valueDelim)
	if err != nil {
		return nil, err
	}
	return makeObject(props, schema)
}

// cookieParamDecoder decodes values of cookie parameters.
type cookieParamDecoder struct {
	header *fasthttp.RequestHeader
}

func (d *cookieParamDecoder) DecodePrimitive(param string, sm *oai.SerializationMethod, schema *oai.SchemaRef) (interface{}, error) {
	if sm.Style != oai.SerializationForm {
		return nil, SerializationMethodError{
			Style:   sm.Style,
			Explode: sm.Explode,
		}
	}

	cookie := d.header.Cookie(param)
	if len(cookie) == 0 {
		// HTTP request does not contain a corresponding cookie.
		return nil, nil
	}
	return parsePrimitive(string(cookie), schema)
}

func (d *cookieParamDecoder) DecodeArray(param string, sm *oai.SerializationMethod, schema *oai.SchemaRef) ([]interface{}, error) {
	if sm.Style != oai.SerializationForm || sm.Explode {
		return nil, SerializationMethodError{
			Style:   sm.Style,
			Explode: sm.Explode,
		}
	}

	cookie := d.header.Cookie(param)
	if len(cookie) == 0 {
		// HTTP request does not contain a corresponding cookie.
		return nil, nil
	}
	return parseArray(strings.Split(string(cookie), ","), schema)
}

func (d *cookieParamDecoder) DecodeObject(param string, sm *oai.SerializationMethod, schema *oai.SchemaRef) (map[string]interface{}, error) {
	if sm.Style != oai.SerializationForm || sm.Explode {
		return nil, SerializationMethodError{
			Style:   sm.Style,
			Explode: sm.Explode,
		}
	}

	cookie := d.header.Cookie(param)
	if len(cookie) == 0 {
		// HTTP request does not contain a corresponding cookie.
		return nil, nil
	}
	props, err := propsFromString(string(cookie), ",", ",")
	if err != nil {
		return nil, err
	}
	return makeObject(props, schema)
}

// propsFromString returns a properties map that is created by splitting a source string by propDelim and valueDelim.
// The source string must have a valid format: pairs <propName><valueDelim><propValue> separated by <propDelim>.
// The function returns an error when the source string has an invalid format.
func propsFromString(src, propDelim, valueDelim string) (map[string]string, error) {
	props := make(map[string]string)
	pairs := strings.Split(src, propDelim)

	// When propDelim and valueDelim is equal the source string follow the next rule:
	// every even item of pairs is a property's rawName, and the subsequent odd item is a property's value.
	if propDelim == valueDelim {
		// Taking into account the rule above, a valid source string must be split by propDelim
		// to an array with an even number of items.
		if len(pairs)%2 != 0 {
			return nil, &ParseError{
				Kind:   KindInvalidFormat,
				Value:  src,
				Reason: fmt.Sprintf("a value must be a list of object's properties in format \"rawName%svalue\" separated by %s", valueDelim, propDelim),
			}
		}
		for i := 0; i < len(pairs)/2; i++ {
			props[pairs[i*2]] = pairs[i*2+1]
		}
		return props, nil
	}

	// When propDelim and valueDelim is not equal the source string follow the next rule:
	// every item of pairs is a string that follows format <propName><valueDelim><propValue>.
	for _, pair := range pairs {
		prop := strings.Split(pair, valueDelim)
		if len(prop) != 2 {
			return nil, &ParseError{
				Kind:   KindInvalidFormat,
				Value:  src,
				Reason: fmt.Sprintf("a value must be a list of object's properties in format \"rawName%svalue\" separated by %s", valueDelim, propDelim),
			}
		}
		props[prop[0]] = prop[1]
	}
	return props, nil
}

// makeObject returns an object that contains properties from props.
// A value of every property is parsed as a primitive value.
// The function returns an error when an error happened while parse object's properties.
func makeObject(props map[string]string, schema *oai.SchemaRef) (map[string]interface{}, error) {
	obj := make(map[string]interface{})
	for propName, propSchema := range schema.Value.Properties {
		value, err := parsePrimitive(props[propName], propSchema)
		if err != nil {
			if errors.Is(err, ParseError{}) {
				return nil, ParseError{path: []interface{}{propName}, Cause: err}
			}
			return nil, fmt.Errorf("property %q: %w", propName, err)
		}
		obj[propName] = value
	}
	return obj, nil
}

// parseArray returns an array that contains items from a raw array.
// Every item is parsed as a primitive value.
// The function returns an error when an error happened while parse array's items.
func parseArray(raw []string, schemaRef *oai.SchemaRef) ([]interface{}, error) {
	value := make([]interface{}, 0, len(raw))
	for i, v := range raw {
		item, err := parsePrimitive(v, schemaRef.Value.Items)
		if err != nil {
			var v ParseError
			if ok := errors.Is(err, v); ok {
				return nil, ParseError{path: []interface{}{i}, Cause: v}
			}
			return nil, fmt.Errorf("item %d: %w", i, err)
		}
		value = append(value, item)
	}
	return value, nil
}

// parsePrimitive returns a value that is created by parsing a source string to a primitive type
// that is specified by a schema. The function returns nil when the source string is empty.
// The function panics when a schema has a non-primitive type.
func parsePrimitive(raw string, schema *oai.SchemaRef) (interface{}, error) {
	if len(raw) == 0 {
		return nil, nil
	}
	switch schema.Value.Type {
	case typeInteger:
		v, err := strconv.ParseFloat(raw, 64)
		if err != nil {
			return nil, &ParseError{Kind: KindInvalidFormat, Value: raw, Reason: "an invalid integer", Cause: err}
		}
		return v, nil
	case typeNumber:
		v, err := strconv.ParseFloat(raw, 64)
		if err != nil {
			return nil, &ParseError{Kind: KindInvalidFormat, Value: raw, Reason: "an invalid number", Cause: err}
		}
		return v, nil
	case typeBoolean:
		v, err := strconv.ParseBool(raw)
		if err != nil {
			return nil, &ParseError{Kind: KindInvalidFormat, Value: raw, Reason: "an invalid number", Cause: err}
		}
		return v, nil
	case typeString:
		return raw, nil
	default:
		panic(fmt.Sprintf("schema has non primitive type %q", schema.Value.Type))
	}
}
