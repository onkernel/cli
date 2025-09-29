package proxies

import (
	"context"
	"fmt"

	"github.com/onkernel/cli/pkg/util"
	"github.com/onkernel/kernel-go-sdk"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

func (p ProxyCmd) Create(ctx context.Context, in ProxyCreateInput) error {
	// Validate proxy type
	var proxyType kernel.ProxyNewParamsType
	switch in.Type {
	case "datacenter":
		proxyType = kernel.ProxyNewParamsTypeDatacenter
	case "isp":
		proxyType = kernel.ProxyNewParamsTypeIsp
	case "residential":
		proxyType = kernel.ProxyNewParamsTypeResidential
	case "mobile":
		proxyType = kernel.ProxyNewParamsTypeMobile
	case "custom":
		proxyType = kernel.ProxyNewParamsTypeCustom
	default:
		return fmt.Errorf("invalid proxy type: %s", in.Type)
	}

	params := kernel.ProxyNewParams{
		Type: proxyType,
	}

	if in.Name != "" {
		params.Name = kernel.Opt(in.Name)
	}

	// Build config based on type
	switch proxyType {
	case kernel.ProxyNewParamsTypeDatacenter:
		if in.Country == "" {
			return fmt.Errorf("--country is required for datacenter proxy type")
		}
		params.Config = kernel.ProxyNewParamsConfigUnion{
			OfProxyNewsConfigDatacenterProxyConfig: &kernel.ProxyNewParamsConfigDatacenterProxyConfig{
				Country: in.Country,
			},
		}

	case kernel.ProxyNewParamsTypeIsp:
		if in.Country == "" {
			return fmt.Errorf("--country is required for ISP proxy type")
		}
		params.Config = kernel.ProxyNewParamsConfigUnion{
			OfProxyNewsConfigIspProxyConfig: &kernel.ProxyNewParamsConfigIspProxyConfig{
				Country: in.Country,
			},
		}

	case kernel.ProxyNewParamsTypeResidential:
		config := kernel.ProxyNewParamsConfigResidentialProxyConfig{}

		// Validate that if city is provided, country must also be provided
		if in.City != "" && in.Country == "" {
			return fmt.Errorf("--country is required when --city is specified")
		}

		if in.Country != "" {
			config.Country = kernel.Opt(in.Country)
		}
		if in.City != "" {
			config.City = kernel.Opt(in.City)
		}
		if in.State != "" {
			config.State = kernel.Opt(in.State)
		}
		if in.Zip != "" {
			config.Zip = kernel.Opt(in.Zip)
		}
		if in.ASN != "" {
			config.Asn = kernel.Opt(in.ASN)
		}
		if in.OS != "" {
			// Validate OS value
			switch in.OS {
			case "windows", "macos", "android":
				config.Os = in.OS
			default:
				return fmt.Errorf("invalid OS value: %s (must be windows, macos, or android)", in.OS)
			}
		}
		params.Config = kernel.ProxyNewParamsConfigUnion{
			OfProxyNewsConfigResidentialProxyConfig: &config,
		}

	case kernel.ProxyNewParamsTypeMobile:
		config := kernel.ProxyNewParamsConfigMobileProxyConfig{}

		// Validate that if city is provided, country must also be provided
		if in.City != "" && in.Country == "" {
			return fmt.Errorf("--country is required when --city is specified")
		}

		if in.Country != "" {
			config.Country = kernel.Opt(in.Country)
		}
		if in.City != "" {
			config.City = kernel.Opt(in.City)
		}
		if in.State != "" {
			config.State = kernel.Opt(in.State)
		}
		if in.Zip != "" {
			config.Zip = kernel.Opt(in.Zip)
		}
		if in.ASN != "" {
			config.Asn = kernel.Opt(in.ASN)
		}
		if in.Carrier != "" {
			// The API will validate the carrier value
			config.Carrier = in.Carrier
		}
		params.Config = kernel.ProxyNewParamsConfigUnion{
			OfProxyNewsConfigMobileProxyConfig: &config,
		}

	case kernel.ProxyNewParamsTypeCustom:
		if in.Host == "" {
			return fmt.Errorf("--host is required for custom proxy type")
		}
		if in.Port == 0 {
			return fmt.Errorf("--port is required for custom proxy type")
		}

		config := kernel.ProxyNewParamsConfigCreateCustomProxyConfig{
			Host: in.Host,
			Port: int64(in.Port),
		}
		if in.Username != "" {
			config.Username = kernel.Opt(in.Username)
		}
		if in.Password != "" {
			config.Password = kernel.Opt(in.Password)
		}
		params.Config = kernel.ProxyNewParamsConfigUnion{
			OfProxyNewsConfigCreateCustomProxyConfig: &config,
		}
	}

	pterm.Info.Printf("Creating %s proxy...\n", proxyType)

	proxy, err := p.proxies.New(ctx, params)
	if err != nil {
		return util.CleanedUpSdkError{Err: err}
	}

	pterm.Success.Printf("Successfully created proxy\n")

	// Display created proxy details
	rows := pterm.TableData{{"Property", "Value"}}
	rows = append(rows, []string{"ID", proxy.ID})

	name := proxy.Name
	if name == "" {
		name = "-"
	}
	rows = append(rows, []string{"Name", name})
	rows = append(rows, []string{"Type", string(proxy.Type)})

	PrintTableNoPad(rows, true)
	return nil
}

func runProxiesCreate(command *cobra.Command, args []string) error {
	client := command.Context().Value("kernel_client").(kernel.Client)

	// Get all flag values
	proxyType, _ := command.Flags().GetString("type")
	name, _ := command.Flags().GetString("name")
	country, _ := command.Flags().GetString("country")
	city, _ := command.Flags().GetString("city")
	state, _ := command.Flags().GetString("state")
	zip, _ := command.Flags().GetString("zip")
	asn, _ := command.Flags().GetString("asn")
	os, _ := command.Flags().GetString("os")
	carrier, _ := command.Flags().GetString("carrier")
	host, _ := command.Flags().GetString("host")
	port, _ := command.Flags().GetInt("port")
	username, _ := command.Flags().GetString("username")
	password, _ := command.Flags().GetString("password")

	svc := client.Proxies
	p := ProxyCmd{proxies: &svc}
	return p.Create(command.Context(), ProxyCreateInput{
		Name:     name,
		Type:     proxyType,
		Country:  country,
		City:     city,
		State:    state,
		Zip:      zip,
		ASN:      asn,
		OS:       os,
		Carrier:  carrier,
		Host:     host,
		Port:     port,
		Username: username,
		Password: password,
	})
}
