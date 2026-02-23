package swagger

import "embed"

//go:embed index.html swagger.json
var SwaggerUI embed.FS
