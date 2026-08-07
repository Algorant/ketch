package scrape

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/1broseidon/ketch/config"
)

// obscuraConn renders a page through Obscura's one-shot CLI. The CLI emits
// rendered HTML on stdout and diagnostics on stderr, so it fits BrowserConn
// without requiring a long-lived daemon or a CDP client dependency.
type obscuraConn struct {
	bin        string
	stealth    bool
	storageDir string
}

func NewObscuraConn(bin string, stealth bool, storageDir string) (BrowserConn, error) {
	if bin == "" {
		bin = "obscura"
	}
	resolved, err := exec.LookPath(bin)
	if err != nil {
		return nil, fmt.Errorf("obscura not found at %q: %w", bin, err)
	}
	return &obscuraConn{bin: resolved, stealth: stealth, storageDir: storageDir}, nil
}

func (o *obscuraConn) Fetch(ctx context.Context, rawURL string) (string, error) {
	args := []string{"fetch", rawURL, "--dump", "html", "--quiet", "--timeout", "30", "--allow-private-network"}
	if o.stealth {
		args = append(args, "--stealth")
	}
	if o.storageDir != "" {
		args = append(args, "--storage-dir", filepath.Clean(o.storageDir))
	}

	var stdout, stderr bytes.Buffer
	cmd := exec.CommandContext(ctx, o.bin, args...)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	cmd.Env = config.ScrubbedEnviron()
	if err := cmd.Run(); err != nil {
		if ctx.Err() != nil {
			return "", ctx.Err()
		}
		diagnostic := strings.TrimSpace(stderr.String())
		if diagnostic == "" {
			diagnostic = strings.TrimSpace(stdout.String())
		}
		return "", fmt.Errorf("obscura fetch failed: %w: %s", err, diagnostic)
	}
	if stdout.Len() == 0 {
		return "", fmt.Errorf("obscura fetch returned empty HTML: %s", strings.TrimSpace(stderr.String()))
	}
	return stdout.String(), nil
}

func (o *obscuraConn) Close() {}

var _ BrowserConn = (*obscuraConn)(nil)
