package soda

import (
	"reflect"
	"strings"
	"sync"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/utils"
	"github.com/gorilla/schema"
)

var decoderPool = &sync.Pool{New: func() interface{} {
	decoder := schema.NewDecoder()
	decoder.IgnoreUnknownKeys(true)
	return decoder
}}

type parserFunc func(*fiber.Ctx, interface{}) error

var parameterParsers = map[string]parserFunc{
	"query":  queryParser,
	"header": headerParser,
	"path":   pathParser,
	"cookie": cookieParser,
}

func queryParser(c *fiber.Ctx, out interface{}) error {
	return c.QueryParser(out)
}

func headerParser(c *fiber.Ctx, out interface{}) error {
	data := make(map[string][]string)
	c.Request().Header.VisitAll(func(key, val []byte) {
		k := utils.UnsafeString(key)
		v := utils.UnsafeString(val)

		if strings.Contains(v, ",") && equalSliceField("header", out, k) {
			values := strings.Split(v, ",")
			for i := 0; i < len(values); i++ {
				data[k] = append(data[k], values[i])
			}
		} else {
			data[k] = append(data[k], v)
		}
	})

	return parseToStruct("header", out, data)
}

func pathParser(c *fiber.Ctx, out interface{}) error {
	data := make(map[string][]string)
	for _, k := range c.Route().Params {
		data[k] = []string{c.Params(k)}
	}
	return parseToStruct("path", out, data)
}

func cookieParser(c *fiber.Ctx, out interface{}) error {
	data := make(map[string][]string)
	c.Request().Header.VisitAllCookie(func(key, val []byte) {
		k := utils.UnsafeString(key)
		v := utils.UnsafeString(val)
		if strings.Contains(v, ",") && equalSliceField("cookie", out, k) {
			values := strings.Split(v, ",")
			for i := 0; i < len(values); i++ {
				data[k] = append(data[k], values[i])
			}
		} else {
			data[k] = append(data[k], v)
		}
	})
	return parseToStruct("cookie", out, data)
}

func parseToStruct(aliasTag string, out interface{}, data map[string][]string) error {
	// Get decoder from pool
	schemaDecoder := decoderPool.Get().(*schema.Decoder)
	defer decoderPool.Put(schemaDecoder)

	// Set alias tag
	schemaDecoder.SetAliasTag(aliasTag)
	return schemaDecoder.Decode(out, data)
}

func equalSliceField(tagName string, out interface{}, key string) bool {
	// Get type of interface
	outTyp := reflect.TypeOf(out).Elem()
	key = utils.ToLower(key)
	// Must be a struct to match a field
	if outTyp.Kind() != reflect.Struct {
		return false
	}
	// Copy interface to an value to be used
	outVal := reflect.ValueOf(out).Elem()
	// Loop over each field
	for i := 0; i < outTyp.NumField(); i++ {
		// Get field value data
		structField := outVal.Field(i)
		// Can this field be changed?
		if !structField.CanSet() {
			continue
		}
		// Does the field type equals input?
		if structField.Kind() != reflect.Slice {
			continue
		}
		// Get field key data
		typeField := outTyp.Field(i)
		// Get tag from field if exist
		inputFieldName := typeField.Tag.Get(tagName)
		if inputFieldName == "" {
			inputFieldName = typeField.Name
		} else {
			inputFieldName = strings.Split(inputFieldName, ",")[0]
		}
		// Compare field/tag with provided key
		if utils.ToLower(inputFieldName) == key {
			return true
		}
	}
	return false
}
