// tools/go.mod
module github.com/latency-space/tools

go 1.21

require (
	github.com/cloudflare/cloudflare-go v0.79.0
	github.com/latency-space/shared v0.0.0
)

require (
	github.com/goccy/go-json v0.10.2 // indirect
	github.com/google/go-querystring v1.1.0 // indirect
	github.com/hashicorp/go-cleanhttp v0.5.2 // indirect
	github.com/hashicorp/go-retryablehttp v0.7.4 // indirect
	golang.org/x/net v0.17.0 // indirect
	golang.org/x/text v0.13.0 // indirect
	golang.org/x/time v0.3.0 // indirect
)

replace github.com/latency-space/shared => ../shared