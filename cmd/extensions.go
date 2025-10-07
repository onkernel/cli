package cmd

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/onkernel/cli/pkg/util"
	"github.com/onkernel/kernel-go-sdk"
	"github.com/onkernel/kernel-go-sdk/option"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

// ExtensionsService defines the subset of the Kernel SDK extension client that we use.
type ExtensionsService interface {
	List(ctx context.Context, opts ...option.RequestOption) (res *[]kernel.ExtensionListResponse, err error)
	Delete(ctx context.Context, idOrName string, opts ...option.RequestOption) (err error)
	Download(ctx context.Context, idOrName string, opts ...option.RequestOption) (res *http.Response, err error)
	DownloadFromChromeStore(ctx context.Context, query kernel.ExtensionDownloadFromChromeStoreParams, opts ...option.RequestOption) (res *http.Response, err error)
	Upload(ctx context.Context, body kernel.ExtensionUploadParams, opts ...option.RequestOption) (res *kernel.ExtensionUploadResponse, err error)
}

type ExtensionsListInput struct{}

type ExtensionsDeleteInput struct {
	Identifier  string
	SkipConfirm bool
}

type ExtensionsDownloadInput struct {
	Identifier string
	Output     string
}

type ExtensionsDownloadWebStoreInput struct {
	URL    string
	Output string
	OS     string
}

type ExtensionsUploadInput struct {
	Dir  string
	Name string
}

// ExtensionsCmd handles extension operations independent of cobra.
type ExtensionsCmd struct {
	extensions ExtensionsService
}

func (e ExtensionsCmd) List(ctx context.Context, _ ExtensionsListInput) error {
	pterm.Info.Println("Fetching extensions...")
	items, err := e.extensions.List(ctx)
	if err != nil {
		return util.CleanedUpSdkError{Err: err}
	}
	if items == nil || len(*items) == 0 {
		pterm.Info.Println("No extensions found")
		return nil
	}
	rows := pterm.TableData{{"Extension ID", "Name", "Created At", "Size (bytes)", "Last Used At"}}
	for _, it := range *items {
		name := it.Name
		if name == "" {
			name = "-"
		}
		rows = append(rows, []string{
			it.ID,
			name,
			util.FormatLocal(it.CreatedAt),
			fmt.Sprintf("%d", it.SizeBytes),
			util.FormatLocal(it.LastUsedAt),
		})
	}
	printTableNoPad(rows, true)
	return nil
}

func (e ExtensionsCmd) Delete(ctx context.Context, in ExtensionsDeleteInput) error {
	if in.Identifier == "" {
		pterm.Error.Println("Missing identifier")
		return nil
	}

	if !in.SkipConfirm {
		msg := fmt.Sprintf("Are you sure you want to delete extension '%s'?", in.Identifier)
		pterm.DefaultInteractiveConfirm.DefaultText = msg
		ok, _ := pterm.DefaultInteractiveConfirm.Show()
		if !ok {
			pterm.Info.Println("Deletion cancelled")
			return nil
		}
	}

	if err := e.extensions.Delete(ctx, in.Identifier); err != nil {
		if util.IsNotFound(err) {
			pterm.Info.Printf("Extension '%s' not found\n", in.Identifier)
			return nil
		}
		return util.CleanedUpSdkError{Err: err}
	}
	pterm.Success.Printf("Deleted extension: %s\n", in.Identifier)
	return nil
}

func (e ExtensionsCmd) Download(ctx context.Context, in ExtensionsDownloadInput) error {
	if in.Identifier == "" {
		pterm.Error.Println("Missing identifier")
		return nil
	}
	res, err := e.extensions.Download(ctx, in.Identifier)
	if err != nil {
		return util.CleanedUpSdkError{Err: err}
	}
	defer res.Body.Close()
	if in.Output == "" {
		pterm.Error.Println("Missing --to output directory")
		_, _ = io.Copy(io.Discard, res.Body)
		return nil
	}

	outDir, err := filepath.Abs(in.Output)
	if err != nil {
		pterm.Error.Printf("Failed to resolve output path: %v\n", err)
		_, _ = io.Copy(io.Discard, res.Body)
		return nil
	}
	// Create directory if not exists; if exists, ensure empty
	if st, err := os.Stat(outDir); err == nil {
		if !st.IsDir() {
			pterm.Error.Printf("Output path exists and is not a directory: %s\n", outDir)
			_, _ = io.Copy(io.Discard, res.Body)
			return nil
		}
		entries, _ := os.ReadDir(outDir)
		if len(entries) > 0 {
			pterm.Error.Printf("Output directory must be empty: %s\n", outDir)
			_, _ = io.Copy(io.Discard, res.Body)
			return nil
		}
	} else {
		if err := os.MkdirAll(outDir, 0o755); err != nil {
			pterm.Error.Printf("Failed to create output directory: %v\n", err)
			_, _ = io.Copy(io.Discard, res.Body)
			return nil
		}
	}

	// Write response to a temp zip, then extract
	tmpZip, err := os.CreateTemp("", "kernel-ext-*.zip")
	if err != nil {
		pterm.Error.Printf("Failed to create temp zip: %v\n", err)
		_, _ = io.Copy(io.Discard, res.Body)
		return nil
	}
	tmpName := tmpZip.Name()
	defer func() { _ = os.Remove(tmpName) }()
	if _, err := io.Copy(tmpZip, res.Body); err != nil {
		_ = tmpZip.Close()
		pterm.Error.Printf("Failed to read response: %v\n", err)
		return nil
	}
	_ = tmpZip.Close()
	if err := util.Unzip(tmpName, outDir); err != nil {
		pterm.Error.Printf("Failed to extract zip: %v\n", err)
		return nil
	}
	pterm.Success.Printf("Extracted extension to %s\n", outDir)
	return nil
}

func (e ExtensionsCmd) DownloadWebStore(ctx context.Context, in ExtensionsDownloadWebStoreInput) error {
	if in.URL == "" {
		pterm.Error.Println("Missing URL argument")
		return nil
	}
	params := kernel.ExtensionDownloadFromChromeStoreParams{URL: in.URL}
	switch in.OS {
	case "", string(kernel.ExtensionDownloadFromChromeStoreParamsOsLinux):
		// default linux
	case string(kernel.ExtensionDownloadFromChromeStoreParamsOsMac):
		params.Os = kernel.ExtensionDownloadFromChromeStoreParamsOsMac
	case string(kernel.ExtensionDownloadFromChromeStoreParamsOsWin):
		params.Os = kernel.ExtensionDownloadFromChromeStoreParamsOsWin
	default:
		pterm.Error.Println("--os must be one of mac, win, linux")
		return nil
	}

	res, err := e.extensions.DownloadFromChromeStore(ctx, params)
	if err != nil {
		return util.CleanedUpSdkError{Err: err}
	}
	defer res.Body.Close()

	if in.Output == "" {
		pterm.Error.Println("Missing --to output directory")
		_, _ = io.Copy(io.Discard, res.Body)
		return nil
	}

	outDir, err := filepath.Abs(in.Output)
	if err != nil {
		pterm.Error.Printf("Failed to resolve output path: %v\n", err)
		_, _ = io.Copy(io.Discard, res.Body)
		return nil
	}
	if st, err := os.Stat(outDir); err == nil {
		if !st.IsDir() {
			pterm.Error.Printf("Output path exists and is not a directory: %s\n", outDir)
			_, _ = io.Copy(io.Discard, res.Body)
			return nil
		}
		entries, _ := os.ReadDir(outDir)
		if len(entries) > 0 {
			pterm.Error.Printf("Output directory must be empty: %s\n", outDir)
			_, _ = io.Copy(io.Discard, res.Body)
			return nil
		}
	} else {
		if err := os.MkdirAll(outDir, 0o755); err != nil {
			pterm.Error.Printf("Failed to create output directory: %v\n", err)
			_, _ = io.Copy(io.Discard, res.Body)
			return nil
		}
	}

	// Save to temp zip then extract
	var bodyBuf bytes.Buffer
	if _, err := io.Copy(&bodyBuf, res.Body); err != nil {
		pterm.Error.Printf("Failed to read response: %v\n", err)
		return nil
	}
	tmpZip, err := os.CreateTemp("", "kernel-webstore-*.zip")
	if err != nil {
		pterm.Error.Printf("Failed to create temp zip: %v\n", err)
		return nil
	}
	tmpName := tmpZip.Name()
	if _, err := tmpZip.Write(bodyBuf.Bytes()); err != nil {
		_ = tmpZip.Close()
		pterm.Error.Printf("Failed to write temp zip: %v\n", err)
		return nil
	}
	_ = tmpZip.Close()
	defer os.Remove(tmpName)
	if err := util.Unzip(tmpName, outDir); err != nil {
		pterm.Error.Printf("Failed to extract zip: %v\n", err)
		return nil
	}
	pterm.Success.Printf("Extracted extension to %s\n", outDir)
	return nil
}

func (e ExtensionsCmd) Upload(ctx context.Context, in ExtensionsUploadInput) error {
	if in.Dir == "" {
		return fmt.Errorf("missing directory argument")
	}
	absDir, err := filepath.Abs(in.Dir)
	if err != nil {
		return fmt.Errorf("failed to resolve directory: %w", err)
	}
	stat, err := os.Stat(absDir)
	if err != nil || !stat.IsDir() {
		return fmt.Errorf("directory %s does not exist", absDir)
	}

	tmpFile := filepath.Join(os.TempDir(), fmt.Sprintf("kernel_ext_%d.zip", time.Now().UnixNano()))
	pterm.Info.Println("Zipping extension directory...")
	if err := util.ZipDirectory(absDir, tmpFile); err != nil {
		pterm.Error.Println("Failed to zip directory")
		return err
	}
	defer os.Remove(tmpFile)

	f, err := os.Open(tmpFile)
	if err != nil {
		return fmt.Errorf("failed to open temp zip: %w", err)
	}
	defer f.Close()

	params := kernel.ExtensionUploadParams{File: f}
	if in.Name != "" {
		params.Name = kernel.Opt(in.Name)
	}
	item, err := e.extensions.Upload(ctx, params)
	if err != nil {
		return util.CleanedUpSdkError{Err: err}
	}

	name := item.Name
	if name == "" {
		name = "-"
	}
	rows := pterm.TableData{{"Property", "Value"}}
	rows = append(rows, []string{"ID", item.ID})
	rows = append(rows, []string{"Name", name})
	rows = append(rows, []string{"Created At", util.FormatLocal(item.CreatedAt)})
	rows = append(rows, []string{"Size (bytes)", fmt.Sprintf("%d", item.SizeBytes)})
	printTableNoPad(rows, true)
	return nil
}

// --- Cobra wiring ---

var extensionsCmd = &cobra.Command{
	Use:   "extensions",
	Short: "Manage browser extensions",
	Long:  "Commands for managing Kernel browser extensions",
}

var extensionsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List extensions",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		client := getKernelClient(cmd)
		svc := client.Extensions
		e := ExtensionsCmd{extensions: &svc}
		return e.List(cmd.Context(), ExtensionsListInput{})
	},
}

var extensionsDeleteCmd = &cobra.Command{
	Use:   "delete <id-or-name>",
	Short: "Delete an extension by ID or name",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client := getKernelClient(cmd)
		skip, _ := cmd.Flags().GetBool("yes")
		svc := client.Extensions
		e := ExtensionsCmd{extensions: &svc}
		return e.Delete(cmd.Context(), ExtensionsDeleteInput{Identifier: args[0], SkipConfirm: skip})
	},
}

var extensionsDownloadCmd = &cobra.Command{
	Use:   "download <id-or-name>",
	Short: "Download an extension archive",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client := getKernelClient(cmd)
		out, _ := cmd.Flags().GetString("to")
		svc := client.Extensions
		e := ExtensionsCmd{extensions: &svc}
		return e.Download(cmd.Context(), ExtensionsDownloadInput{Identifier: args[0], Output: out})
	},
}

var extensionsDownloadWebStoreCmd = &cobra.Command{
	Use:   "download-web-store <url>",
	Short: "Download an extension from the Chrome Web Store",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client := getKernelClient(cmd)
		out, _ := cmd.Flags().GetString("to")
		osFlag, _ := cmd.Flags().GetString("os")
		svc := client.Extensions
		e := ExtensionsCmd{extensions: &svc}
		return e.DownloadWebStore(cmd.Context(), ExtensionsDownloadWebStoreInput{URL: args[0], Output: out, OS: osFlag})
	},
}

var extensionsUploadCmd = &cobra.Command{
	Use:   "upload <directory>",
	Short: "Upload an unpacked browser extension directory",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client := getKernelClient(cmd)
		name, _ := cmd.Flags().GetString("name")
		svc := client.Extensions
		e := ExtensionsCmd{extensions: &svc}
		return e.Upload(cmd.Context(), ExtensionsUploadInput{Dir: args[0], Name: name})
	},
}

func init() {
	extensionsCmd.AddCommand(extensionsListCmd)
	extensionsCmd.AddCommand(extensionsDeleteCmd)
	extensionsCmd.AddCommand(extensionsDownloadCmd)
	extensionsCmd.AddCommand(extensionsDownloadWebStoreCmd)
	extensionsCmd.AddCommand(extensionsUploadCmd)

	extensionsDeleteCmd.Flags().BoolP("yes", "y", false, "Skip confirmation prompt")
	extensionsDownloadCmd.Flags().String("to", "", "Output zip file path")
	extensionsDownloadWebStoreCmd.Flags().String("to", "", "Output zip file path for the downloaded archive")
	extensionsDownloadWebStoreCmd.Flags().String("os", "", "Target OS: mac, win, or linux (default linux)")
	extensionsUploadCmd.Flags().String("name", "", "Optional unique extension name")
}
