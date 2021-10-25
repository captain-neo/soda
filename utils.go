package soda

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2/utils"
)

func parseStringSlice(val string) interface{} {
	ss := strings.Split(val, " ")
	result := make([]interface{}, 0, len(ss))
	for _, s := range ss {
		result = append(result, s)
	}
	return result
}

func parseIntSlice(val string) interface{} {
	ss := strings.Split(val, " ")
	result := make([]interface{}, 0, len(ss))
	for _, s := range ss {
		result = append(result, toInt(s))
	}
	return result
}

func parseFloatSlice(val string) interface{} {
	ss := strings.Split(val, " ")
	result := make([]interface{}, 0, len(ss))
	for _, s := range ss {
		result = append(result, toFloat(s))
	}
	return result
}

func toBool(v string) bool {
	if v == "" {
		return true
	}
	b, _ := strconv.ParseBool(v)
	return b
}

func toUint(v string) uint64 {
	u, _ := strconv.ParseUint(v, 10, 64)
	return u
}

func toInt(v string) int {
	i, _ := strconv.Atoi(v)
	return i
}

func toFloat(v string) float64 {
	f, _ := strconv.ParseFloat(v, 64)
	return f
}

var (
	matchFirstCap = regexp.MustCompile("(.)([A-Z][a-z]+)")
	matchAllCap   = regexp.MustCompile("([a-z0-9])([A-Z])")
)

func toKebabCase(str string) string {
	kebab := matchFirstCap.ReplaceAllString(str, "${1}-${2}")
	kebab = matchAllCap.ReplaceAllString(kebab, "${1}-${2}")
	return strings.ToLower(kebab)
}

func toCamelCase(str string) string {
	kebab := strings.ReplaceAll(str, "-", " ")
	return strings.ReplaceAll(strings.Title(kebab), " ", "")
}

func getHandlerName(fn interface{}) string {
	parts := strings.Split(utils.FunctionName(fn), "/")
	if len(parts) == 0 {
		panic("cannot get fn name")
	}
	return strings.ReplaceAll(parts[len(parts)-1], ".", "")
}
