package proxies

import (
	"context"
	"fmt"

	"github.com/onkernel/cli/pkg/util"
	"github.com/onkernel/kernel-go-sdk"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

func (p ProxyCmd) Get(ctx context.Context, in ProxyGetInput) error {
	item, err := p.proxies.Get(ctx, in.ID)
	if err != nil {
		return util.CleanedUpSdkError{Err: err}
	}

	// Display proxy details
	rows := pterm.TableData{{"Property", "Value"}}

	rows = append(rows, []string{"ID", item.ID})

	name := item.Name
	if name == "" {
		name = "-"
	}
	rows = append(rows, []string{"Name", name})
	rows = append(rows, []string{"Type", string(item.Type)})

	// Display type-specific config details
	rows = append(rows, getProxyConfigRows(item)...)

	PrintTableNoPad(rows, true)
	return nil
}

func getProxyConfigRows(proxy *kernel.ProxyGetResponse) [][]string {
	var rows [][]string

	switch proxy.Type {
	case kernel.ProxyGetResponseTypeDatacenter:
		dc := proxy.Config.AsProxyGetResponseConfigDatacenterProxyConfig()
		if dc.Country != "" {
			rows = append(rows, []string{"Country", dc.Country})
		}
	case kernel.ProxyGetResponseTypeIsp:
		isp := proxy.Config.AsProxyGetResponseConfigIspProxyConfig()
		if isp.Country != "" {
			rows = append(rows, []string{"Country", isp.Country})
		}
	case kernel.ProxyGetResponseTypeResidential:
		res := proxy.Config.AsProxyGetResponseConfigResidentialProxyConfig()
		if res.Country != "" || res.City != "" || res.State != "" || res.Zip != "" || res.Asn != "" || res.Os != "" {
			if res.Country != "" {
				rows = append(rows, []string{"Country", res.Country})
			}
			if res.City != "" {
				rows = append(rows, []string{"City", res.City})
			}
			if res.State != "" {
				rows = append(rows, []string{"State", res.State})
			}
			if res.Zip != "" {
				rows = append(rows, []string{"ZIP", res.Zip})
			}
			if res.Asn != "" {
				rows = append(rows, []string{"ASN", res.Asn})
			}
			if res.Os != "" {
				rows = append(rows, []string{"OS", res.Os})
			}
		}
	case kernel.ProxyGetResponseTypeMobile:
		mob := proxy.Config.AsProxyGetResponseConfigMobileProxyConfig()
		if mob.Country != "" || mob.City != "" || mob.State != "" || mob.Zip != "" || mob.Asn != "" || mob.Carrier != "" {
			if mob.Country != "" {
				rows = append(rows, []string{"Country", mob.Country})
			}
			if mob.City != "" {
				rows = append(rows, []string{"City", mob.City})
			}
			if mob.State != "" {
				rows = append(rows, []string{"State", mob.State})
			}
			if mob.Zip != "" {
				rows = append(rows, []string{"ZIP", mob.Zip})
			}
			if mob.Asn != "" {
				rows = append(rows, []string{"ASN", mob.Asn})
			}
			if mob.Carrier != "" {
				rows = append(rows, []string{"Carrier", mob.Carrier})
			}
		}
	case kernel.ProxyGetResponseTypeCustom:
		custom := proxy.Config.AsProxyGetResponseConfigCustomProxyConfig()
		if custom.Host != "" {
			rows = append(rows, []string{"Host", custom.Host})
			rows = append(rows, []string{"Port", fmt.Sprintf("%d", custom.Port)})
			if custom.Username != "" {
				rows = append(rows, []string{"Username", custom.Username})
			}
			hasPassword := "No"
			if custom.HasPassword {
				hasPassword = "Yes"
			}
			rows = append(rows, []string{"Has Password", hasPassword})
		}
	}

	return rows
}

func runProxiesGet(cmd *cobra.Command, args []string) error {
	client := GetKernelClient(cmd)
	svc := client.Proxies
	p := ProxyCmd{proxies: &svc}
	return p.Get(cmd.Context(), ProxyGetInput{ID: args[0]})
}
