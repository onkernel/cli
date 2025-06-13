package util

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"sync/atomic"

	kernel "github.com/onkernel/kernel-go-sdk"
	"github.com/onkernel/kernel-go-sdk/option"
	"github.com/pterm/pterm"
)

var printedUpgradeMessage atomic.Bool

// NewClient returns a kernel API client preconfigured with middleware that
// detects when a newer CLI/SDK version is required and informs the user.
//
// It mirrors kernel.NewClient but injects an HTTP middleware that intercepts
// 400 responses with error codes "sdk_upgrade_required" or
// "sdk_update_required". When encountered, a helpful upgrade message is
// displayed once per process.
func NewClient(opts ...option.RequestOption) kernel.Client {
	upgradeMw := func(req *http.Request, next option.MiddlewareNext) (*http.Response, error) {
		resp, err := next(req)
		if resp == nil {
			return resp, err
		}
		if resp.StatusCode != http.StatusBadRequest {
			return resp, err
		}

		// Read and buffer body so that downstream can still consume it.
		var buf bytes.Buffer
		if resp.Body != nil {
			_, _ = io.Copy(&buf, resp.Body)
			resp.Body.Close()
			resp.Body = io.NopCloser(bytes.NewBuffer(buf.Bytes()))
		}

		var body struct {
			Code string `json:"code"`
		}
		_ = json.Unmarshal(buf.Bytes(), &body)
		if body.Code == "sdk_upgrade_required" || body.Code == "sdk_update_required" {
			if !printedUpgradeMessage.Swap(true) {
				showUpgradeMessage()
			}
			// Immediately terminate the program with a non-zero exit code so
			// no further processing occurs.
			os.Exit(1)
		}
		return resp, err
	}

	opts = append(opts, option.WithMiddleware(upgradeMw))
	return kernel.NewClient(opts...)
}

// showUpgradeMessage prints an upgrade notice and sets the flag to ensure it
// is only displayed once per process.
func showUpgradeMessage() {
	pterm.Error.Println("Your Kernel CLI is out of date and is not compatible with this API.")
	pterm.Info.Println("Please upgrade by running: `brew upgrade onkernel/tap/kernel`")
}
