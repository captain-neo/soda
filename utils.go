package soda

import (
	"strconv"
	"strings"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
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
		if v, e := toIntE(s); e == nil {
			result = append(result, v)
		}
	}
	return result
}

func parseFloatSlice(val string) interface{} {
	ss := strings.Split(val, " ")
	result := make([]interface{}, 0, len(ss))
	for _, s := range ss {
		if v, e := toFloatE(s); e == nil {
			result = append(result, v)
		}
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

func toIntE(v string) (int, error) {
	return strconv.Atoi(v)
}

func toInt(v string) int {
	i, _ := toIntE(v)
	return i
}

func toFloatE(v string) (float64, error) {
	return strconv.ParseFloat(v, 64)
}

func toFloat(v string) float64 {
	f, _ := toFloatE(v)
	return f
}

// var (
// 	matchFirstCap = regexp.MustCompile("(.)([A-Z][a-z]+)")
// 	matchAllCap   = regexp.MustCompile("([a-z0-9])([A-Z])")
// )

func toCamelCase(str string) string {
	kebab := strings.ReplaceAll(str, "-", " ")
	return strings.ReplaceAll(cases.Title(language.English).String(kebab), " ", "")
}

func summary2operationID(summary string) string {
	return strings.ReplaceAll(strings.ReplaceAll(summary, " ", "-"), "/", "-")
}
