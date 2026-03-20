//go:build tools

package tools

import (
	// tfplugindocs generates docs/ from provider schema + templates/
	_ "github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs"
)
