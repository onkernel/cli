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
	switch proxy.Type {
	case kernel.ProxyListResponseTypeDatacenter:
		dc := proxy.Config.AsProxyListResponseConfigDatacenterProxyConfig()
		if dc.Country != "" {
			return fmt.Sprintf("Country: %s", dc.Country)
		}
	case kernel.ProxyListResponseTypeIsp:
		isp := proxy.Config.AsProxyListResponseConfigIspProxyConfig()
		if isp.Country != "" {
			return fmt.Sprintf("Country: %s", isp.Country)
		}
	case kernel.ProxyListResponseTypeResidential:
		res := proxy.Config.AsProxyListResponseConfigResidentialProxyConfig()
		if res.Country != "" || res.City != "" || res.State != "" {
			parts := []string{}
			if res.Country != "" {
				parts = append(parts, fmt.Sprintf("Country: %s", res.Country))
			}
			if res.City != "" {
				parts = append(parts, fmt.Sprintf("City: %s", res.City))
			}
			if res.State != "" {
				parts = append(parts, fmt.Sprintf("State: %s", res.State))
			}
			return strings.Join(parts, ", ")
		}
	case kernel.ProxyListResponseTypeMobile:
		mob := proxy.Config.AsProxyListResponseConfigMobileProxyConfig()
		if mob.Country != "" || mob.Carrier != "" {
			parts := []string{}
			if mob.Country != "" {
				parts = append(parts, fmt.Sprintf("Country: %s", mob.Country))
			}
			if mob.Carrier != "" {
				parts = append(parts, fmt.Sprintf("Carrier: %s", mob.Carrier))
			}
			return strings.Join(parts, ", ")
		}
	case kernel.ProxyListResponseTypeCustom:
		custom := proxy.Config.AsProxyListResponseConfigCustomProxyConfig()
		if custom.Host != "" {
			auth := ""
			if custom.Username != "" {
				auth = fmt.Sprintf(", Auth: %s", custom.Username)
			}
			return fmt.Sprintf("%s:%d%s", custom.Host, custom.Port, auth)
		}
	}
	return "-"
}

func runProxiesList(cmd *cobra.Command, args []string) error {
	client := GetKernelClient(cmd)
	svc := client.Proxies
	p := ProxyCmd{proxies: &svc}
	return p.List(cmd.Context())
}
