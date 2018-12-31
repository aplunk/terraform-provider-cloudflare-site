package main

import (
	"github.com/aplunk/terraform-provider-cloudflare-site/cloudflaresite"
	"github.com/hashicorp/terraform/plugin"
	"github.com/hashicorp/terraform/terraform"
)

func main() {
	plugin.Serve(&plugin.ServeOpts{
		ProviderFunc: func() terraform.ResourceProvider {
			return cloudflaresite.Provider()
		},
	})
}
