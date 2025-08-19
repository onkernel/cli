package cmd

import (
	"bytes"
	"context"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/onkernel/kernel-go-sdk"
	"github.com/onkernel/kernel-go-sdk/option"
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
