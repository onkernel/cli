package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/onkernel/kernel-go-sdk"
	"github.com/onkernel/kernel-go-sdk/option"
	"github.com/onkernel/kernel-go-sdk/packages/ssestream"
	"github.com/onkernel/kernel-go-sdk/shared"
	"github.com/pterm/pterm"
)

// outBuf captures pterm output during tests.
var outBuf bytes.Buffer

// setupStdoutCapture sets pterm's default output to an in-memory buffer.
func setupStdoutCapture(t *testing.T) {
	outBuf.Reset()
	pterm.SetDefaultOutput(&outBuf)
	// Prefix printers capture writer at init; set explicitly
	pterm.Info.Writer = &outBuf
	pterm.Error.Writer = &outBuf
	pterm.Success.Writer = &outBuf
	pterm.Warning.Writer = &outBuf
	pterm.Debug.Writer = &outBuf
	pterm.Fatal.Writer = &outBuf
	// Ensure tables render to our buffer
	pterm.DefaultTable = *pterm.DefaultTable.WithWriter(&outBuf)
	// Restore after test completes
	t.Cleanup(func() {
		pterm.SetDefaultOutput(os.Stdout)
		pterm.Info.Writer = os.Stdout
		pterm.Error.Writer = os.Stdout
		pterm.Success.Writer = os.Stdout
		pterm.Warning.Writer = os.Stdout
		pterm.Debug.Writer = os.Stdout
		pterm.Fatal.Writer = os.Stdout
		pterm.DefaultTable = *pterm.DefaultTable.WithWriter(os.Stdout)
		outBuf.Reset()
	})
}

// FakeBrowsersService is a configurable fake implementing BrowsersService.
type FakeBrowsersService struct {
	ListFunc       func(ctx context.Context, opts ...option.RequestOption) (*[]kernel.BrowserListResponse, error)
	NewFunc        func(ctx context.Context, body kernel.BrowserNewParams, opts ...option.RequestOption) (*kernel.BrowserNewResponse, error)
	DeleteFunc     func(ctx context.Context, body kernel.BrowserDeleteParams, opts ...option.RequestOption) error
	DeleteByIDFunc func(ctx context.Context, id string, opts ...option.RequestOption) error
}

func (f *FakeBrowsersService) List(ctx context.Context, opts ...option.RequestOption) (*[]kernel.BrowserListResponse, error) {
	if f.ListFunc != nil {
		return f.ListFunc(ctx, opts...)
	}
	return &[]kernel.BrowserListResponse{}, nil
}

func (f *FakeBrowsersService) New(ctx context.Context, body kernel.BrowserNewParams, opts ...option.RequestOption) (*kernel.BrowserNewResponse, error) {
	if f.NewFunc != nil {
		return f.NewFunc(ctx, body, opts...)
	}
	return &kernel.BrowserNewResponse{}, nil
}

func (f *FakeBrowsersService) Delete(ctx context.Context, body kernel.BrowserDeleteParams, opts ...option.RequestOption) error {
	if f.DeleteFunc != nil {
		return f.DeleteFunc(ctx, body, opts...)
	}
	return nil
}

func (f *FakeBrowsersService) DeleteByID(ctx context.Context, id string, opts ...option.RequestOption) error {
	if f.DeleteByIDFunc != nil {
		return f.DeleteByIDFunc(ctx, id, opts...)
	}
	return nil
}

func TestBrowsersList_PrintsEmptyMessage(t *testing.T) {
	setupStdoutCapture(t)

	fake := &FakeBrowsersService{
		ListFunc: func(ctx context.Context, opts ...option.RequestOption) (*[]kernel.BrowserListResponse, error) {
			empty := []kernel.BrowserListResponse{}
			return &empty, nil
		},
	}
	b := BrowsersCmd{browsers: fake}
	_ = b.List(context.Background())

	out := outBuf.String()
	if !strings.Contains(out, "No running or persistent browsers found") {
		t.Fatalf("expected empty message, got: %s", out)
	}
}

func TestBrowsersList_PrintsTableWithRows(t *testing.T) {
	setupStdoutCapture(t)

	created := time.Date(2025, 1, 2, 3, 4, 5, 0, time.UTC)
	rows := []kernel.BrowserListResponse{
		{
			SessionID:          "sess-1",
			CdpWsURL:           "ws://cdp-1",
			BrowserLiveViewURL: "http://view-1",
			CreatedAt:          created,
			Persistence:        kernel.BrowserPersistence{ID: "pid-1"},
		},
		{
			SessionID:          "sess-2",
			CdpWsURL:           "ws://cdp-2",
			BrowserLiveViewURL: "",
			CreatedAt:          created,
			Persistence:        kernel.BrowserPersistence{ID: ""},
		},
	}

	fake := &FakeBrowsersService{
		ListFunc: func(ctx context.Context, opts ...option.RequestOption) (*[]kernel.BrowserListResponse, error) {
			return &rows, nil
		},
	}
	b := BrowsersCmd{browsers: fake}
	_ = b.List(context.Background())

	out := outBuf.String()
	if !strings.Contains(out, "sess-1") || !strings.Contains(out, "sess-2") {
		t.Fatalf("expected session IDs in output, got: %s", out)
	}
	if !strings.Contains(out, "pid-1") {
		t.Fatalf("expected persistent ID in output, got: %s", out)
	}
}

func TestBrowsersList_PrintsErrorOnFailure(t *testing.T) {
	setupStdoutCapture(t)

	fake := &FakeBrowsersService{
		ListFunc: func(ctx context.Context, opts ...option.RequestOption) (*[]kernel.BrowserListResponse, error) {
			return nil, errors.New("list failed")
		},
	}
	b := BrowsersCmd{browsers: fake}
	_ = b.List(context.Background())

	out := outBuf.String()
	if !strings.Contains(out, "Failed to list browsers: list failed") {
		t.Fatalf("expected error message, got: %s", out)
	}
}

func TestBrowsersCreate_PrintsResponse(t *testing.T) {
	setupStdoutCapture(t)

	fake := &FakeBrowsersService{
		NewFunc: func(ctx context.Context, body kernel.BrowserNewParams, opts ...option.RequestOption) (*kernel.BrowserNewResponse, error) {
			resp := &kernel.BrowserNewResponse{
				SessionID:          "sess-new",
				CdpWsURL:           "ws://cdp-new",
				BrowserLiveViewURL: "http://view-new",
				Persistence:        kernel.BrowserPersistence{ID: "pid-new"},
			}
			return resp, nil
		},
	}

	b := BrowsersCmd{browsers: fake}
	in := BrowsersCreateInput{
		PersistenceID:  "pid-new",
		TimeoutSeconds: 120,
		Stealth:        BoolFlag{Set: true, Value: true},
		Headless:       BoolFlag{Set: true, Value: false},
	}
	_ = b.Create(context.Background(), in)

	out := outBuf.String()
	if !strings.Contains(out, "Session ID") || !strings.Contains(out, "sess-new") {
		t.Fatalf("expected session details, got: %s", out)
	}
	if !strings.Contains(out, "CDP WebSocket URL") || !strings.Contains(out, "ws://cdp-new") {
		t.Fatalf("expected cdp url, got: %s", out)
	}
	if !strings.Contains(out, "Live View URL") || !strings.Contains(out, "http://view-new") {
		t.Fatalf("expected live view url, got: %s", out)
	}
	if !strings.Contains(out, "Persistent ID") || !strings.Contains(out, "pid-new") {
		t.Fatalf("expected persistent id, got: %s", out)
	}
}

func TestBrowsersCreate_PrintsErrorOnFailure(t *testing.T) {
	setupStdoutCapture(t)

	fake := &FakeBrowsersService{
		NewFunc: func(ctx context.Context, body kernel.BrowserNewParams, opts ...option.RequestOption) (*kernel.BrowserNewResponse, error) {
			return nil, errors.New("create failed")
		},
	}
	b := BrowsersCmd{browsers: fake}
	_ = b.Create(context.Background(), BrowsersCreateInput{})

	out := outBuf.String()
	if !strings.Contains(out, "Failed to create browser: create failed") {
		t.Fatalf("expected create error message, got: %s", out)
	}
}

func TestBrowsersDelete_SkipConfirm_Success(t *testing.T) {
	setupStdoutCapture(t)

	fake := &FakeBrowsersService{
		DeleteFunc: func(ctx context.Context, body kernel.BrowserDeleteParams, opts ...option.RequestOption) error {
			return nil
		},
		DeleteByIDFunc: func(ctx context.Context, id string, opts ...option.RequestOption) error {
			return nil
		},
	}
	b := BrowsersCmd{browsers: fake}
	_ = b.Delete(context.Background(), BrowsersDeleteInput{Identifier: "any", SkipConfirm: true})

	out := outBuf.String()
	if !strings.Contains(out, "Successfully deleted (or already absent) browser: any") {
		t.Fatalf("expected success message, got: %s", out)
	}
}

func TestBrowsersDelete_SkipConfirm_Failure(t *testing.T) {
	setupStdoutCapture(t)

	fake := &FakeBrowsersService{
		DeleteFunc: func(ctx context.Context, body kernel.BrowserDeleteParams, opts ...option.RequestOption) error {
			return errors.New("left failed")
		},
		DeleteByIDFunc: func(ctx context.Context, id string, opts ...option.RequestOption) error {
			return errors.New("right failed")
		},
	}
	b := BrowsersCmd{browsers: fake}
	_ = b.Delete(context.Background(), BrowsersDeleteInput{Identifier: "any", SkipConfirm: true})

	out := outBuf.String()
	if !strings.Contains(out, "Failed to delete browser: right failed") && !strings.Contains(out, "Failed to delete browser: left failed") {
		t.Fatalf("expected failure message, got: %s", out)
	}
}

func TestBrowsersDelete_WithConfirm_NotFound(t *testing.T) {
	setupStdoutCapture(t)

	list := []kernel.BrowserListResponse{{SessionID: "s-1", Persistence: kernel.BrowserPersistence{ID: "p-1"}}}
	fake := &FakeBrowsersService{
		ListFunc: func(ctx context.Context, opts ...option.RequestOption) (*[]kernel.BrowserListResponse, error) {
			return &list, nil
		},
	}
	b := BrowsersCmd{browsers: fake}
	_ = b.Delete(context.Background(), BrowsersDeleteInput{Identifier: "missing", SkipConfirm: false})

	out := outBuf.String()
	if !strings.Contains(out, "Browser 'missing' not found") {
		t.Fatalf("expected not found message, got: %s", out)
	}
}

func TestBrowsersView_ByID_PrintsURL(t *testing.T) {
	setupStdoutCapture(t)

	list := []kernel.BrowserListResponse{{
		SessionID:          "abc",
		BrowserLiveViewURL: "http://live-url",
	}}
	fake := &FakeBrowsersService{
		ListFunc: func(ctx context.Context, opts ...option.RequestOption) (*[]kernel.BrowserListResponse, error) {
			return &list, nil
		},
	}
	b := BrowsersCmd{browsers: fake}
	_ = b.View(context.Background(), BrowsersViewInput{Identifier: "abc"})

	out := outBuf.String()
	if !strings.Contains(out, "http://live-url") {
		t.Fatalf("expected live view url, got: %s", out)
	}
}

func TestBrowsersView_NotFound_ByEither(t *testing.T) {
	setupStdoutCapture(t)

	list := []kernel.BrowserListResponse{{SessionID: "abc", Persistence: kernel.BrowserPersistence{ID: "pid-xyz"}}}
	fake := &FakeBrowsersService{
		ListFunc: func(ctx context.Context, opts ...option.RequestOption) (*[]kernel.BrowserListResponse, error) {
			return &list, nil
		},
	}
	b := BrowsersCmd{browsers: fake}
	_ = b.View(context.Background(), BrowsersViewInput{Identifier: "missing"})

	out := outBuf.String()
	if !strings.Contains(out, "Browser 'missing' not found") {
		t.Fatalf("expected not found message, got: %s", out)
	}
}

func TestBrowsersView_PrintsErrorOnListFailure(t *testing.T) {
	setupStdoutCapture(t)

	fake := &FakeBrowsersService{
		ListFunc: func(ctx context.Context, opts ...option.RequestOption) (*[]kernel.BrowserListResponse, error) {
			return nil, errors.New("list error")
		},
	}
	b := BrowsersCmd{browsers: fake}
	_ = b.View(context.Background(), BrowsersViewInput{Identifier: "any"})

	out := outBuf.String()
	if !strings.Contains(out, "Failed to list browsers: list error") {
		t.Fatalf("expected error message, got: %s", out)
	}
}

// --- Fakes for sub-services ---

type FakeReplaysService struct {
	ListFunc     func(ctx context.Context, id string, opts ...option.RequestOption) (*[]kernel.BrowserReplayListResponse, error)
	DownloadFunc func(ctx context.Context, replayID string, query kernel.BrowserReplayDownloadParams, opts ...option.RequestOption) (*http.Response, error)
	StartFunc    func(ctx context.Context, id string, body kernel.BrowserReplayStartParams, opts ...option.RequestOption) (*kernel.BrowserReplayStartResponse, error)
	StopFunc     func(ctx context.Context, replayID string, body kernel.BrowserReplayStopParams, opts ...option.RequestOption) error
}

func (f *FakeReplaysService) List(ctx context.Context, id string, opts ...option.RequestOption) (*[]kernel.BrowserReplayListResponse, error) {
	if f.ListFunc != nil {
		return f.ListFunc(ctx, id, opts...)
	}
	empty := []kernel.BrowserReplayListResponse{}
	return &empty, nil
}
func (f *FakeReplaysService) Download(ctx context.Context, replayID string, query kernel.BrowserReplayDownloadParams, opts ...option.RequestOption) (*http.Response, error) {
	if f.DownloadFunc != nil {
		return f.DownloadFunc(ctx, replayID, query, opts...)
	}
	return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader("")), Header: http.Header{}}, nil
}
func (f *FakeReplaysService) Start(ctx context.Context, id string, body kernel.BrowserReplayStartParams, opts ...option.RequestOption) (*kernel.BrowserReplayStartResponse, error) {
	if f.StartFunc != nil {
		return f.StartFunc(ctx, id, body, opts...)
	}
	return &kernel.BrowserReplayStartResponse{ReplayID: "r-1", ReplayViewURL: "http://view", StartedAt: time.Now()}, nil
}
func (f *FakeReplaysService) Stop(ctx context.Context, replayID string, body kernel.BrowserReplayStopParams, opts ...option.RequestOption) error {
	if f.StopFunc != nil {
		return f.StopFunc(ctx, replayID, body, opts...)
	}
	return nil
}

type FakeFSService struct {
	NewDirectoryFunc       func(ctx context.Context, id string, body kernel.BrowserFNewDirectoryParams, opts ...option.RequestOption) error
	DeleteDirectoryFunc    func(ctx context.Context, id string, body kernel.BrowserFDeleteDirectoryParams, opts ...option.RequestOption) error
	DeleteFileFunc         func(ctx context.Context, id string, body kernel.BrowserFDeleteFileParams, opts ...option.RequestOption) error
	DownloadDirZipFunc     func(ctx context.Context, id string, query kernel.BrowserFDownloadDirZipParams, opts ...option.RequestOption) (*http.Response, error)
	FileInfoFunc           func(ctx context.Context, id string, query kernel.BrowserFFileInfoParams, opts ...option.RequestOption) (*kernel.BrowserFFileInfoResponse, error)
	ListFilesFunc          func(ctx context.Context, id string, query kernel.BrowserFListFilesParams, opts ...option.RequestOption) (*[]kernel.BrowserFListFilesResponse, error)
	MoveFunc               func(ctx context.Context, id string, body kernel.BrowserFMoveParams, opts ...option.RequestOption) error
	ReadFileFunc           func(ctx context.Context, id string, query kernel.BrowserFReadFileParams, opts ...option.RequestOption) (*http.Response, error)
	SetFilePermissionsFunc func(ctx context.Context, id string, body kernel.BrowserFSetFilePermissionsParams, opts ...option.RequestOption) error
	UploadFunc             func(ctx context.Context, id string, body kernel.BrowserFUploadParams, opts ...option.RequestOption) error
	UploadZipFunc          func(ctx context.Context, id string, body kernel.BrowserFUploadZipParams, opts ...option.RequestOption) error
	WriteFileFunc          func(ctx context.Context, id string, contents io.Reader, body kernel.BrowserFWriteFileParams, opts ...option.RequestOption) error
}

func (f *FakeFSService) NewDirectory(ctx context.Context, id string, body kernel.BrowserFNewDirectoryParams, opts ...option.RequestOption) error {
	if f.NewDirectoryFunc != nil {
		return f.NewDirectoryFunc(ctx, id, body, opts...)
	}
	return nil
}
func (f *FakeFSService) DeleteDirectory(ctx context.Context, id string, body kernel.BrowserFDeleteDirectoryParams, opts ...option.RequestOption) error {
	if f.DeleteDirectoryFunc != nil {
		return f.DeleteDirectoryFunc(ctx, id, body, opts...)
	}
	return nil
}
func (f *FakeFSService) DeleteFile(ctx context.Context, id string, body kernel.BrowserFDeleteFileParams, opts ...option.RequestOption) error {
	if f.DeleteFileFunc != nil {
		return f.DeleteFileFunc(ctx, id, body, opts...)
	}
	return nil
}
func (f *FakeFSService) DownloadDirZip(ctx context.Context, id string, query kernel.BrowserFDownloadDirZipParams, opts ...option.RequestOption) (*http.Response, error) {
	if f.DownloadDirZipFunc != nil {
		return f.DownloadDirZipFunc(ctx, id, query, opts...)
	}
	return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": []string{"application/zip"}}, Body: io.NopCloser(strings.NewReader("zip"))}, nil
}
func (f *FakeFSService) FileInfo(ctx context.Context, id string, query kernel.BrowserFFileInfoParams, opts ...option.RequestOption) (*kernel.BrowserFFileInfoResponse, error) {
	if f.FileInfoFunc != nil {
		return f.FileInfoFunc(ctx, id, query, opts...)
	}
	return &kernel.BrowserFFileInfoResponse{Path: query.Path, Name: "name", Mode: "-rw-r--r--", IsDir: false, SizeBytes: 5, ModTime: time.Unix(0, 0)}, nil
}
func (f *FakeFSService) ListFiles(ctx context.Context, id string, query kernel.BrowserFListFilesParams, opts ...option.RequestOption) (*[]kernel.BrowserFListFilesResponse, error) {
	if f.ListFilesFunc != nil {
		return f.ListFilesFunc(ctx, id, query, opts...)
	}
	files := []kernel.BrowserFListFilesResponse{{Name: "f1", Path: "/f1", Mode: "-rw-r--r--", SizeBytes: 1, ModTime: time.Unix(0, 0)}}
	return &files, nil
}
func (f *FakeFSService) Move(ctx context.Context, id string, body kernel.BrowserFMoveParams, opts ...option.RequestOption) error {
	if f.MoveFunc != nil {
		return f.MoveFunc(ctx, id, body, opts...)
	}
	return nil
}
func (f *FakeFSService) ReadFile(ctx context.Context, id string, query kernel.BrowserFReadFileParams, opts ...option.RequestOption) (*http.Response, error) {
	if f.ReadFileFunc != nil {
		return f.ReadFileFunc(ctx, id, query, opts...)
	}
	return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": []string{"application/octet-stream"}}, Body: io.NopCloser(strings.NewReader("content"))}, nil
}
func (f *FakeFSService) SetFilePermissions(ctx context.Context, id string, body kernel.BrowserFSetFilePermissionsParams, opts ...option.RequestOption) error {
	if f.SetFilePermissionsFunc != nil {
		return f.SetFilePermissionsFunc(ctx, id, body, opts...)
	}
	return nil
}
func (f *FakeFSService) Upload(ctx context.Context, id string, body kernel.BrowserFUploadParams, opts ...option.RequestOption) error {
	if f.UploadFunc != nil {
		return f.UploadFunc(ctx, id, body, opts...)
	}
	return nil
}
func (f *FakeFSService) UploadZip(ctx context.Context, id string, body kernel.BrowserFUploadZipParams, opts ...option.RequestOption) error {
	if f.UploadZipFunc != nil {
		return f.UploadZipFunc(ctx, id, body, opts...)
	}
	return nil
}
func (f *FakeFSService) WriteFile(ctx context.Context, id string, contents io.Reader, body kernel.BrowserFWriteFileParams, opts ...option.RequestOption) error {
	if f.WriteFileFunc != nil {
		return f.WriteFileFunc(ctx, id, contents, body, opts...)
	}
	return nil
}

type FakeProcessService struct {
	ExecFunc         func(ctx context.Context, id string, body kernel.BrowserProcessExecParams, opts ...option.RequestOption) (*kernel.BrowserProcessExecResponse, error)
	KillFunc         func(ctx context.Context, processID string, params kernel.BrowserProcessKillParams, opts ...option.RequestOption) (*kernel.BrowserProcessKillResponse, error)
	SpawnFunc        func(ctx context.Context, id string, body kernel.BrowserProcessSpawnParams, opts ...option.RequestOption) (*kernel.BrowserProcessSpawnResponse, error)
	StatusFunc       func(ctx context.Context, processID string, query kernel.BrowserProcessStatusParams, opts ...option.RequestOption) (*kernel.BrowserProcessStatusResponse, error)
	StdinFunc        func(ctx context.Context, processID string, params kernel.BrowserProcessStdinParams, opts ...option.RequestOption) (*kernel.BrowserProcessStdinResponse, error)
	StdoutStreamFunc func(ctx context.Context, processID string, query kernel.BrowserProcessStdoutStreamParams, opts ...option.RequestOption) *ssestream.Stream[kernel.BrowserProcessStdoutStreamResponse]
}

func (f *FakeProcessService) Exec(ctx context.Context, id string, body kernel.BrowserProcessExecParams, opts ...option.RequestOption) (*kernel.BrowserProcessExecResponse, error) {
	if f.ExecFunc != nil {
		return f.ExecFunc(ctx, id, body, opts...)
	}
	return &kernel.BrowserProcessExecResponse{ExitCode: 0, DurationMs: 10}, nil
}
func (f *FakeProcessService) Kill(ctx context.Context, processID string, params kernel.BrowserProcessKillParams, opts ...option.RequestOption) (*kernel.BrowserProcessKillResponse, error) {
	if f.KillFunc != nil {
		return f.KillFunc(ctx, processID, params, opts...)
	}
	return &kernel.BrowserProcessKillResponse{Ok: true}, nil
}
func (f *FakeProcessService) Spawn(ctx context.Context, id string, body kernel.BrowserProcessSpawnParams, opts ...option.RequestOption) (*kernel.BrowserProcessSpawnResponse, error) {
	if f.SpawnFunc != nil {
		return f.SpawnFunc(ctx, id, body, opts...)
	}
	return &kernel.BrowserProcessSpawnResponse{ProcessID: "proc-1", Pid: 123, StartedAt: time.Now()}, nil
}
func (f *FakeProcessService) Status(ctx context.Context, processID string, query kernel.BrowserProcessStatusParams, opts ...option.RequestOption) (*kernel.BrowserProcessStatusResponse, error) {
	if f.StatusFunc != nil {
		return f.StatusFunc(ctx, processID, query, opts...)
	}
	return &kernel.BrowserProcessStatusResponse{State: kernel.BrowserProcessStatusResponseStateRunning, CPUPct: 1.5, MemBytes: 2048, ExitCode: 0}, nil
}
func (f *FakeProcessService) Stdin(ctx context.Context, processID string, params kernel.BrowserProcessStdinParams, opts ...option.RequestOption) (*kernel.BrowserProcessStdinResponse, error) {
	if f.StdinFunc != nil {
		return f.StdinFunc(ctx, processID, params, opts...)
	}
	return &kernel.BrowserProcessStdinResponse{WrittenBytes: int64(len(params.DataB64))}, nil
}
func (f *FakeProcessService) StdoutStreamStreaming(ctx context.Context, processID string, query kernel.BrowserProcessStdoutStreamParams, opts ...option.RequestOption) *ssestream.Stream[kernel.BrowserProcessStdoutStreamResponse] {
	if f.StdoutStreamFunc != nil {
		return f.StdoutStreamFunc(ctx, processID, query, opts...)
	}
	return makeStream([]kernel.BrowserProcessStdoutStreamResponse{{Stream: kernel.BrowserProcessStdoutStreamResponseStreamStdout, DataB64: "aGVsbG8=", Event: ""}, {Event: "exit", ExitCode: 0}})
}

type FakeLogService struct {
	StreamFunc func(ctx context.Context, id string, query kernel.BrowserLogStreamParams, opts ...option.RequestOption) *ssestream.Stream[shared.LogEvent]
}

func (f *FakeLogService) StreamStreaming(ctx context.Context, id string, query kernel.BrowserLogStreamParams, opts ...option.RequestOption) *ssestream.Stream[shared.LogEvent] {
	if f.StreamFunc != nil {
		return f.StreamFunc(ctx, id, query, opts...)
	}
	now := time.Now()
	return makeStream([]shared.LogEvent{{Message: "m1", Timestamp: now}, {Message: "m2", Timestamp: now}})
}

// --- Helpers for SSE streams ---

type testDecoder struct {
	data [][]byte
	idx  int
}

func (d *testDecoder) Event() ssestream.Event { return ssestream.Event{Data: d.data[d.idx-1]} }
func (d *testDecoder) Next() bool {
	if d.idx >= len(d.data) {
		return false
	}
	d.idx++
	return true
}
func (d *testDecoder) Close() error { return nil }
func (d *testDecoder) Err() error   { return nil }

func makeStream[T any](vals []T) *ssestream.Stream[T] {
	var events [][]byte
	for _, v := range vals {
		b, _ := json.Marshal(v)
		events = append(events, b)
	}
	return ssestream.NewStream[T](&testDecoder{data: events}, nil)
}

// --- Tests for Logs ---

func TestBrowsersLogsStream_PrintsEvents(t *testing.T) {
	setupStdoutCapture(t)
	fakeBrowsers := &FakeBrowsersService{ListFunc: func(ctx context.Context, opts ...option.RequestOption) (*[]kernel.BrowserListResponse, error) {
		rows := []kernel.BrowserListResponse{{SessionID: "id"}}
		return &rows, nil
	}}
	b := BrowsersCmd{browsers: fakeBrowsers, logs: &FakeLogService{}}
	_ = b.LogsStream(context.Background(), BrowsersLogsStreamInput{Identifier: "id", Source: string(kernel.BrowserLogStreamParamsSourcePath), Follow: BoolFlag{Set: true, Value: true}, Path: "/var/log.txt"})
	out := outBuf.String()
	if !strings.Contains(out, "m1") || !strings.Contains(out, "m2") {
		t.Fatalf("expected log messages, got: %s", out)
	}
}

// --- Tests for Replays ---

func TestBrowsersReplaysList_PrintsRows(t *testing.T) {
	setupStdoutCapture(t)
	created := time.Unix(0, 0)
	replays := []kernel.BrowserReplayListResponse{{ReplayID: "r1", StartedAt: created, FinishedAt: created, ReplayViewURL: "http://v"}}
	fake := &FakeReplaysService{ListFunc: func(ctx context.Context, id string, opts ...option.RequestOption) (*[]kernel.BrowserReplayListResponse, error) {
		return &replays, nil
	}}
	fakeBrowsers := &FakeBrowsersService{ListFunc: func(ctx context.Context, opts ...option.RequestOption) (*[]kernel.BrowserListResponse, error) {
		rows := []kernel.BrowserListResponse{{SessionID: "id"}}
		return &rows, nil
	}}
	b := BrowsersCmd{browsers: fakeBrowsers, replays: fake}
	_ = b.ReplaysList(context.Background(), BrowsersReplaysListInput{Identifier: "id"})
	out := outBuf.String()
	if !strings.Contains(out, "r1") || !strings.Contains(out, "http://v") {
		t.Fatalf("expected replay rows, got: %s", out)
	}
}

func TestBrowsersReplaysStart_PrintsInfo(t *testing.T) {
	setupStdoutCapture(t)
	fake := &FakeReplaysService{StartFunc: func(ctx context.Context, id string, body kernel.BrowserReplayStartParams, opts ...option.RequestOption) (*kernel.BrowserReplayStartResponse, error) {
		return &kernel.BrowserReplayStartResponse{ReplayID: "rid", ReplayViewURL: "http://view", StartedAt: time.Unix(0, 0)}, nil
	}}
	fakeBrowsers := &FakeBrowsersService{ListFunc: func(ctx context.Context, opts ...option.RequestOption) (*[]kernel.BrowserListResponse, error) {
		rows := []kernel.BrowserListResponse{{SessionID: "id"}}
		return &rows, nil
	}}
	b := BrowsersCmd{browsers: fakeBrowsers, replays: fake}
	_ = b.ReplaysStart(context.Background(), BrowsersReplaysStartInput{Identifier: "id", Framerate: 30, MaxDurationSeconds: 60})
	out := outBuf.String()
	if !strings.Contains(out, "rid") || !strings.Contains(out, "http://view") {
		t.Fatalf("expected start output, got: %s", out)
	}
}

func TestBrowsersReplaysStop_PrintsSuccess(t *testing.T) {
	setupStdoutCapture(t)
	fake := &FakeReplaysService{}
	fakeBrowsers := &FakeBrowsersService{ListFunc: func(ctx context.Context, opts ...option.RequestOption) (*[]kernel.BrowserListResponse, error) {
		rows := []kernel.BrowserListResponse{{SessionID: "id"}}
		return &rows, nil
	}}
	b := BrowsersCmd{browsers: fakeBrowsers, replays: fake}
	_ = b.ReplaysStop(context.Background(), BrowsersReplaysStopInput{Identifier: "id", ReplayID: "rid"})
	out := outBuf.String()
	if !strings.Contains(out, "Stopped replay rid") {
		t.Fatalf("expected stop message, got: %s", out)
	}
}

func TestBrowsersReplaysDownload_SavesFile(t *testing.T) {
	setupStdoutCapture(t)
	dir := t.TempDir()
	outPath := filepath.Join(dir, "replay.mp4")
	fake := &FakeReplaysService{DownloadFunc: func(ctx context.Context, replayID string, query kernel.BrowserReplayDownloadParams, opts ...option.RequestOption) (*http.Response, error) {
		return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": []string{"video/mp4"}}, Body: io.NopCloser(strings.NewReader("mp4data"))}, nil
	}}
	fakeBrowsers := &FakeBrowsersService{ListFunc: func(ctx context.Context, opts ...option.RequestOption) (*[]kernel.BrowserListResponse, error) {
		rows := []kernel.BrowserListResponse{{SessionID: "id"}}
		return &rows, nil
	}}
	b := BrowsersCmd{browsers: fakeBrowsers, replays: fake}
	_ = b.ReplaysDownload(context.Background(), BrowsersReplaysDownloadInput{Identifier: "id", ReplayID: "rid", Output: outPath})
	data, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("expected file saved, err: %v", err)
	}
	if string(data) != "mp4data" {
		t.Fatalf("expected content saved, got: %s", string(data))
	}
}

// --- Tests for Process ---

func TestBrowsersProcessExec_PrintsSummary(t *testing.T) {
	setupStdoutCapture(t)
	fake := &FakeProcessService{}
	fakeBrowsers := &FakeBrowsersService{ListFunc: func(ctx context.Context, opts ...option.RequestOption) (*[]kernel.BrowserListResponse, error) {
		rows := []kernel.BrowserListResponse{{SessionID: "id"}}
		return &rows, nil
	}}
	b := BrowsersCmd{browsers: fakeBrowsers, process: fake}
	_ = b.ProcessExec(context.Background(), BrowsersProcessExecInput{Identifier: "id", Command: "echo"})
	out := outBuf.String()
	if !strings.Contains(out, "Exit Code") || !strings.Contains(out, "Duration") {
		t.Fatalf("expected exec summary, got: %s", out)
	}
}

func TestBrowsersProcessSpawn_PrintsInfo(t *testing.T) {
	setupStdoutCapture(t)
	fake := &FakeProcessService{}
	fakeBrowsers := &FakeBrowsersService{ListFunc: func(ctx context.Context, opts ...option.RequestOption) (*[]kernel.BrowserListResponse, error) {
		rows := []kernel.BrowserListResponse{{SessionID: "id"}}
		return &rows, nil
	}}
	b := BrowsersCmd{browsers: fakeBrowsers, process: fake}
	_ = b.ProcessSpawn(context.Background(), BrowsersProcessSpawnInput{Identifier: "id", Command: "sleep"})
	out := outBuf.String()
	if !strings.Contains(out, "Process ID") || !strings.Contains(out, "PID") {
		t.Fatalf("expected spawn info, got: %s", out)
	}
}

func TestBrowsersProcessKill_PrintsSuccess(t *testing.T) {
	setupStdoutCapture(t)
	fake := &FakeProcessService{}
	fakeBrowsers := &FakeBrowsersService{ListFunc: func(ctx context.Context, opts ...option.RequestOption) (*[]kernel.BrowserListResponse, error) {
		rows := []kernel.BrowserListResponse{{SessionID: "id"}}
		return &rows, nil
	}}
	b := BrowsersCmd{browsers: fakeBrowsers, process: fake}
	_ = b.ProcessKill(context.Background(), BrowsersProcessKillInput{Identifier: "id", ProcessID: "proc", Signal: "TERM"})
	out := outBuf.String()
	if !strings.Contains(out, "Sent TERM to process proc") {
		t.Fatalf("expected kill message, got: %s", out)
	}
}

func TestBrowsersProcessStatus_PrintsFields(t *testing.T) {
	setupStdoutCapture(t)
	fake := &FakeProcessService{}
	fakeBrowsers := &FakeBrowsersService{ListFunc: func(ctx context.Context, opts ...option.RequestOption) (*[]kernel.BrowserListResponse, error) {
		rows := []kernel.BrowserListResponse{{SessionID: "id"}}
		return &rows, nil
	}}
	b := BrowsersCmd{browsers: fakeBrowsers, process: fake}
	_ = b.ProcessStatus(context.Background(), BrowsersProcessStatusInput{Identifier: "id", ProcessID: "proc"})
	out := outBuf.String()
	if !strings.Contains(out, "State") || !strings.Contains(out, "CPU %") || !strings.Contains(out, "Mem Bytes") {
		t.Fatalf("expected status fields, got: %s", out)
	}
}

func TestBrowsersProcessStdin_PrintsSuccess(t *testing.T) {
	setupStdoutCapture(t)
	fake := &FakeProcessService{}
	fakeBrowsers := &FakeBrowsersService{ListFunc: func(ctx context.Context, opts ...option.RequestOption) (*[]kernel.BrowserListResponse, error) {
		rows := []kernel.BrowserListResponse{{SessionID: "id"}}
		return &rows, nil
	}}
	b := BrowsersCmd{browsers: fakeBrowsers, process: fake}
	_ = b.ProcessStdin(context.Background(), BrowsersProcessStdinInput{Identifier: "id", ProcessID: "proc", DataB64: "ZGF0YQ=="})
	out := outBuf.String()
	if !strings.Contains(out, "Wrote to stdin") {
		t.Fatalf("expected stdin message, got: %s", out)
	}
}

func TestBrowsersProcessStdoutStream_PrintsExit(t *testing.T) {
	setupStdoutCapture(t)
	fake := &FakeProcessService{}
	fakeBrowsers := &FakeBrowsersService{ListFunc: func(ctx context.Context, opts ...option.RequestOption) (*[]kernel.BrowserListResponse, error) {
		rows := []kernel.BrowserListResponse{{SessionID: "id"}}
		return &rows, nil
	}}
	b := BrowsersCmd{browsers: fakeBrowsers, process: fake}
	_ = b.ProcessStdoutStream(context.Background(), BrowsersProcessStdoutStreamInput{Identifier: "id", ProcessID: "proc"})
	out := outBuf.String()
	if !strings.Contains(out, "process exited with code 0") {
		t.Fatalf("expected exit message, got: %s", out)
	}
}

// --- Tests for FS ---

func TestBrowsersFSNewDirectory_PrintsSuccess(t *testing.T) {
	setupStdoutCapture(t)
	fake := &FakeFSService{}
	fakeBrowsers := &FakeBrowsersService{ListFunc: func(ctx context.Context, opts ...option.RequestOption) (*[]kernel.BrowserListResponse, error) {
		rows := []kernel.BrowserListResponse{{SessionID: "id"}}
		return &rows, nil
	}}
	b := BrowsersCmd{browsers: fakeBrowsers, fs: fake}
	_ = b.FSNewDirectory(context.Background(), BrowsersFSNewDirInput{Identifier: "id", Path: "/tmp/x"})
	out := outBuf.String()
	if !strings.Contains(out, "Created directory /tmp/x") {
		t.Fatalf("expected created message, got: %s", out)
	}
}

func TestBrowsersFSDeleteDirectory_PrintsSuccess(t *testing.T) {
	setupStdoutCapture(t)
	fake := &FakeFSService{}
	fakeBrowsers := &FakeBrowsersService{ListFunc: func(ctx context.Context, opts ...option.RequestOption) (*[]kernel.BrowserListResponse, error) {
		rows := []kernel.BrowserListResponse{{SessionID: "id"}}
		return &rows, nil
	}}
	b := BrowsersCmd{browsers: fakeBrowsers, fs: fake}
	_ = b.FSDeleteDirectory(context.Background(), BrowsersFSDeleteDirInput{Identifier: "id", Path: "/tmp/x"})
	out := outBuf.String()
	if !strings.Contains(out, "Deleted directory /tmp/x") {
		t.Fatalf("expected deleted message, got: %s", out)
	}
}

func TestBrowsersFSDeleteFile_PrintsSuccess(t *testing.T) {
	setupStdoutCapture(t)
	fake := &FakeFSService{}
	fakeBrowsers := &FakeBrowsersService{ListFunc: func(ctx context.Context, opts ...option.RequestOption) (*[]kernel.BrowserListResponse, error) {
		rows := []kernel.BrowserListResponse{{SessionID: "id"}}
		return &rows, nil
	}}
	b := BrowsersCmd{browsers: fakeBrowsers, fs: fake}
	_ = b.FSDeleteFile(context.Background(), BrowsersFSDeleteFileInput{Identifier: "id", Path: "/tmp/file"})
	out := outBuf.String()
	if !strings.Contains(out, "Deleted file /tmp/file") {
		t.Fatalf("expected deleted message, got: %s", out)
	}
}

func TestBrowsersFSDownloadDirZip_SavesFile(t *testing.T) {
	dir := t.TempDir()
	outPath := filepath.Join(dir, "out.zip")
	fake := &FakeFSService{}
	fakeBrowsers := &FakeBrowsersService{ListFunc: func(ctx context.Context, opts ...option.RequestOption) (*[]kernel.BrowserListResponse, error) {
		rows := []kernel.BrowserListResponse{{SessionID: "id"}}
		return &rows, nil
	}}
	b := BrowsersCmd{browsers: fakeBrowsers, fs: fake}
	_ = b.FSDownloadDirZip(context.Background(), BrowsersFSDownloadDirZipInput{Identifier: "id", Path: "/tmp", Output: outPath})
	data, err := os.ReadFile(outPath)
	if err != nil || len(data) == 0 {
		t.Fatalf("expected zip saved, err=%v size=%d", err, len(data))
	}
}

func TestBrowsersFSFileInfo_PrintsFields(t *testing.T) {
	setupStdoutCapture(t)
	fake := &FakeFSService{FileInfoFunc: func(ctx context.Context, id string, query kernel.BrowserFFileInfoParams, opts ...option.RequestOption) (*kernel.BrowserFFileInfoResponse, error) {
		return &kernel.BrowserFFileInfoResponse{Path: "/tmp/a", Name: "a", Mode: "-rw-r--r--", IsDir: false, SizeBytes: 1, ModTime: time.Unix(0, 0)}, nil
	}}
	fakeBrowsers := &FakeBrowsersService{ListFunc: func(ctx context.Context, opts ...option.RequestOption) (*[]kernel.BrowserListResponse, error) {
		rows := []kernel.BrowserListResponse{{SessionID: "id"}}
		return &rows, nil
	}}
	b := BrowsersCmd{browsers: fakeBrowsers, fs: fake}
	_ = b.FSFileInfo(context.Background(), BrowsersFSFileInfoInput{Identifier: "id", Path: "/tmp/a"})
	out := outBuf.String()
	if !strings.Contains(out, "Path") || !strings.Contains(out, "/tmp/a") {
		t.Fatalf("expected fields, got: %s", out)
	}
}

func TestBrowsersFSListFiles_PrintsRows(t *testing.T) {
	setupStdoutCapture(t)
	fake := &FakeFSService{}
	fakeBrowsers := &FakeBrowsersService{ListFunc: func(ctx context.Context, opts ...option.RequestOption) (*[]kernel.BrowserListResponse, error) {
		rows := []kernel.BrowserListResponse{{SessionID: "id"}}
		return &rows, nil
	}}
	b := BrowsersCmd{browsers: fakeBrowsers, fs: fake}
	_ = b.FSListFiles(context.Background(), BrowsersFSListFilesInput{Identifier: "id", Path: "/"})
	out := outBuf.String()
	if !strings.Contains(out, "f1") || !strings.Contains(out, "/f1") {
		t.Fatalf("expected list row, got: %s", out)
	}
}

func TestBrowsersFSMove_PrintsSuccess(t *testing.T) {
	setupStdoutCapture(t)
	fake := &FakeFSService{}
	fakeBrowsers := &FakeBrowsersService{ListFunc: func(ctx context.Context, opts ...option.RequestOption) (*[]kernel.BrowserListResponse, error) {
		rows := []kernel.BrowserListResponse{{SessionID: "id"}}
		return &rows, nil
	}}
	b := BrowsersCmd{browsers: fakeBrowsers, fs: fake}
	_ = b.FSMove(context.Background(), BrowsersFSMoveInput{Identifier: "id", SrcPath: "/a", DestPath: "/b"})
	out := outBuf.String()
	if !strings.Contains(out, "Moved /a -> /b") {
		t.Fatalf("expected move message, got: %s", out)
	}
}

func TestBrowsersFSReadFile_SavesFile(t *testing.T) {
	dir := t.TempDir()
	outPath := filepath.Join(dir, "file.txt")
	fake := &FakeFSService{}
	fakeBrowsers := &FakeBrowsersService{ListFunc: func(ctx context.Context, opts ...option.RequestOption) (*[]kernel.BrowserListResponse, error) {
		rows := []kernel.BrowserListResponse{{SessionID: "id"}}
		return &rows, nil
	}}
	b := BrowsersCmd{browsers: fakeBrowsers, fs: fake}
	_ = b.FSReadFile(context.Background(), BrowsersFSReadFileInput{Identifier: "id", Path: "/tmp/x", Output: outPath})
	data, err := os.ReadFile(outPath)
	if err != nil || string(data) != "content" {
		t.Fatalf("expected file saved, err=%v content=%s", err, string(data))
	}
}

func TestBrowsersFSSetPermissions_PrintsSuccess(t *testing.T) {
	setupStdoutCapture(t)
	fake := &FakeFSService{}
	fakeBrowsers := &FakeBrowsersService{ListFunc: func(ctx context.Context, opts ...option.RequestOption) (*[]kernel.BrowserListResponse, error) {
		rows := []kernel.BrowserListResponse{{SessionID: "id"}}
		return &rows, nil
	}}
	b := BrowsersCmd{browsers: fakeBrowsers, fs: fake}
	_ = b.FSSetPermissions(context.Background(), BrowsersFSSetPermsInput{Identifier: "id", Path: "/tmp/a", Mode: "644"})
	out := outBuf.String()
	if !strings.Contains(out, "Updated permissions for /tmp/a") {
		t.Fatalf("expected perms message, got: %s", out)
	}
}

func TestBrowsersFSUpload_MappingAndDestDir_Success(t *testing.T) {
	setupStdoutCapture(t)
	var captured kernel.BrowserFUploadParams
	fake := &FakeFSService{UploadFunc: func(ctx context.Context, id string, body kernel.BrowserFUploadParams, opts ...option.RequestOption) error {
		captured = body
		return nil
	}}
	fakeBrowsers := &FakeBrowsersService{ListFunc: func(ctx context.Context, opts ...option.RequestOption) (*[]kernel.BrowserListResponse, error) {
		rows := []kernel.BrowserListResponse{{SessionID: "id"}}
		return &rows, nil
	}}
	b := BrowsersCmd{browsers: fakeBrowsers, fs: fake}
	in := BrowsersFSUploadInput{Identifier: "id", Mappings: []struct {
		Local string
		Dest  string
	}{{Local: __writeTempFile(t, "a"), Dest: "/remote/a"}}, DestDir: "/remote/dir", Paths: []string{__writeTempFile(t, "b")}}
	_ = b.FSUpload(context.Background(), in)
	out := outBuf.String()
	if !strings.Contains(out, "Uploaded") {
		t.Fatalf("expected upload message, got: %s", out)
	}
	if len(captured.Files) != 2 {
		t.Fatalf("expected 2 files sent, got %d", len(captured.Files))
	}
}

func TestBrowsersFSUploadZip_Success(t *testing.T) {
	setupStdoutCapture(t)
	z := __writeTempFile(t, "zipdata")
	fake := &FakeFSService{UploadZipFunc: func(ctx context.Context, id string, body kernel.BrowserFUploadZipParams, opts ...option.RequestOption) error {
		return nil
	}}
	fakeBrowsers := &FakeBrowsersService{ListFunc: func(ctx context.Context, opts ...option.RequestOption) (*[]kernel.BrowserListResponse, error) {
		rows := []kernel.BrowserListResponse{{SessionID: "id"}}
		return &rows, nil
	}}
	b := BrowsersCmd{browsers: fakeBrowsers, fs: fake}
	_ = b.FSUploadZip(context.Background(), BrowsersFSUploadZipInput{Identifier: "id", ZipPath: z, DestDir: "/dst"})
	out := outBuf.String()
	if !strings.Contains(out, "Uploaded zip") {
		t.Fatalf("expected upload zip message, got: %s", out)
	}
}

func TestBrowsersFSWriteFile_FromBase64_And_FromInput(t *testing.T) {
	setupStdoutCapture(t)
	fake := &FakeFSService{WriteFileFunc: func(ctx context.Context, id string, contents io.Reader, body kernel.BrowserFWriteFileParams, opts ...option.RequestOption) error {
		return nil
	}}
	fakeBrowsers := &FakeBrowsersService{ListFunc: func(ctx context.Context, opts ...option.RequestOption) (*[]kernel.BrowserListResponse, error) {
		rows := []kernel.BrowserListResponse{{SessionID: "id"}}
		return &rows, nil
	}}
	b := BrowsersCmd{browsers: fakeBrowsers, fs: fake}
	// input mode
	p := __writeTempFile(t, "hello")
	_ = b.FSWriteFile(context.Background(), BrowsersFSWriteFileInput{Identifier: "id", DestPath: "/y", SourcePath: p, Mode: "644"})
	out := outBuf.String()
	if !strings.Contains(out, "Wrote file to /y") {
		t.Fatalf("expected write messages, got: %s", out)
	}
}

// helper to create temp file with contents
func __writeTempFile(t *testing.T, data string) string {
	t.Helper()
	f, err := os.CreateTemp(t.TempDir(), "cli-test-*")
	if err != nil {
		t.Fatalf("temp: %v", err)
	}
	if _, err := f.WriteString(data); err != nil {
		t.Fatalf("write: %v", err)
	}
	_ = f.Close()
	return f.Name()
}
