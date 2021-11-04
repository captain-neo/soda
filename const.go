package soda

const (
	OpenAPITag = "oai"
)

const (
	KeyParameter   = "__parameter__"
	KeyRequestBody = "__request_body__"
)

// parameter props.
const (
	propIn      = "in"
	propExplode = "explode"
	propStyle   = "style"
)

// schema props.
const (
	propName = "name"
	// generic properties.
	propTitle           = "title"
	propDescription     = "description"
	propType            = "type"
	propDeprecated      = "deprecated"
	propAllowEmptyValue = "allowEmptyValue"
	propNullable        = "nullable"
	propReadOnly        = "readOnly"
	propWriteOnly       = "writeOnly"
	propEnum            = "enum"
	propDefault         = "default"
	propExample         = "example"
	propRequired        = "required"
	// string specified properties.
	propMinLength = "minLength"
	propMaxLength = "maxLength"
	propPattern   = "pattern"
	propFormat    = "format"
	// number specified properties.
	propMultipleOf       = "multipleOf"
	propMinimum          = "minimum"
	propMaximum          = "maximum"
	propExclusiveMaximum = "exclusiveMaximum"
	propExclusiveMinimum = "exclusiveMinimum"
	// array specified properties.
	propMinItems    = "minItems"
	propMaxItems    = "maxItems"
	propUniqueItems = "uniqueItems"
)

const (
	typeBoolean = "boolean"
	typeNumber  = "number"
	typeString  = "string"
	typeInteger = "integer"
	typeArray   = "array"
	typeObject  = "object"
)

const redocHTML = `
<!DOCTYPE html>
<html>
<head>
   <title>OpenAPI Doc</title>
   <!-- needed for adaptive design -->
   <meta charset="utf-8"/>
   <meta rawName="viewport" content="width=device-width, initial-scale=1">
   <link href="https://fonts.googleapis.com/css?family=Montserrat:300,400,700|Roboto:300,400,700" rel="stylesheet">
   <!--
   ReDoc doesn't change outer page styles
   -->
   <style>
       body {
           margin: 0;
           padding: 0;
       }
   </style>
</head>
<body>
<redoc spec-url="/openapi.json"></redoc>
<script src="https://cdn.jsdelivr.net/npm/redoc@next/bundles/redoc.standalone.js"></script>
</body>
</html>
`

const swaggerHTML = `
<!DOCTYPE html>
<html>
<head>
   <link type="text/css" rel="stylesheet" href="https://cdn.jsdelivr.net/npm/swagger-ui-dist@3/swagger-ui.css">
   <title>OpenAPI Doc Swagger</title>
</head>
<body>
<div id="swagger-ui">
</div>
<script src="https://cdn.jsdelivr.net/npm/swagger-ui-dist@3/swagger-ui-bundle.js"></script>
<script>
   const ui = SwaggerUIBundle({
       url: '/openapi.json',
       dom_id: '#swagger-ui',
       presets: [
           SwaggerUIBundle.presets.apis,
           SwaggerUIBundle.SwaggerUIStandalonePreset
       ],
       layout: "BaseLayout",
       deepLinking: true,
       showExtensions: true,
       showCommonExtensions: true
   });
</script>
</body>
</html>
`
