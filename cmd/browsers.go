package cmd

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/onkernel/cli/pkg/util"
	"github.com/onkernel/kernel-go-sdk"
	"github.com/onkernel/kernel-go-sdk/option"
	"github.com/onkernel/kernel-go-sdk/packages/ssestream"
	"github.com/onkernel/kernel-go-sdk/shared"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

// BrowsersService defines the subset of the Kernel SDK browser client that we use.
// See https://github.com/onkernel/kernel-go-sdk/blob/main/browser.go
type BrowsersService interface {
	List(ctx context.Context, opts ...option.RequestOption) (res *[]kernel.BrowserListResponse, err error)
	New(ctx context.Context, body kernel.BrowserNewParams, opts ...option.RequestOption) (res *kernel.BrowserNewResponse, err error)
	Delete(ctx context.Context, body kernel.BrowserDeleteParams, opts ...option.RequestOption) (err error)
	DeleteByID(ctx context.Context, id string, opts ...option.RequestOption) (err error)
}

// BrowserReplaysService defines the subset we use for browser replays.
type BrowserReplaysService interface {
	List(ctx context.Context, id string, opts ...option.RequestOption) (res *[]kernel.BrowserReplayListResponse, err error)
	Download(ctx context.Context, replayID string, query kernel.BrowserReplayDownloadParams, opts ...option.RequestOption) (res *http.Response, err error)
	Start(ctx context.Context, id string, body kernel.BrowserReplayStartParams, opts ...option.RequestOption) (res *kernel.BrowserReplayStartResponse, err error)
	Stop(ctx context.Context, replayID string, body kernel.BrowserReplayStopParams, opts ...option.RequestOption) (err error)
}

// BrowserFSService defines the subset we use for browser filesystem APIs.
type BrowserFSService interface {
	NewDirectory(ctx context.Context, id string, body kernel.BrowserFNewDirectoryParams, opts ...option.RequestOption) (err error)
	DeleteDirectory(ctx context.Context, id string, body kernel.BrowserFDeleteDirectoryParams, opts ...option.RequestOption) (err error)
	DeleteFile(ctx context.Context, id string, body kernel.BrowserFDeleteFileParams, opts ...option.RequestOption) (err error)
	DownloadDirZip(ctx context.Context, id string, query kernel.BrowserFDownloadDirZipParams, opts ...option.RequestOption) (res *http.Response, err error)
	FileInfo(ctx context.Context, id string, query kernel.BrowserFFileInfoParams, opts ...option.RequestOption) (res *kernel.BrowserFFileInfoResponse, err error)
	ListFiles(ctx context.Context, id string, query kernel.BrowserFListFilesParams, opts ...option.RequestOption) (res *[]kernel.BrowserFListFilesResponse, err error)
	Move(ctx context.Context, id string, body kernel.BrowserFMoveParams, opts ...option.RequestOption) (err error)
	ReadFile(ctx context.Context, id string, query kernel.BrowserFReadFileParams, opts ...option.RequestOption) (res *http.Response, err error)
	SetFilePermissions(ctx context.Context, id string, body kernel.BrowserFSetFilePermissionsParams, opts ...option.RequestOption) (err error)
	Upload(ctx context.Context, id string, body kernel.BrowserFUploadParams, opts ...option.RequestOption) (err error)
	UploadZip(ctx context.Context, id string, body kernel.BrowserFUploadZipParams, opts ...option.RequestOption) (err error)
	WriteFile(ctx context.Context, id string, contents io.Reader, body kernel.BrowserFWriteFileParams, opts ...option.RequestOption) (err error)
}

// BrowserProcessService defines the subset we use for browser process APIs.
type BrowserProcessService interface {
	Exec(ctx context.Context, id string, body kernel.BrowserProcessExecParams, opts ...option.RequestOption) (res *kernel.BrowserProcessExecResponse, err error)
	Kill(ctx context.Context, processID string, params kernel.BrowserProcessKillParams, opts ...option.RequestOption) (res *kernel.BrowserProcessKillResponse, err error)
	Spawn(ctx context.Context, id string, body kernel.BrowserProcessSpawnParams, opts ...option.RequestOption) (res *kernel.BrowserProcessSpawnResponse, err error)
	Status(ctx context.Context, processID string, query kernel.BrowserProcessStatusParams, opts ...option.RequestOption) (res *kernel.BrowserProcessStatusResponse, err error)
	Stdin(ctx context.Context, processID string, params kernel.BrowserProcessStdinParams, opts ...option.RequestOption) (res *kernel.BrowserProcessStdinResponse, err error)
	StdoutStreamStreaming(ctx context.Context, processID string, query kernel.BrowserProcessStdoutStreamParams, opts ...option.RequestOption) (stream *ssestream.Stream[kernel.BrowserProcessStdoutStreamResponse])
}

// BrowserLogService defines the subset we use for browser log APIs.
type BrowserLogService interface {
	StreamStreaming(ctx context.Context, id string, query kernel.BrowserLogStreamParams, opts ...option.RequestOption) (stream *ssestream.Stream[shared.LogEvent])
}

// BoolFlag captures whether a boolean flag was set explicitly and its value.
type BoolFlag struct {
	Set   bool
	Value bool
}

// Inputs for each command
type BrowsersCreateInput struct {
	PersistenceID      string
	TimeoutSeconds     int
	Stealth            BoolFlag
	Headless           BoolFlag
	ProfileID          string
	ProfileName        string
	ProfileSaveChanges BoolFlag
	ProxyID            string
}

type BrowsersDeleteInput struct {
	Identifier  string
	SkipConfirm bool
}

type BrowsersViewInput struct {
	Identifier string
}

// BrowsersCmd is a cobra-independent command handler for browsers operations.
type BrowsersCmd struct {
	browsers BrowsersService
	replays  BrowserReplaysService
	fs       BrowserFSService
	process  BrowserProcessService
	logs     BrowserLogService
}

func (b BrowsersCmd) List(ctx context.Context) error {
	pterm.Info.Println("Fetching browsers...")

	browsers, err := b.browsers.List(ctx)
	if err != nil {
		return util.CleanedUpSdkError{Err: err}
	}

	if browsers == nil || len(*browsers) == 0 {
		pterm.Info.Println("No running or persistent browsers found")
		return nil
	}

	// Prepare table data
	tableData := pterm.TableData{
		{"Browser ID", "Created At", "Persistent ID", "Profile", "CDP WS URL", "Live View URL"},
	}

	for _, browser := range *browsers {
		persistentID := "-"
		if browser.Persistence.ID != "" {
			persistentID = browser.Persistence.ID
		}

		profile := "-"
		if browser.Profile.Name != "" {
			profile = browser.Profile.Name
		} else if browser.Profile.ID != "" {
			profile = browser.Profile.ID
		}

		tableData = append(tableData, []string{
			browser.SessionID,
			util.FormatLocal(browser.CreatedAt),
			persistentID,
			profile,
			truncateURL(browser.CdpWsURL, 50),
			truncateURL(browser.BrowserLiveViewURL, 50),
		})
	}

	PrintTableNoPad(tableData, true)
	return nil
}

func (b BrowsersCmd) Create(ctx context.Context, in BrowsersCreateInput) error {
	pterm.Info.Println("Creating browser session...")
	params := kernel.BrowserNewParams{}
	if in.PersistenceID != "" {
		params.Persistence = kernel.BrowserPersistenceParam{ID: in.PersistenceID}
	}
	if in.TimeoutSeconds > 0 {
		params.TimeoutSeconds = kernel.Opt(int64(in.TimeoutSeconds))
	}
	if in.Stealth.Set {
		params.Stealth = kernel.Opt(in.Stealth.Value)
	}
	if in.Headless.Set {
		params.Headless = kernel.Opt(in.Headless.Value)
	}

	// Validate profile selection: at most one of profile-id or profile-name must be provided
	if in.ProfileID != "" && in.ProfileName != "" {
		pterm.Error.Println("must specify at most one of --profile-id or --profile-name")
		return nil
	} else if in.ProfileID != "" || in.ProfileName != "" {
		params.Profile = kernel.BrowserNewParamsProfile{
			SaveChanges: kernel.Opt(in.ProfileSaveChanges.Value),
		}
		if in.ProfileID != "" {
			params.Profile.ID = kernel.Opt(in.ProfileID)
		} else if in.ProfileName != "" {
			params.Profile.Name = kernel.Opt(in.ProfileName)
		}
	}

	// Add proxy if specified
	if in.ProxyID != "" {
		params.ProxyID = kernel.Opt(in.ProxyID)
	}

	browser, err := b.browsers.New(ctx, params)
	if err != nil {
		return util.CleanedUpSdkError{Err: err}
	}

	tableData := pterm.TableData{
		{"Property", "Value"},
		{"Session ID", browser.SessionID},
		{"CDP WebSocket URL", browser.CdpWsURL},
	}
	if browser.BrowserLiveViewURL != "" {
		tableData = append(tableData, []string{"Live View URL", browser.BrowserLiveViewURL})
	}
	if browser.Persistence.ID != "" {
		tableData = append(tableData, []string{"Persistent ID", browser.Persistence.ID})
	}
	if browser.Profile.ID != "" || browser.Profile.Name != "" {
		profVal := browser.Profile.Name
		if profVal == "" {
			profVal = browser.Profile.ID
		}
		tableData = append(tableData, []string{"Profile", profVal})
	}

	PrintTableNoPad(tableData, true)
	return nil
}

func (b BrowsersCmd) Delete(ctx context.Context, in BrowsersDeleteInput) error {
	if !in.SkipConfirm {
		browsers, err := b.browsers.List(ctx)
		if err != nil {
			return util.CleanedUpSdkError{Err: err}
		}
		if browsers == nil || len(*browsers) == 0 {
			pterm.Error.Println("No browsers found")
			return nil
		}

		var found *kernel.BrowserListResponse
		for _, br := range *browsers {
			if br.SessionID == in.Identifier || br.Persistence.ID == in.Identifier {
				bCopy := br
				found = &bCopy
				break
			}
		}
		if found == nil {
			pterm.Error.Printf("Browser '%s' not found\n", in.Identifier)
			return nil
		}

		var confirmMsg string
		if found.Persistence.ID == in.Identifier {
			confirmMsg = fmt.Sprintf("Are you sure you want to delete browser with persistent ID \"%s\"?", in.Identifier)
		} else {
			confirmMsg = fmt.Sprintf("Are you sure you want to delete browser with ID \"%s\"?", in.Identifier)
		}
		pterm.DefaultInteractiveConfirm.DefaultText = confirmMsg
		result, _ := pterm.DefaultInteractiveConfirm.Show()
		if !result {
			pterm.Info.Println("Deletion cancelled")
			return nil
		}

		if found.Persistence.ID == in.Identifier {
			pterm.Info.Printf("Deleting browser with persistent ID: %s\n", in.Identifier)
			err = b.browsers.Delete(ctx, kernel.BrowserDeleteParams{PersistentID: in.Identifier})
			if err != nil && !util.IsNotFound(err) {
				return util.CleanedUpSdkError{Err: err}
			}
			pterm.Success.Printf("Successfully deleted browser with persistent ID: %s\n", in.Identifier)
			return nil
		}

		pterm.Info.Printf("Deleting browser with ID: %s\n", in.Identifier)
		err = b.browsers.DeleteByID(ctx, in.Identifier)
		if err != nil && !util.IsNotFound(err) {
			return util.CleanedUpSdkError{Err: err}
		}
		pterm.Success.Printf("Successfully deleted browser with ID: %s\n", in.Identifier)
		return nil
	}

	// Skip confirmation: try both deletion modes without listing first
	// Treat not found as a success (idempotent delete)
	var nonNotFoundErrors []error

	// Attempt by session ID
	if err := b.browsers.DeleteByID(ctx, in.Identifier); err != nil {
		if !util.IsNotFound(err) {
			nonNotFoundErrors = append(nonNotFoundErrors, err)
		}
	}

	// Attempt by persistent ID
	if err := b.browsers.Delete(ctx, kernel.BrowserDeleteParams{PersistentID: in.Identifier}); err != nil {
		if !util.IsNotFound(err) {
			nonNotFoundErrors = append(nonNotFoundErrors, err)
		}
	}

	if len(nonNotFoundErrors) >= 2 {
		// Both failed with meaningful errors; report one
		return util.CleanedUpSdkError{Err: nonNotFoundErrors[0]}
	}

	pterm.Success.Printf("Successfully deleted (or already absent) browser: %s\n", in.Identifier)
	return nil
}

func (b BrowsersCmd) View(ctx context.Context, in BrowsersViewInput) error {
	browsers, err := b.browsers.List(ctx)
	if err != nil {
		return util.CleanedUpSdkError{Err: err}
	}

	if browsers == nil || len(*browsers) == 0 {
		pterm.Error.Println("No browsers found")
		return nil
	}

	var foundBrowser *kernel.BrowserListResponse
	for _, browser := range *browsers {
		if browser.Persistence.ID == in.Identifier || browser.SessionID == in.Identifier {
			foundBrowser = &browser
			break
		}
	}

	if foundBrowser == nil {
		pterm.Error.Printf("Browser '%s' not found\n", in.Identifier)
		return nil
	}

	// Output just the URL
	pterm.Info.Println(foundBrowser.BrowserLiveViewURL)
	return nil
}

// Logs
type BrowsersLogsStreamInput struct {
	Identifier        string
	Source            string
	Follow            BoolFlag
	Path              string
	SupervisorProcess string
}

func (b BrowsersCmd) LogsStream(ctx context.Context, in BrowsersLogsStreamInput) error {
	if b.logs == nil {
		pterm.Error.Println("logs service not available")
		return nil
	}
	br, err := b.resolveBrowserByIdentifier(ctx, in.Identifier)
	if err != nil {
		return util.CleanedUpSdkError{Err: err}
	}
	if br == nil {
		pterm.Error.Printf("Browser '%s' not found\n", in.Identifier)
		return nil
	}
	params := kernel.BrowserLogStreamParams{Source: kernel.BrowserLogStreamParamsSource(in.Source)}
	if in.Follow.Set {
		params.Follow = kernel.Opt(in.Follow.Value)
	}
	if in.Path != "" {
		params.Path = kernel.Opt(in.Path)
	}
	if in.SupervisorProcess != "" {
		params.SupervisorProcess = kernel.Opt(in.SupervisorProcess)
	}
	stream := b.logs.StreamStreaming(ctx, br.SessionID, params)
	if stream == nil {
		pterm.Error.Println("failed to open log stream")
		return nil
	}
	defer stream.Close()
	for stream.Next() {
		ev := stream.Current()
		pterm.Println(fmt.Sprintf("[%s] %s", util.FormatLocal(ev.Timestamp), ev.Message))
	}
	if err := stream.Err(); err != nil {
		return util.CleanedUpSdkError{Err: err}
	}
	return nil
}

// Replays
type BrowsersReplaysListInput struct {
	Identifier string
}

type BrowsersReplaysStartInput struct {
	Identifier         string
	Framerate          int
	MaxDurationSeconds int
}

type BrowsersReplaysStopInput struct {
	Identifier string
	ReplayID   string
}

type BrowsersReplaysDownloadInput struct {
	Identifier string
	ReplayID   string
	Output     string
}

func (b BrowsersCmd) ReplaysList(ctx context.Context, in BrowsersReplaysListInput) error {
	br, err := b.resolveBrowserByIdentifier(ctx, in.Identifier)
	if err != nil {
		return util.CleanedUpSdkError{Err: err}
	}
	if br == nil {
		pterm.Error.Printf("Browser '%s' not found\n", in.Identifier)
		return nil
	}
	items, err := b.replays.List(ctx, br.SessionID)
	if err != nil {
		return util.CleanedUpSdkError{Err: err}
	}
	if items == nil || len(*items) == 0 {
		pterm.Info.Println("No replays found")
		return nil
	}
	rows := pterm.TableData{{"Replay ID", "Started At", "Finished At", "View URL"}}
	for _, r := range *items {
		rows = append(rows, []string{r.ReplayID, util.FormatLocal(r.StartedAt), util.FormatLocal(r.FinishedAt), truncateURL(r.ReplayViewURL, 60)})
	}
	PrintTableNoPad(rows, true)
	return nil
}

func (b BrowsersCmd) ReplaysStart(ctx context.Context, in BrowsersReplaysStartInput) error {
	br, err := b.resolveBrowserByIdentifier(ctx, in.Identifier)
	if err != nil {
		return util.CleanedUpSdkError{Err: err}
	}
	if br == nil {
		pterm.Error.Printf("Browser '%s' not found\n", in.Identifier)
		return nil
	}
	body := kernel.BrowserReplayStartParams{}
	if in.Framerate > 0 {
		body.Framerate = kernel.Opt(int64(in.Framerate))
	}
	if in.MaxDurationSeconds > 0 {
		body.MaxDurationInSeconds = kernel.Opt(int64(in.MaxDurationSeconds))
	}
	res, err := b.replays.Start(ctx, br.SessionID, body)
	if err != nil {
		return util.CleanedUpSdkError{Err: err}
	}
	rows := pterm.TableData{{"Property", "Value"}, {"Replay ID", res.ReplayID}, {"View URL", res.ReplayViewURL}, {"Started At", util.FormatLocal(res.StartedAt)}}
	PrintTableNoPad(rows, true)
	return nil
}

func (b BrowsersCmd) ReplaysStop(ctx context.Context, in BrowsersReplaysStopInput) error {
	br, err := b.resolveBrowserByIdentifier(ctx, in.Identifier)
	if err != nil {
		return util.CleanedUpSdkError{Err: err}
	}
	if br == nil {
		pterm.Error.Printf("Browser '%s' not found\n", in.Identifier)
		return nil
	}
	err = b.replays.Stop(ctx, in.ReplayID, kernel.BrowserReplayStopParams{ID: br.SessionID})
	if err != nil {
		return util.CleanedUpSdkError{Err: err}
	}
	pterm.Success.Printf("Stopped replay %s for browser %s\n", in.ReplayID, br.SessionID)
	return nil
}

func (b BrowsersCmd) ReplaysDownload(ctx context.Context, in BrowsersReplaysDownloadInput) error {
	br, err := b.resolveBrowserByIdentifier(ctx, in.Identifier)
	if err != nil {
		return util.CleanedUpSdkError{Err: err}
	}
	if br == nil {
		pterm.Error.Printf("Browser '%s' not found\n", in.Identifier)
		return nil
	}
	res, err := b.replays.Download(ctx, in.ReplayID, kernel.BrowserReplayDownloadParams{ID: br.SessionID})
	if err != nil {
		return util.CleanedUpSdkError{Err: err}
	}
	defer res.Body.Close()
	if in.Output == "" {
		pterm.Info.Printf("Downloaded replay %s (%s)\n", in.ReplayID, res.Header.Get("content-type"))
		_, _ = io.Copy(io.Discard, res.Body)
		return nil
	}
	f, err := os.Create(in.Output)
	if err != nil {
		pterm.Error.Printf("Failed to create file: %v\n", err)
		return nil
	}
	defer f.Close()
	if _, err := io.Copy(f, res.Body); err != nil {
		pterm.Error.Printf("Failed to write file: %v\n", err)
		return nil
	}
	pterm.Success.Printf("Saved replay to %s\n", in.Output)
	return nil
}

// Process
type BrowsersProcessExecInput struct {
	Identifier string
	Command    string
	Args       []string
	Cwd        string
	Timeout    int
	AsUser     string
	AsRoot     BoolFlag
}

type BrowsersProcessSpawnInput = BrowsersProcessExecInput

type BrowsersProcessKillInput struct {
	Identifier string
	ProcessID  string
	Signal     string
}

type BrowsersProcessStatusInput struct {
	Identifier string
	ProcessID  string
}

type BrowsersProcessStdinInput struct {
	Identifier string
	ProcessID  string
	DataB64    string
}

type BrowsersProcessStdoutStreamInput struct {
	Identifier string
	ProcessID  string
}

func (b BrowsersCmd) ProcessExec(ctx context.Context, in BrowsersProcessExecInput) error {
	if b.process == nil {
		pterm.Error.Println("process service not available")
		return nil
	}
	br, err := b.resolveBrowserByIdentifier(ctx, in.Identifier)
	if err != nil {
		return util.CleanedUpSdkError{Err: err}
	}
	if br == nil {
		pterm.Error.Printf("Browser '%s' not found\n", in.Identifier)
		return nil
	}
	params := kernel.BrowserProcessExecParams{Command: in.Command}
	if len(in.Args) > 0 {
		params.Args = in.Args
	}
	if in.Cwd != "" {
		params.Cwd = kernel.Opt(in.Cwd)
	}
	if in.Timeout > 0 {
		params.TimeoutSec = kernel.Opt(int64(in.Timeout))
	}
	if in.AsUser != "" {
		params.AsUser = kernel.Opt(in.AsUser)
	}
	if in.AsRoot.Set {
		params.AsRoot = kernel.Opt(in.AsRoot.Value)
	}
	res, err := b.process.Exec(ctx, br.SessionID, params)
	if err != nil {
		return util.CleanedUpSdkError{Err: err}
	}
	rows := pterm.TableData{{"Property", "Value"}, {"Exit Code", fmt.Sprintf("%d", res.ExitCode)}, {"Duration (ms)", fmt.Sprintf("%d", res.DurationMs)}}
	PrintTableNoPad(rows, true)
	if res.StdoutB64 != "" {
		data, err := base64.StdEncoding.DecodeString(res.StdoutB64)
		if err != nil {
			pterm.Error.Printf("stdout decode error: %v\n", err)
		} else if len(data) > 0 {
			pterm.Info.Println("stdout:")
			os.Stdout.Write(data)
			if data[len(data)-1] != '\n' {
				fmt.Println()
			}
		}
	}
	if res.StderrB64 != "" {
		data, err := base64.StdEncoding.DecodeString(res.StderrB64)
		if err != nil {
			pterm.Error.Printf("stderr decode error: %v\n", err)
		} else if len(data) > 0 {
			pterm.Info.Println("stderr:")
			os.Stderr.Write(data)
			if data[len(data)-1] != '\n' {
				fmt.Fprintln(os.Stderr)
			}
		}
	}
	return nil
}

func (b BrowsersCmd) ProcessSpawn(ctx context.Context, in BrowsersProcessSpawnInput) error {
	if b.process == nil {
		pterm.Error.Println("process service not available")
		return nil
	}
	br, err := b.resolveBrowserByIdentifier(ctx, in.Identifier)
	if err != nil {
		return util.CleanedUpSdkError{Err: err}
	}
	if br == nil {
		pterm.Error.Printf("Browser '%s' not found\n", in.Identifier)
		return nil
	}
	params := kernel.BrowserProcessSpawnParams{Command: in.Command}
	if len(in.Args) > 0 {
		params.Args = in.Args
	}
	if in.Cwd != "" {
		params.Cwd = kernel.Opt(in.Cwd)
	}
	if in.Timeout > 0 {
		params.TimeoutSec = kernel.Opt(int64(in.Timeout))
	}
	if in.AsUser != "" {
		params.AsUser = kernel.Opt(in.AsUser)
	}
	if in.AsRoot.Set {
		params.AsRoot = kernel.Opt(in.AsRoot.Value)
	}
	res, err := b.process.Spawn(ctx, br.SessionID, params)
	if err != nil {
		return util.CleanedUpSdkError{Err: err}
	}
	rows := pterm.TableData{{"Property", "Value"}, {"Process ID", res.ProcessID}, {"PID", fmt.Sprintf("%d", res.Pid)}, {"Started At", util.FormatLocal(res.StartedAt)}}
	PrintTableNoPad(rows, true)
	return nil
}

func (b BrowsersCmd) ProcessKill(ctx context.Context, in BrowsersProcessKillInput) error {
	if b.process == nil {
		pterm.Error.Println("process service not available")
		return nil
	}
	br, err := b.resolveBrowserByIdentifier(ctx, in.Identifier)
	if err != nil {
		return util.CleanedUpSdkError{Err: err}
	}
	if br == nil {
		pterm.Error.Printf("Browser '%s' not found\n", in.Identifier)
		return nil
	}
	params := kernel.BrowserProcessKillParams{ID: br.SessionID, Signal: kernel.BrowserProcessKillParamsSignal(in.Signal)}
	_, err = b.process.Kill(ctx, in.ProcessID, params)
	if err != nil {
		return util.CleanedUpSdkError{Err: err}
	}
	pterm.Success.Printf("Sent %s to process %s on %s\n", in.Signal, in.ProcessID, br.SessionID)
	return nil
}

func (b BrowsersCmd) ProcessStatus(ctx context.Context, in BrowsersProcessStatusInput) error {
	if b.process == nil {
		pterm.Error.Println("process service not available")
		return nil
	}
	br, err := b.resolveBrowserByIdentifier(ctx, in.Identifier)
	if err != nil {
		return util.CleanedUpSdkError{Err: err}
	}
	if br == nil {
		pterm.Error.Printf("Browser '%s' not found\n", in.Identifier)
		return nil
	}
	res, err := b.process.Status(ctx, in.ProcessID, kernel.BrowserProcessStatusParams{ID: br.SessionID})
	if err != nil {
		return util.CleanedUpSdkError{Err: err}
	}
	rows := pterm.TableData{{"Property", "Value"}, {"State", string(res.State)}, {"CPU %", fmt.Sprintf("%.2f", res.CPUPct)}, {"Mem Bytes", fmt.Sprintf("%d", res.MemBytes)}, {"Exit Code", fmt.Sprintf("%d", res.ExitCode)}}
	PrintTableNoPad(rows, true)
	return nil
}

func (b BrowsersCmd) ProcessStdin(ctx context.Context, in BrowsersProcessStdinInput) error {
	if b.process == nil {
		pterm.Error.Println("process service not available")
		return nil
	}
	br, err := b.resolveBrowserByIdentifier(ctx, in.Identifier)
	if err != nil {
		return util.CleanedUpSdkError{Err: err}
	}
	if br == nil {
		pterm.Error.Printf("Browser '%s' not found\n", in.Identifier)
		return nil
	}
	_, err = b.process.Stdin(ctx, in.ProcessID, kernel.BrowserProcessStdinParams{ID: br.SessionID, DataB64: in.DataB64})
	if err != nil {
		return util.CleanedUpSdkError{Err: err}
	}
	pterm.Success.Println("Wrote to stdin")
	return nil
}

func (b BrowsersCmd) ProcessStdoutStream(ctx context.Context, in BrowsersProcessStdoutStreamInput) error {
	if b.process == nil {
		pterm.Error.Println("process service not available")
		return nil
	}
	br, err := b.resolveBrowserByIdentifier(ctx, in.Identifier)
	if err != nil {
		return util.CleanedUpSdkError{Err: err}
	}
	if br == nil {
		pterm.Error.Printf("Browser '%s' not found\n", in.Identifier)
		return nil
	}
	stream := b.process.StdoutStreamStreaming(ctx, in.ProcessID, kernel.BrowserProcessStdoutStreamParams{ID: br.SessionID})
	if stream == nil {
		pterm.Error.Println("failed to open stdout stream")
		return nil
	}
	defer stream.Close()
	for stream.Next() {
		ev := stream.Current()
		if ev.Event == "exit" {
			pterm.Info.Printf("process exited with code %d\n", ev.ExitCode)
			continue
		}
		data, err := base64.StdEncoding.DecodeString(ev.DataB64)
		if err != nil {
			pterm.Error.Printf("decode error: %v\n", err)
			continue
		}
		os.Stdout.Write(data)
	}
	if err := stream.Err(); err != nil {
		return util.CleanedUpSdkError{Err: err}
	}
	return nil
}

// FS (minimal scaffolding)
type BrowsersFSNewDirInput struct {
	Identifier string
	Path       string
	Mode       string
}

type BrowsersFSDeleteDirInput struct {
	Identifier string
	Path       string
}

type BrowsersFSDeleteFileInput struct {
	Identifier string
	Path       string
}

type BrowsersFSDownloadDirZipInput struct {
	Identifier string
	Path       string
	Output     string
}

type BrowsersFSFileInfoInput struct {
	Identifier string
	Path       string
}

type BrowsersFSListFilesInput struct {
	Identifier string
	Path       string
}

type BrowsersFSMoveInput struct {
	Identifier string
	SrcPath    string
	DestPath   string
}

type BrowsersFSReadFileInput struct {
	Identifier string
	Path       string
	Output     string
}

type BrowsersFSSetPermsInput struct {
	Identifier string
	Path       string
	Mode       string
	Owner      string
	Group      string
}

// Upload inputs
type BrowsersFSUploadInput struct {
	Identifier string
	Mappings   []struct {
		Local string
		Dest  string
	}
	DestDir string
	Paths   []string
}

type BrowsersFSUploadZipInput struct {
	Identifier string
	ZipPath    string
	DestDir    string
}

type BrowsersFSWriteFileInput struct {
	Identifier string
	DestPath   string
	Mode       string
	SourcePath string
}

func (b BrowsersCmd) FSNewDirectory(ctx context.Context, in BrowsersFSNewDirInput) error {
	if b.fs == nil {
		pterm.Error.Println("fs service not available")
		return nil
	}
	br, err := b.resolveBrowserByIdentifier(ctx, in.Identifier)
	if err != nil {
		return util.CleanedUpSdkError{Err: err}
	}
	if br == nil {
		pterm.Error.Printf("Browser '%s' not found\n", in.Identifier)
		return nil
	}
	params := kernel.BrowserFNewDirectoryParams{Path: in.Path}
	if in.Mode != "" {
		params.Mode = kernel.Opt(in.Mode)
	}
	if err := b.fs.NewDirectory(ctx, br.SessionID, params); err != nil {
		return util.CleanedUpSdkError{Err: err}
	}
	pterm.Success.Printf("Created directory %s\n", in.Path)
	return nil
}

func (b BrowsersCmd) FSDeleteDirectory(ctx context.Context, in BrowsersFSDeleteDirInput) error {
	if b.fs == nil {
		pterm.Error.Println("fs service not available")
		return nil
	}
	br, err := b.resolveBrowserByIdentifier(ctx, in.Identifier)
	if err != nil {
		return util.CleanedUpSdkError{Err: err}
	}
	if br == nil {
		pterm.Error.Printf("Browser '%s' not found\n", in.Identifier)
		return nil
	}
	if err := b.fs.DeleteDirectory(ctx, br.SessionID, kernel.BrowserFDeleteDirectoryParams{Path: in.Path}); err != nil {
		return util.CleanedUpSdkError{Err: err}
	}
	pterm.Success.Printf("Deleted directory %s\n", in.Path)
	return nil
}

func (b BrowsersCmd) FSDeleteFile(ctx context.Context, in BrowsersFSDeleteFileInput) error {
	if b.fs == nil {
		pterm.Error.Println("fs service not available")
		return nil
	}
	br, err := b.resolveBrowserByIdentifier(ctx, in.Identifier)
	if err != nil {
		return util.CleanedUpSdkError{Err: err}
	}
	if br == nil {
		pterm.Error.Printf("Browser '%s' not found\n", in.Identifier)
		return nil
	}
	if err := b.fs.DeleteFile(ctx, br.SessionID, kernel.BrowserFDeleteFileParams{Path: in.Path}); err != nil {
		return util.CleanedUpSdkError{Err: err}
	}
	pterm.Success.Printf("Deleted file %s\n", in.Path)
	return nil
}

func (b BrowsersCmd) FSDownloadDirZip(ctx context.Context, in BrowsersFSDownloadDirZipInput) error {
	if b.fs == nil {
		pterm.Error.Println("fs service not available")
		return nil
	}
	br, err := b.resolveBrowserByIdentifier(ctx, in.Identifier)
	if err != nil {
		return util.CleanedUpSdkError{Err: err}
	}
	if br == nil {
		pterm.Error.Printf("Browser '%s' not found\n", in.Identifier)
		return nil
	}
	res, err := b.fs.DownloadDirZip(ctx, br.SessionID, kernel.BrowserFDownloadDirZipParams{Path: in.Path})
	if err != nil {
		return util.CleanedUpSdkError{Err: err}
	}
	defer res.Body.Close()
	if in.Output == "" {
		_, _ = io.Copy(io.Discard, res.Body)
		pterm.Info.Println("Downloaded zip (discarded; specify --output to save)")
		return nil
	}
	f, err := os.Create(in.Output)
	if err != nil {
		pterm.Error.Printf("Failed to create file: %v\n", err)
		return nil
	}
	defer f.Close()
	if _, err := io.Copy(f, res.Body); err != nil {
		pterm.Error.Printf("Failed to write file: %v\n", err)
		return nil
	}
	pterm.Success.Printf("Saved zip to %s\n", in.Output)
	return nil
}

func (b BrowsersCmd) FSFileInfo(ctx context.Context, in BrowsersFSFileInfoInput) error {
	if b.fs == nil {
		pterm.Error.Println("fs service not available")
		return nil
	}
	br, err := b.resolveBrowserByIdentifier(ctx, in.Identifier)
	if err != nil {
		return util.CleanedUpSdkError{Err: err}
	}
	if br == nil {
		pterm.Error.Printf("Browser '%s' not found\n", in.Identifier)
		return nil
	}
	res, err := b.fs.FileInfo(ctx, br.SessionID, kernel.BrowserFFileInfoParams{Path: in.Path})
	if err != nil {
		return util.CleanedUpSdkError{Err: err}
	}
	rows := pterm.TableData{{"Property", "Value"}, {"Path", res.Path}, {"Name", res.Name}, {"Mode", res.Mode}, {"IsDir", fmt.Sprintf("%t", res.IsDir)}, {"SizeBytes", fmt.Sprintf("%d", res.SizeBytes)}, {"ModTime", util.FormatLocal(res.ModTime)}}
	PrintTableNoPad(rows, true)
	return nil
}

func (b BrowsersCmd) FSListFiles(ctx context.Context, in BrowsersFSListFilesInput) error {
	if b.fs == nil {
		pterm.Error.Println("fs service not available")
		return nil
	}
	br, err := b.resolveBrowserByIdentifier(ctx, in.Identifier)
	if err != nil {
		return util.CleanedUpSdkError{Err: err}
	}
	if br == nil {
		pterm.Error.Printf("Browser '%s' not found\n", in.Identifier)
		return nil
	}
	res, err := b.fs.ListFiles(ctx, br.SessionID, kernel.BrowserFListFilesParams{Path: in.Path})
	if err != nil {
		return util.CleanedUpSdkError{Err: err}
	}
	if res == nil || len(*res) == 0 {
		pterm.Info.Println("No files found")
		return nil
	}
	rows := pterm.TableData{{"Mode", "Size", "ModTime", "Name", "Path"}}
	for _, f := range *res {
		rows = append(rows, []string{f.Mode, fmt.Sprintf("%d", f.SizeBytes), util.FormatLocal(f.ModTime), f.Name, f.Path})
	}
	PrintTableNoPad(rows, true)
	return nil
}

func (b BrowsersCmd) FSMove(ctx context.Context, in BrowsersFSMoveInput) error {
	if b.fs == nil {
		pterm.Error.Println("fs service not available")
		return nil
	}
	br, err := b.resolveBrowserByIdentifier(ctx, in.Identifier)
	if err != nil {
		return util.CleanedUpSdkError{Err: err}
	}
	if br == nil {
		pterm.Error.Printf("Browser '%s' not found\n", in.Identifier)
		return nil
	}
	if err := b.fs.Move(ctx, br.SessionID, kernel.BrowserFMoveParams{SrcPath: in.SrcPath, DestPath: in.DestPath}); err != nil {
		return util.CleanedUpSdkError{Err: err}
	}
	pterm.Success.Printf("Moved %s -> %s\n", in.SrcPath, in.DestPath)
	return nil
}

func (b BrowsersCmd) FSReadFile(ctx context.Context, in BrowsersFSReadFileInput) error {
	if b.fs == nil {
		pterm.Error.Println("fs service not available")
		return nil
	}
	br, err := b.resolveBrowserByIdentifier(ctx, in.Identifier)
	if err != nil {
		return util.CleanedUpSdkError{Err: err}
	}
	if br == nil {
		pterm.Error.Printf("Browser '%s' not found\n", in.Identifier)
		return nil
	}
	res, err := b.fs.ReadFile(ctx, br.SessionID, kernel.BrowserFReadFileParams{Path: in.Path})
	if err != nil {
		return util.CleanedUpSdkError{Err: err}
	}
	defer res.Body.Close()
	if in.Output == "" {
		_, _ = io.Copy(os.Stdout, res.Body)
		return nil
	}
	f, err := os.Create(in.Output)
	if err != nil {
		pterm.Error.Printf("Failed to create file: %v\n", err)
		return nil
	}
	defer f.Close()
	if _, err := io.Copy(f, res.Body); err != nil {
		pterm.Error.Printf("Failed to write file: %v\n", err)
		return nil
	}
	pterm.Success.Printf("Saved file to %s\n", in.Output)
	return nil
}

func (b BrowsersCmd) FSSetPermissions(ctx context.Context, in BrowsersFSSetPermsInput) error {
	if b.fs == nil {
		pterm.Error.Println("fs service not available")
		return nil
	}
	br, err := b.resolveBrowserByIdentifier(ctx, in.Identifier)
	if err != nil {
		return util.CleanedUpSdkError{Err: err}
	}
	if br == nil {
		pterm.Error.Printf("Browser '%s' not found\n", in.Identifier)
		return nil
	}
	params := kernel.BrowserFSetFilePermissionsParams{Path: in.Path, Mode: in.Mode}
	if in.Owner != "" {
		params.Owner = kernel.Opt(in.Owner)
	}
	if in.Group != "" {
		params.Group = kernel.Opt(in.Group)
	}
	if err := b.fs.SetFilePermissions(ctx, br.SessionID, params); err != nil {
		return util.CleanedUpSdkError{Err: err}
	}
	pterm.Success.Printf("Updated permissions for %s\n", in.Path)
	return nil
}

func (b BrowsersCmd) FSUpload(ctx context.Context, in BrowsersFSUploadInput) error {
	if b.fs == nil {
		pterm.Error.Println("fs service not available")
		return nil
	}
	br, err := b.resolveBrowserByIdentifier(ctx, in.Identifier)
	if err != nil {
		return util.CleanedUpSdkError{Err: err}
	}
	if br == nil {
		pterm.Error.Printf("Browser '%s' not found\n", in.Identifier)
		return nil
	}
	var files []kernel.BrowserFUploadParamsFile
	var toClose []io.Closer
	for _, m := range in.Mappings {
		f, err := os.Open(m.Local)
		if err != nil {
			pterm.Error.Printf("Failed to open %s: %v\n", m.Local, err)
			for _, c := range toClose {
				_ = c.Close()
			}
			return nil
		}
		toClose = append(toClose, f)
		files = append(files, kernel.BrowserFUploadParamsFile{DestPath: m.Dest, File: f})
	}
	if in.DestDir != "" && len(in.Paths) > 0 {
		for _, lp := range in.Paths {
			f, err := os.Open(lp)
			if err != nil {
				pterm.Error.Printf("Failed to open %s: %v\n", lp, err)
				for _, c := range toClose {
					_ = c.Close()
				}
				return nil
			}
			toClose = append(toClose, f)
			dest := filepath.Join(in.DestDir, filepath.Base(lp))
			files = append(files, kernel.BrowserFUploadParamsFile{DestPath: dest, File: f})
		}
	}
	if len(files) == 0 {
		pterm.Error.Println("no files specified for upload")
		return nil
	}
	defer func() {
		for _, c := range toClose {
			_ = c.Close()
		}
	}()
	if err := b.fs.Upload(ctx, br.SessionID, kernel.BrowserFUploadParams{Files: files}); err != nil {
		return util.CleanedUpSdkError{Err: err}
	}
	if len(files) == 1 {
		pterm.Success.Println("Uploaded 1 file")
	} else {
		pterm.Success.Printf("Uploaded %d files\n", len(files))
	}
	return nil
}

func (b BrowsersCmd) FSUploadZip(ctx context.Context, in BrowsersFSUploadZipInput) error {
	if b.fs == nil {
		pterm.Error.Println("fs service not available")
		return nil
	}
	br, err := b.resolveBrowserByIdentifier(ctx, in.Identifier)
	if err != nil {
		return util.CleanedUpSdkError{Err: err}
	}
	if br == nil {
		pterm.Error.Printf("Browser '%s' not found\n", in.Identifier)
		return nil
	}
	f, err := os.Open(in.ZipPath)
	if err != nil {
		pterm.Error.Printf("Failed to open zip: %v\n", err)
		return nil
	}
	defer f.Close()
	if err := b.fs.UploadZip(ctx, br.SessionID, kernel.BrowserFUploadZipParams{DestPath: in.DestDir, ZipFile: f}); err != nil {
		return util.CleanedUpSdkError{Err: err}
	}
	pterm.Success.Printf("Uploaded zip to %s\n", in.DestDir)
	return nil
}

func (b BrowsersCmd) FSWriteFile(ctx context.Context, in BrowsersFSWriteFileInput) error {
	if b.fs == nil {
		pterm.Error.Println("fs service not available")
		return nil
	}
	br, err := b.resolveBrowserByIdentifier(ctx, in.Identifier)
	if err != nil {
		return util.CleanedUpSdkError{Err: err}
	}
	if br == nil {
		pterm.Error.Printf("Browser '%s' not found\n", in.Identifier)
		return nil
	}
	var reader io.Reader
	if in.SourcePath != "" {
		f, err := os.Open(in.SourcePath)
		if err != nil {
			pterm.Error.Printf("Failed to open input: %v\n", err)
			return nil
		}
		defer f.Close()
		reader = f
	} else {
		pterm.Error.Println("--source is required")
		return nil
	}
	params := kernel.BrowserFWriteFileParams{Path: in.DestPath}
	if in.Mode != "" {
		params.Mode = kernel.Opt(in.Mode)
	}
	if err := b.fs.WriteFile(ctx, br.SessionID, reader, params); err != nil {
		return util.CleanedUpSdkError{Err: err}
	}
	pterm.Success.Printf("Wrote file to %s\n", in.DestPath)
	return nil
}

var browsersCmd = &cobra.Command{
	Use:   "browsers",
	Short: "Manage browsers",
	Long:  "Commands for managing Kernel browsers",
}

var browsersListCmd = &cobra.Command{
	Use:   "list",
	Short: "List running or persistent browsers",
	RunE:  runBrowsersList,
}

var browsersCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new browser session",
	RunE:  runBrowsersCreate,
}

var browsersDeleteCmd = &cobra.Command{
	Use:   "delete <id-or-persistent-id>",
	Short: "Delete a browser",
	Args:  cobra.ExactArgs(1),
	RunE:  runBrowsersDelete,
}

var browsersViewCmd = &cobra.Command{
	Use:   "view <id|persistent-id>",
	Short: "Get the live view URL for a browser",
	Args:  cobra.ExactArgs(1),
	RunE:  runBrowsersView,
}

func init() {
	browsersCmd.AddCommand(browsersListCmd)
	browsersCmd.AddCommand(browsersCreateCmd)
	browsersCmd.AddCommand(browsersDeleteCmd)
	browsersCmd.AddCommand(browsersViewCmd)

	// logs
	logsRoot := &cobra.Command{Use: "logs", Short: "Browser logs operations"}
	logsStream := &cobra.Command{Use: "stream <id|persistent-id>", Short: "Stream browser logs", Args: cobra.ExactArgs(1), RunE: runBrowsersLogsStream}
	logsStream.Flags().String("source", "", "Log source: path or supervisor")
	logsStream.Flags().Bool("follow", true, "Follow the log stream")
	logsStream.Flags().String("path", "", "File path when source=path")
	logsStream.Flags().String("supervisor-process", "", "Supervisor process name when source=supervisor. Useful values to use: chromium, kernel-images-api, neko")
	_ = logsStream.MarkFlagRequired("source")
	logsRoot.AddCommand(logsStream)
	browsersCmd.AddCommand(logsRoot)

	// replays
	replaysRoot := &cobra.Command{Use: "replays", Short: "Manage browser replays"}
	replaysList := &cobra.Command{Use: "list <id|persistent-id>", Short: "List replays for a browser", Args: cobra.ExactArgs(1), RunE: runBrowsersReplaysList}
	replaysStart := &cobra.Command{Use: "start <id|persistent-id>", Short: "Start a replay recording", Args: cobra.ExactArgs(1), RunE: runBrowsersReplaysStart}
	replaysStart.Flags().Int("framerate", 0, "Recording framerate (fps)")
	replaysStart.Flags().Int("max-duration", 0, "Maximum duration in seconds")
	replaysStop := &cobra.Command{Use: "stop <id|persistent-id> <replay-id>", Short: "Stop a replay recording", Args: cobra.ExactArgs(2), RunE: runBrowsersReplaysStop}
	replaysDownload := &cobra.Command{Use: "download <id|persistent-id> <replay-id>", Short: "Download a replay video", Args: cobra.ExactArgs(2), RunE: runBrowsersReplaysDownload}
	replaysDownload.Flags().StringP("output", "o", "", "Output file path for the replay video")
	replaysRoot.AddCommand(replaysList, replaysStart, replaysStop, replaysDownload)
	browsersCmd.AddCommand(replaysRoot)

	// process
	procRoot := &cobra.Command{Use: "process", Short: "Manage processes inside the browser VM"}
	procExec := &cobra.Command{Use: "exec <id|persistent-id> [--] [command...]", Short: "Execute a command synchronously", Args: cobra.MinimumNArgs(1), RunE: runBrowsersProcessExec}
	procExec.Flags().String("command", "", "Command to execute (optional; if omitted, trailing args are executed via /bin/bash -c)")
	procExec.Flags().StringSlice("args", []string{}, "Command arguments")
	procExec.Flags().String("cwd", "", "Working directory")
	procExec.Flags().Int("timeout", 0, "Timeout in seconds")
	procExec.Flags().String("as-user", "", "Run as user")
	procExec.Flags().Bool("as-root", false, "Run as root")
	procSpawn := &cobra.Command{Use: "spawn <id|persistent-id> [--] [command...]", Short: "Execute a command asynchronously", Args: cobra.MinimumNArgs(1), RunE: runBrowsersProcessSpawn}
	procSpawn.Flags().String("command", "", "Command to execute (optional; if omitted, trailing args are executed via /bin/bash -c)")
	procSpawn.Flags().StringSlice("args", []string{}, "Command arguments")
	procSpawn.Flags().String("cwd", "", "Working directory")
	procSpawn.Flags().Int("timeout", 0, "Timeout in seconds")
	procSpawn.Flags().String("as-user", "", "Run as user")
	procSpawn.Flags().Bool("as-root", false, "Run as root")
	procKill := &cobra.Command{Use: "kill <id|persistent-id> <process-id>", Short: "Send a signal to a process", Args: cobra.ExactArgs(2), RunE: runBrowsersProcessKill}
	procKill.Flags().String("signal", "TERM", "Signal to send (TERM, KILL, INT, HUP)")
	procStatus := &cobra.Command{Use: "status <id|persistent-id> <process-id>", Short: "Get process status", Args: cobra.ExactArgs(2), RunE: runBrowsersProcessStatus}
	procStdin := &cobra.Command{Use: "stdin <id|persistent-id> <process-id>", Short: "Write to process stdin (base64)", Args: cobra.ExactArgs(2), RunE: runBrowsersProcessStdin}
	procStdin.Flags().String("data-b64", "", "Base64-encoded data to write to stdin")
	_ = procStdin.MarkFlagRequired("data-b64")
	procStdoutStream := &cobra.Command{Use: "stdout-stream <id|persistent-id> <process-id>", Short: "Stream process stdout/stderr", Args: cobra.ExactArgs(2), RunE: runBrowsersProcessStdoutStream}
	procRoot.AddCommand(procExec, procSpawn, procKill, procStatus, procStdin, procStdoutStream)
	browsersCmd.AddCommand(procRoot)

	// fs
	fsRoot := &cobra.Command{Use: "fs", Short: "Browser filesystem operations"}
	fsNewDir := &cobra.Command{Use: "new-directory <id|persistent-id>", Short: "Create a new directory", Args: cobra.ExactArgs(1), RunE: runBrowsersFSNewDirectory}
	fsNewDir.Flags().String("path", "", "Absolute directory path to create")
	_ = fsNewDir.MarkFlagRequired("path")
	fsNewDir.Flags().String("mode", "", "Directory mode (octal string)")
	fsDelDir := &cobra.Command{Use: "delete-directory <id|persistent-id>", Short: "Delete a directory", Args: cobra.ExactArgs(1), RunE: runBrowsersFSDeleteDirectory}
	fsDelDir.Flags().String("path", "", "Absolute directory path to delete")
	_ = fsDelDir.MarkFlagRequired("path")
	fsDelFile := &cobra.Command{Use: "delete-file <id|persistent-id>", Short: "Delete a file", Args: cobra.ExactArgs(1), RunE: runBrowsersFSDeleteFile}
	fsDelFile.Flags().String("path", "", "Absolute file path to delete")
	_ = fsDelFile.MarkFlagRequired("path")
	fsDownloadZip := &cobra.Command{Use: "download-dir-zip <id|persistent-id>", Short: "Download a directory as zip", Args: cobra.ExactArgs(1), RunE: runBrowsersFSDownloadDirZip}
	fsDownloadZip.Flags().String("path", "", "Absolute directory path to download")
	_ = fsDownloadZip.MarkFlagRequired("path")
	fsDownloadZip.Flags().StringP("output", "o", "", "Output zip file path")
	fsFileInfo := &cobra.Command{Use: "file-info <id|persistent-id>", Short: "Get file or directory info", Args: cobra.ExactArgs(1), RunE: runBrowsersFSFileInfo}
	fsFileInfo.Flags().String("path", "", "Absolute file or directory path")
	_ = fsFileInfo.MarkFlagRequired("path")
	fsListFiles := &cobra.Command{Use: "list-files <id|persistent-id>", Short: "List files in a directory", Args: cobra.ExactArgs(1), RunE: runBrowsersFSListFiles}
	fsListFiles.Flags().String("path", "", "Absolute directory path")
	_ = fsListFiles.MarkFlagRequired("path")
	fsMove := &cobra.Command{Use: "move <id|persistent-id>", Short: "Move or rename a file or directory", Args: cobra.ExactArgs(1), RunE: runBrowsersFSMove}
	fsMove.Flags().String("src", "", "Absolute source path")
	fsMove.Flags().String("dest", "", "Absolute destination path")
	_ = fsMove.MarkFlagRequired("src")
	_ = fsMove.MarkFlagRequired("dest")
	fsReadFile := &cobra.Command{Use: "read-file <id|persistent-id>", Short: "Read a file", Args: cobra.ExactArgs(1), RunE: runBrowsersFSReadFile}
	fsReadFile.Flags().String("path", "", "Absolute file path")
	_ = fsReadFile.MarkFlagRequired("path")
	fsReadFile.Flags().StringP("output", "o", "", "Output file path (optional)")
	fsSetPerms := &cobra.Command{Use: "set-permissions <id|persistent-id>", Short: "Set file permissions or ownership", Args: cobra.ExactArgs(1), RunE: runBrowsersFSSetPermissions}
	fsSetPerms.Flags().String("path", "", "Absolute path")
	fsSetPerms.Flags().String("mode", "", "File mode bits (octal string)")
	_ = fsSetPerms.MarkFlagRequired("path")
	_ = fsSetPerms.MarkFlagRequired("mode")
	fsSetPerms.Flags().String("owner", "", "New owner username or UID")
	fsSetPerms.Flags().String("group", "", "New group name or GID")

	// fs upload
	fsUpload := &cobra.Command{Use: "upload <id|persistent-id>", Short: "Upload one or more files", Args: cobra.ExactArgs(1), RunE: runBrowsersFSUpload}
	fsUpload.Flags().StringSlice("file", []string{}, "Mapping local:remote (repeatable)")
	fsUpload.Flags().String("dest-dir", "", "Destination directory for uploads")
	fsUpload.Flags().StringSlice("paths", []string{}, "Local file paths to upload")

	// fs upload-zip
	fsUploadZip := &cobra.Command{Use: "upload-zip <id|persistent-id>", Short: "Upload a zip and extract it", Args: cobra.ExactArgs(1), RunE: runBrowsersFSUploadZip}
	fsUploadZip.Flags().String("zip", "", "Local zip file path")
	_ = fsUploadZip.MarkFlagRequired("zip")
	fsUploadZip.Flags().String("dest-dir", "", "Destination directory to extract to")
	_ = fsUploadZip.MarkFlagRequired("dest-dir")

	// fs write-file
	fsWriteFile := &cobra.Command{Use: "write-file <id|persistent-id>", Short: "Write a file from local data", Args: cobra.ExactArgs(1), RunE: runBrowsersFSWriteFile}
	fsWriteFile.Flags().String("path", "", "Destination absolute file path")
	_ = fsWriteFile.MarkFlagRequired("path")
	fsWriteFile.Flags().String("mode", "", "File mode (octal string)")
	fsWriteFile.Flags().String("source", "", "Local source file path")
	_ = fsWriteFile.MarkFlagRequired("source")

	fsRoot.AddCommand(fsNewDir, fsDelDir, fsDelFile, fsDownloadZip, fsFileInfo, fsListFiles, fsMove, fsReadFile, fsSetPerms, fsUpload, fsUploadZip, fsWriteFile)
	browsersCmd.AddCommand(fsRoot)

	// Add flags for create command
	browsersCreateCmd.Flags().StringP("persistent-id", "p", "", "Unique identifier for browser session persistence")
	browsersCreateCmd.Flags().BoolP("stealth", "s", false, "Launch browser in stealth mode to avoid detection")
	browsersCreateCmd.Flags().BoolP("headless", "H", false, "Launch browser without GUI access")
	browsersCreateCmd.Flags().IntP("timeout", "t", 60, "Timeout in seconds for the browser session")
	browsersCreateCmd.Flags().String("profile-id", "", "Profile ID to load into the browser session (mutually exclusive with --profile-name)")
	browsersCreateCmd.Flags().String("profile-name", "", "Profile name to load into the browser session (mutually exclusive with --profile-id)")
	browsersCreateCmd.Flags().Bool("save-changes", false, "If set, save changes back to the profile when the session ends")
	browsersCreateCmd.Flags().String("proxy-id", "", "Proxy ID to use for the browser session")

	// Add flags for delete command
	browsersDeleteCmd.Flags().BoolP("yes", "y", false, "Skip confirmation prompt")

	// no flags for view; it takes a single positional argument
}

func runBrowsersList(cmd *cobra.Command, args []string) error {
	client := getKernelClient(cmd)
	svc := client.Browsers
	b := BrowsersCmd{browsers: &svc}
	return b.List(cmd.Context())
}

func runBrowsersCreate(cmd *cobra.Command, args []string) error {
	client := getKernelClient(cmd)

	// Get flag values
	persistenceID, _ := cmd.Flags().GetString("persistent-id")
	stealthVal, _ := cmd.Flags().GetBool("stealth")
	headlessVal, _ := cmd.Flags().GetBool("headless")
	timeout, _ := cmd.Flags().GetInt("timeout")
	profileID, _ := cmd.Flags().GetString("profile-id")
	profileName, _ := cmd.Flags().GetString("profile-name")
	saveChanges, _ := cmd.Flags().GetBool("save-changes")
	proxyID, _ := cmd.Flags().GetString("proxy-id")

	in := BrowsersCreateInput{
		PersistenceID:      persistenceID,
		TimeoutSeconds:     timeout,
		Stealth:            BoolFlag{Set: cmd.Flags().Changed("stealth"), Value: stealthVal},
		Headless:           BoolFlag{Set: cmd.Flags().Changed("headless"), Value: headlessVal},
		ProfileID:          profileID,
		ProfileName:        profileName,
		ProfileSaveChanges: BoolFlag{Set: cmd.Flags().Changed("save-changes"), Value: saveChanges},
		ProxyID:            proxyID,
	}

	svc := client.Browsers
	b := BrowsersCmd{browsers: &svc}
	return b.Create(cmd.Context(), in)
}

func runBrowsersDelete(cmd *cobra.Command, args []string) error {
	client := getKernelClient(cmd)

	identifier := args[0]
	skipConfirm, _ := cmd.Flags().GetBool("yes")

	in := BrowsersDeleteInput{Identifier: identifier, SkipConfirm: skipConfirm}
	svc := client.Browsers
	b := BrowsersCmd{browsers: &svc}
	return b.Delete(cmd.Context(), in)
}

func runBrowsersView(cmd *cobra.Command, args []string) error {
	client := getKernelClient(cmd)

	identifier := args[0]

	in := BrowsersViewInput{Identifier: identifier}
	svc := client.Browsers
	b := BrowsersCmd{browsers: &svc}
	return b.View(cmd.Context(), in)
}

func runBrowsersLogsStream(cmd *cobra.Command, args []string) error {
	client := getKernelClient(cmd)
	svc := client.Browsers
	followVal, _ := cmd.Flags().GetBool("follow")
	source, _ := cmd.Flags().GetString("source")
	path, _ := cmd.Flags().GetString("path")
	supervisor, _ := cmd.Flags().GetString("supervisor-process")
	b := BrowsersCmd{browsers: &svc, logs: &svc.Logs}
	return b.LogsStream(cmd.Context(), BrowsersLogsStreamInput{
		Identifier:        args[0],
		Source:            source,
		Follow:            BoolFlag{Set: cmd.Flags().Changed("follow"), Value: followVal},
		Path:              path,
		SupervisorProcess: supervisor,
	})
}

func runBrowsersReplaysList(cmd *cobra.Command, args []string) error {
	client := getKernelClient(cmd)
	svc := client.Browsers
	b := BrowsersCmd{browsers: &svc, replays: &svc.Replays}
	return b.ReplaysList(cmd.Context(), BrowsersReplaysListInput{Identifier: args[0]})
}

func runBrowsersReplaysStart(cmd *cobra.Command, args []string) error {
	client := getKernelClient(cmd)
	svc := client.Browsers
	fr, _ := cmd.Flags().GetInt("framerate")
	md, _ := cmd.Flags().GetInt("max-duration")
	b := BrowsersCmd{browsers: &svc, replays: &svc.Replays}
	return b.ReplaysStart(cmd.Context(), BrowsersReplaysStartInput{Identifier: args[0], Framerate: fr, MaxDurationSeconds: md})
}

func runBrowsersReplaysStop(cmd *cobra.Command, args []string) error {
	client := getKernelClient(cmd)
	svc := client.Browsers
	b := BrowsersCmd{browsers: &svc, replays: &svc.Replays}
	return b.ReplaysStop(cmd.Context(), BrowsersReplaysStopInput{Identifier: args[0], ReplayID: args[1]})
}

func runBrowsersReplaysDownload(cmd *cobra.Command, args []string) error {
	client := getKernelClient(cmd)
	svc := client.Browsers
	out, _ := cmd.Flags().GetString("output")
	b := BrowsersCmd{browsers: &svc, replays: &svc.Replays}
	return b.ReplaysDownload(cmd.Context(), BrowsersReplaysDownloadInput{Identifier: args[0], ReplayID: args[1], Output: out})
}

func runBrowsersProcessExec(cmd *cobra.Command, args []string) error {
	client := getKernelClient(cmd)
	svc := client.Browsers
	command, _ := cmd.Flags().GetString("command")
	argv, _ := cmd.Flags().GetStringSlice("args")
	cwd, _ := cmd.Flags().GetString("cwd")
	timeout, _ := cmd.Flags().GetInt("timeout")
	asUser, _ := cmd.Flags().GetString("as-user")
	asRoot, _ := cmd.Flags().GetBool("as-root")
	if command == "" && len(args) > 1 {
		// Treat trailing args after identifier as a shell command
		shellCmd := strings.Join(args[1:], " ")
		command = "/bin/bash"
		argv = []string{"-c", shellCmd}
	}
	b := BrowsersCmd{browsers: &svc, process: &svc.Process}
	return b.ProcessExec(cmd.Context(), BrowsersProcessExecInput{Identifier: args[0], Command: command, Args: argv, Cwd: cwd, Timeout: timeout, AsUser: asUser, AsRoot: BoolFlag{Set: cmd.Flags().Changed("as-root"), Value: asRoot}})
}

func runBrowsersProcessSpawn(cmd *cobra.Command, args []string) error {
	client := getKernelClient(cmd)
	svc := client.Browsers
	command, _ := cmd.Flags().GetString("command")
	argv, _ := cmd.Flags().GetStringSlice("args")
	cwd, _ := cmd.Flags().GetString("cwd")
	timeout, _ := cmd.Flags().GetInt("timeout")
	asUser, _ := cmd.Flags().GetString("as-user")
	asRoot, _ := cmd.Flags().GetBool("as-root")
	if command == "" && len(args) > 1 {
		shellCmd := strings.Join(args[1:], " ")
		command = "/bin/bash"
		argv = []string{"-c", shellCmd}
	}
	b := BrowsersCmd{browsers: &svc, process: &svc.Process}
	return b.ProcessSpawn(cmd.Context(), BrowsersProcessSpawnInput{Identifier: args[0], Command: command, Args: argv, Cwd: cwd, Timeout: timeout, AsUser: asUser, AsRoot: BoolFlag{Set: cmd.Flags().Changed("as-root"), Value: asRoot}})
}

func runBrowsersProcessKill(cmd *cobra.Command, args []string) error {
	client := getKernelClient(cmd)
	svc := client.Browsers
	signal, _ := cmd.Flags().GetString("signal")
	b := BrowsersCmd{browsers: &svc, process: &svc.Process}
	return b.ProcessKill(cmd.Context(), BrowsersProcessKillInput{Identifier: args[0], ProcessID: args[1], Signal: signal})
}

func runBrowsersProcessStatus(cmd *cobra.Command, args []string) error {
	client := getKernelClient(cmd)
	svc := client.Browsers
	b := BrowsersCmd{browsers: &svc, process: &svc.Process}
	return b.ProcessStatus(cmd.Context(), BrowsersProcessStatusInput{Identifier: args[0], ProcessID: args[1]})
}

func runBrowsersProcessStdin(cmd *cobra.Command, args []string) error {
	client := getKernelClient(cmd)
	svc := client.Browsers
	data, _ := cmd.Flags().GetString("data-b64")
	b := BrowsersCmd{browsers: &svc, process: &svc.Process}
	return b.ProcessStdin(cmd.Context(), BrowsersProcessStdinInput{Identifier: args[0], ProcessID: args[1], DataB64: data})
}

func runBrowsersProcessStdoutStream(cmd *cobra.Command, args []string) error {
	client := getKernelClient(cmd)
	svc := client.Browsers
	b := BrowsersCmd{browsers: &svc, process: &svc.Process}
	return b.ProcessStdoutStream(cmd.Context(), BrowsersProcessStdoutStreamInput{Identifier: args[0], ProcessID: args[1]})
}

func runBrowsersFSNewDirectory(cmd *cobra.Command, args []string) error {
	client := getKernelClient(cmd)
	svc := client.Browsers
	path, _ := cmd.Flags().GetString("path")
	mode, _ := cmd.Flags().GetString("mode")
	b := BrowsersCmd{browsers: &svc, fs: &svc.Fs}
	return b.FSNewDirectory(cmd.Context(), BrowsersFSNewDirInput{Identifier: args[0], Path: path, Mode: mode})
}

func runBrowsersFSDeleteDirectory(cmd *cobra.Command, args []string) error {
	client := getKernelClient(cmd)
	svc := client.Browsers
	path, _ := cmd.Flags().GetString("path")
	b := BrowsersCmd{browsers: &svc, fs: &svc.Fs}
	return b.FSDeleteDirectory(cmd.Context(), BrowsersFSDeleteDirInput{Identifier: args[0], Path: path})
}

func runBrowsersFSDeleteFile(cmd *cobra.Command, args []string) error {
	client := getKernelClient(cmd)
	svc := client.Browsers
	path, _ := cmd.Flags().GetString("path")
	b := BrowsersCmd{browsers: &svc, fs: &svc.Fs}
	return b.FSDeleteFile(cmd.Context(), BrowsersFSDeleteFileInput{Identifier: args[0], Path: path})
}

func runBrowsersFSDownloadDirZip(cmd *cobra.Command, args []string) error {
	client := getKernelClient(cmd)
	svc := client.Browsers
	path, _ := cmd.Flags().GetString("path")
	out, _ := cmd.Flags().GetString("output")
	b := BrowsersCmd{browsers: &svc, fs: &svc.Fs}
	return b.FSDownloadDirZip(cmd.Context(), BrowsersFSDownloadDirZipInput{Identifier: args[0], Path: path, Output: out})
}

func runBrowsersFSFileInfo(cmd *cobra.Command, args []string) error {
	client := getKernelClient(cmd)
	svc := client.Browsers
	path, _ := cmd.Flags().GetString("path")
	b := BrowsersCmd{browsers: &svc, fs: &svc.Fs}
	return b.FSFileInfo(cmd.Context(), BrowsersFSFileInfoInput{Identifier: args[0], Path: path})
}

func runBrowsersFSListFiles(cmd *cobra.Command, args []string) error {
	client := getKernelClient(cmd)
	svc := client.Browsers
	path, _ := cmd.Flags().GetString("path")
	b := BrowsersCmd{browsers: &svc, fs: &svc.Fs}
	return b.FSListFiles(cmd.Context(), BrowsersFSListFilesInput{Identifier: args[0], Path: path})
}

func runBrowsersFSMove(cmd *cobra.Command, args []string) error {
	client := getKernelClient(cmd)
	svc := client.Browsers
	src, _ := cmd.Flags().GetString("src")
	dest, _ := cmd.Flags().GetString("dest")
	b := BrowsersCmd{browsers: &svc, fs: &svc.Fs}
	return b.FSMove(cmd.Context(), BrowsersFSMoveInput{Identifier: args[0], SrcPath: src, DestPath: dest})
}

func runBrowsersFSReadFile(cmd *cobra.Command, args []string) error {
	client := getKernelClient(cmd)
	svc := client.Browsers
	path, _ := cmd.Flags().GetString("path")
	out, _ := cmd.Flags().GetString("output")
	b := BrowsersCmd{browsers: &svc, fs: &svc.Fs}
	return b.FSReadFile(cmd.Context(), BrowsersFSReadFileInput{Identifier: args[0], Path: path, Output: out})
}

func runBrowsersFSSetPermissions(cmd *cobra.Command, args []string) error {
	client := getKernelClient(cmd)
	svc := client.Browsers
	path, _ := cmd.Flags().GetString("path")
	mode, _ := cmd.Flags().GetString("mode")
	owner, _ := cmd.Flags().GetString("owner")
	group, _ := cmd.Flags().GetString("group")
	b := BrowsersCmd{browsers: &svc, fs: &svc.Fs}
	return b.FSSetPermissions(cmd.Context(), BrowsersFSSetPermsInput{Identifier: args[0], Path: path, Mode: mode, Owner: owner, Group: group})
}

func runBrowsersFSUpload(cmd *cobra.Command, args []string) error {
	client := getKernelClient(cmd)
	svc := client.Browsers
	fileMaps, _ := cmd.Flags().GetStringSlice("file")
	destDir, _ := cmd.Flags().GetString("dest-dir")
	paths, _ := cmd.Flags().GetStringSlice("paths")
	var mappings []struct {
		Local string
		Dest  string
	}
	for _, m := range fileMaps {
		// format: local:remote
		parts := strings.SplitN(m, ":", 2)
		if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
			pterm.Error.Printf("invalid --file mapping: %s\n", m)
			return nil
		}
		mappings = append(mappings, struct {
			Local string
			Dest  string
		}{Local: parts[0], Dest: parts[1]})
	}
	b := BrowsersCmd{browsers: &svc, fs: &svc.Fs}
	return b.FSUpload(cmd.Context(), BrowsersFSUploadInput{Identifier: args[0], Mappings: mappings, DestDir: destDir, Paths: paths})
}

func runBrowsersFSUploadZip(cmd *cobra.Command, args []string) error {
	client := getKernelClient(cmd)
	svc := client.Browsers
	zipPath, _ := cmd.Flags().GetString("zip")
	destDir, _ := cmd.Flags().GetString("dest-dir")
	b := BrowsersCmd{browsers: &svc, fs: &svc.Fs}
	return b.FSUploadZip(cmd.Context(), BrowsersFSUploadZipInput{Identifier: args[0], ZipPath: zipPath, DestDir: destDir})
}

func runBrowsersFSWriteFile(cmd *cobra.Command, args []string) error {
	client := getKernelClient(cmd)
	svc := client.Browsers
	path, _ := cmd.Flags().GetString("path")
	mode, _ := cmd.Flags().GetString("mode")
	input, _ := cmd.Flags().GetString("source")
	b := BrowsersCmd{browsers: &svc, fs: &svc.Fs}
	return b.FSWriteFile(cmd.Context(), BrowsersFSWriteFileInput{Identifier: args[0], DestPath: path, Mode: mode, SourcePath: input})
}

func truncateURL(url string, maxLen int) string {
	if len(url) <= maxLen {
		return url
	}
	return url[:maxLen-3] + "..."
}

// resolveBrowserByIdentifier finds a browser by session ID or persistent ID.
func (b BrowsersCmd) resolveBrowserByIdentifier(ctx context.Context, identifier string) (*kernel.BrowserListResponse, error) {
	browsers, err := b.browsers.List(ctx)
	if err != nil {
		return nil, err
	}
	if browsers == nil {
		return nil, nil
	}
	for _, br := range *browsers {
		if br.SessionID == identifier || br.Persistence.ID == identifier {
			bCopy := br
			return &bCopy, nil
		}
	}
	return nil, nil
}
