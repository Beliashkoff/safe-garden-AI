// Package prompts holds the Claude system prompts, embedded at build time so a
// single version travels from the RU repo into the worker request payload
// (ARCH §7.1, §11.3). The chat usecase (stage 2.3) puts SystemV1 into
// SendRequest.System.
package prompts

import _ "embed"

//go:embed system_v1.md
var systemV1 string

// SystemV1 returns the v1 agronomist system prompt.
func SystemV1() string { return systemV1 }
