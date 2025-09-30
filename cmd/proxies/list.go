package proxies

import (
	"context"
	"fmt"
	"strings"

	"github.com/onkernel/cli/pkg/util"
	"github.com/onkernel/kernel-go-sdk"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

func (p ProxyCmd) List(ctx context.Context) error {
	pterm.Info.Println("Fetching proxy configurations...")

	items, err := p.proxies.List(ctx)
	if err != nil {
		return util.CleanedUpSdkError{Err: err}
	}

	if items == nil || len(*items) == 0 {
		pterm.Info.Println("No proxy configurations found")
		return nil
	}

	// Prepare table data
	tableData := pterm.TableData{
		{"ID", "Name", "Type", "Config"},
	}

	for _, proxy := range *items {
		name := proxy.Name
		if name == "" {
			name = "-"
		}

		// Format config based on type
		configStr := formatProxyConfig(&proxy)

		tableData = append(tableData, []string{
			proxy.ID,
			name,
			string(proxy.Type),
			configStr,
		})
	}

	PrintTableNoPad(tableData, true)
	return nil
}

func formatProxyConfig(proxy *kernel.ProxyListResponse) string {
	config := &proxy.Config
	switch proxy.Type {
	case kernel.ProxyListResponseTypeDatacenter, kernel.ProxyListResponseTypeIsp:
		if config.Country != "" {
			return fmt.Sprintf("Country: %s", config.Country)
		}
	case kernel.ProxyListResponseTypeResidential:
		parts := []string{}
		if config.Country != "" {
			parts = append(parts, fmt.Sprintf("Country: %s", config.Country))
		}
		if config.City != "" {
			parts = append(parts, fmt.Sprintf("City: %s", config.City))
		}
		if config.State != "" {
			parts = append(parts, fmt.Sprintf("State: %s", config.State))
		}
		if len(parts) > 0 {
			return strings.Join(parts, ", ")
		}
	case kernel.ProxyListResponseTypeMobile:
		parts := []string{}
		if config.Country != "" {
			parts = append(parts, fmt.Sprintf("Country: %s", config.Country))
		}
		if config.Carrier != "" {
			parts = append(parts, fmt.Sprintf("Carrier: %s", config.Carrier))
		}
		if len(parts) > 0 {
			return strings.Join(parts, ", ")
		}
	case kernel.ProxyListResponseTypeCustom:
		if config.Host != "" {
			auth := ""
			if config.Username != "" {
				auth = fmt.Sprintf(", Auth: %s", config.Username)
			}
			return fmt.Sprintf("%s:%d%s", config.Host, config.Port, auth)
		}
	}
	return "-"
}

func runProxiesList(cmd *cobra.Command, args []string) error {
	client := util.GetKernelClient(cmd)
	svc := client.Proxies
	p := ProxyCmd{proxies: &svc}
	return p.List(cmd.Context())
}
