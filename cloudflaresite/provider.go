package cloudflaresite

import "github.com/hashicorp/terraform/helper/schema"

// Provider returns the cloudflare_site provider
func Provider() *schema.Provider {
	return &schema.Provider{
		ResourcesMap: map[string]*schema.Resource{
			"cloudflare_site": resourceCloudflareSite(),
		},
	}
}
