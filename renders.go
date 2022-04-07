package soda

import "strings"

func (s *Soda) Swagger() string {
	const template = `
<!DOCTYPE html>
<html charset="UTF-8">
<head>
    <meta http-equiv="Content-Type" content="text/html;charset=utf-8">
    <title>Swagger UI</title>
    <link type="text/css" rel="stylesheet" href="https://cdn.jsdelivr.net/npm/swagger-ui-dist@3/swagger-ui.css">
    <script src="https://cdn.jsdelivr.net/npm/swagger-ui-dist@3/swagger-ui-bundle.js"></script>
</head>
</html>
<body>
  <div id="ui"></div>
  <script>
    let spec = {:spec};
    let oauth2RedirectUrl;

    let query = window.location.href.indexOf("?");
    if (query > 0) {
        oauth2RedirectUrl = window.location.href.substring(0, query);
    } else {
        oauth2RedirectUrl = window.location.href;
    }

    if (!oauth2RedirectUrl.endsWith("/")) {
        oauth2RedirectUrl += "/";
    }
    oauth2RedirectUrl += "oauth-receiver.html";
    SwaggerUIBundle({
        dom_id: '#ui',
        spec: spec,
        filter: false,
        oauth2RedirectUrl: oauth2RedirectUrl,
    })
  </script>
</body>
`
	return strings.Replace(template, "{:spec}", string(s.GetOpenAPIJSON()), 1)
}

func (s *Soda) Redoc() string {
	const template = `
<!DOCTYPE html>
<html>
  <head>
    <title>Redoc</title>
    <meta charset="utf-8"/>
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <style>
      body {
        margin: 0;
        padding: 0;
      }
    </style>
    <script src="https://cdn.jsdelivr.net/npm/redoc@next/bundles/redoc.standalone.js"></script>
  </head>
  <body>
    <div id="redoc-container"></div>
    <script>
        let spec = {:spec};
        Redoc.init(spec, {
          scrollYOffset: 50
        }, document.getElementById('redoc-container'));
    </script>
  </body>
</html>`
	return strings.Replace(template, "{:spec}", string(s.GetOpenAPIJSON()), 1)
}

func (s *Soda) RapiDoc() string {
	const template = `
<!DOCTYPE html>
<html charset="UTF-8">
  <head>
    <meta http-equiv="Content-Type" content="text/html;charset=utf-8">
    <meta name="viewport" content="width=device-width, minimum-scale=1, initial-scale=1, user-scalable=yes">
    <title>RapiDoc</title>
    <script type="module" src="https://cdn.jsdelivr.net/npm/rapidoc/dist/rapidoc-min.min.js"></script>
  </head>
  <body>
    <rapi-doc id="thedoc" theme="light" render-style="focused" show-header="false" show-components="true" allow-try="true"
      allow-authentication="true" regular-font="Open Sans" mono-font="Roboto Mono" font-size="large"
      schema-description-expanded="true">
    </rapi-doc>
    <script>
      document.addEventListener('DOMContentLoaded', (event) => {
        let docEl = document.getElementById("thedoc");
        docEl.loadSpec({:spec});
        docEl.setAttribute('theme','light');
        docEl.setAttribute('bg-color','#ecf1f7');
        docEl.setAttribute('text-color','#133863');
      })
    </script>
  </body>
</html>`
	return strings.Replace(template, "{:spec}", string(s.GetOpenAPIJSON()), 1)
}
