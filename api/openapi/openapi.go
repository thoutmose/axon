// Package openapi embeds the API contract (openapi.yaml), which is the source
// of truth for every /v1 endpoint. Contract tests check real responses against it.
package openapi

import _ "embed"

//go:embed openapi.yaml
var Spec []byte
