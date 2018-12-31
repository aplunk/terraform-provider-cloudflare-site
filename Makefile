bin/terraform-provider-cloudflare-site: cmd/cloudflaresite/main.go
	go build -o bin/terraform-provider-cloudflare-site github.com/aplunk/terraform-provider-cloudflare-site/cmd/cloudflaresite