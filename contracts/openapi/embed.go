package openapi

import _ "embed"

//go:embed openapi.yml
var Spec []byte
