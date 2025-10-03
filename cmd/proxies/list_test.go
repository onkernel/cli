package proxies

import (
	"context"
	"errors"
	"testing"

	"github.com/onkernel/kernel-go-sdk"
	"github.com/onkernel/kernel-go-sdk/option"
	"github.com/stretchr/testify/assert"
)

func TestProxyList_Empty(t *testing.T) {
	buf := captureOutput(t)
	fake := &FakeProxyService{
		ListFunc: func(ctx context.Context, opts ...option.RequestOption) (*[]kernel.ProxyListResponse, error) {
			empty := []kernel.ProxyListResponse{}
			return &empty, nil
		},
	}

	p := ProxyCmd{proxies: fake}
	err := p.List(context.Background())

	assert.NoError(t, err)
	assert.Contains(t, buf.String(), "No proxy configurations found")
}

func TestProxyList_WithProxies(t *testing.T) {
	buf := captureOutput(t)

	proxies := []kernel.ProxyListResponse{
		createDatacenterProxy("dc-1", "US Datacenter", "US"),
		createResidentialProxy("res-1", "SF Residential", "US", "sanfrancisco", "CA"),
		createCustomProxy("custom-1", "My Proxy", "proxy.example.com", 8080),
		{
			ID:   "mobile-1",
			Name: "Mobile Proxy",
			Type: kernel.ProxyListResponseTypeMobile,
			Config: kernel.ProxyListResponseConfigUnion{
				Country: "US",
				Carrier: "verizon",
			},
		},
		{
			ID:   "isp-1",
			Name: "", // Test empty name
			Type: kernel.ProxyListResponseTypeIsp,
			Config: kernel.ProxyListResponseConfigUnion{
				Country: "EU",
			},
		},
	}

	fake := &FakeProxyService{
		ListFunc: func(ctx context.Context, opts ...option.RequestOption) (*[]kernel.ProxyListResponse, error) {
			return &proxies, nil
		},
	}

	p := ProxyCmd{proxies: fake}
	err := p.List(context.Background())

	assert.NoError(t, err)
	output := buf.String()

	// Check table headers
	assert.Contains(t, output, "ID")
	assert.Contains(t, output, "Name")
	assert.Contains(t, output, "Type")
	assert.Contains(t, output, "Config")

	// Check proxy data - use IDs and short strings that won't be truncated
	assert.Contains(t, output, "dc-1")
	assert.Contains(t, output, "US") // Part of "US Datacenter", may be truncated
	assert.Contains(t, output, "Country")

	assert.Contains(t, output, "res-1")
	assert.Contains(t, output, "SF") // Part of "SF Residential", may be truncated

	assert.Contains(t, output, "custom-1")
	assert.Contains(t, output, "My Proxy")
	assert.Contains(t, output, "custom")
	assert.Contains(t, output, "proxy.example.co") // May be truncated with "..."

	assert.Contains(t, output, "mobile-1")
	assert.Contains(t, output, "Mobile") // May be truncated with "..."
	assert.Contains(t, output, "mobile")

	assert.Contains(t, output, "isp-1")
	assert.Contains(t, output, "-") // Empty name shows as "-"
	assert.Contains(t, output, "isp")
	assert.Contains(t, output, "EU")
}

func TestProxyList_Error(t *testing.T) {
	_ = captureOutput(t)

	fake := &FakeProxyService{
		ListFunc: func(ctx context.Context, opts ...option.RequestOption) (*[]kernel.ProxyListResponse, error) {
			return nil, errors.New("API error")
		},
	}

	p := ProxyCmd{proxies: fake}
	err := p.List(context.Background())

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "API error")
}
