package connection

import (
	gocontext "context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/flanksource/commons/logger"
	"github.com/flanksource/commons/properties"
	"github.com/flanksource/duty/context"
)

// makeBareRepo creates a bare git repository at <root>/remote.git with a
// single commit on the "main" branch, suitable as a Clone source in tests.
func makeBareRepo(t *testing.T, root string) string {
	t.Helper()

	work := filepath.Join(root, "work")
	if err := os.MkdirAll(work, 0o755); err != nil {
		t.Fatalf("mkdir work: %v", err)
	}

	run := func(dir string, args ...string) {
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		cmd.Env = append(os.Environ(),
			"GIT_AUTHOR_NAME=test", "GIT_AUTHOR_EMAIL=test@test",
			"GIT_COMMITTER_NAME=test", "GIT_COMMITTER_EMAIL=test@test",
			"GIT_CONFIG_GLOBAL=/dev/null", "GIT_CONFIG_SYSTEM=/dev/null",
		)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, string(out))
		}
	}

	run(work, "init", "-b", "main")
	if err := os.WriteFile(filepath.Join(work, "README.md"), []byte("hello"), 0o644); err != nil {
		t.Fatalf("write README: %v", err)
	}
	run(work, "add", "README.md")
	run(work, "commit", "-m", "initial")

	bare := filepath.Join(root, "remote.git")
	run(root, "clone", "--bare", work, bare)
	return bare
}

// captureStderr swaps os.Stderr for a pipe and returns a read/stop pair.
// Must be called BEFORE any logger that targets stderr is constructed, since
// slog handlers capture the writer reference at build time.
func captureStderr(t *testing.T) (read func() string, stop func()) {
	t.Helper()

	orig := os.Stderr
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	os.Stderr = w

	var (
		mu  sync.Mutex
		buf strings.Builder
		wg  sync.WaitGroup
	)
	wg.Go(func() {
		b, _ := io.ReadAll(r)
		mu.Lock()
		buf.Write(b)
		mu.Unlock()
	})

	read = func() string {
		mu.Lock()
		defer mu.Unlock()
		return buf.String()
	}
	stop = func() {
		os.Stderr = orig
		_ = w.Close()
		wg.Wait()
		_ = r.Close()
	}
	return
}

// TestGitCloneTraceLogging verifies that -Plog.level.git=trace causes
// GitClient.Clone to emit structured command metadata via the "git" named
// logger, and that the same V(2) progress writer that go-git receives is
// open for writes (i.e. transport output would be forwarded).
func TestGitCloneTraceLogging(t *testing.T) {
	root := t.TempDir()
	remote := makeBareRepo(t, root)

	logger.UseSlog()

	readOut, stop := captureStderr(t)
	defer stop()

	// log.level.git=trace flips the "git" named logger via the property
	// listener. Any logger instantiated while os.Stderr is the pipe will
	// target the pipe for its lifetime.
	properties.Set("log.level.git", "trace")
	t.Cleanup(func() { properties.Set("log.level.git", "info") })

	dst := filepath.Join(root, "checkout")
	gitClient := &GitClient{URL: remote, Branch: "main", Depth: 1}

	ctx := context.NewContext(gocontext.TODO())
	if _, err := gitClient.Clone(ctx, dst); err != nil {
		t.Fatalf("clone: %v", err)
	}

	// Simulate a transport progress line going through the same writer
	// Clone hands to go-git, to prove the path is live. A real network
	// clone emits lines like this via sideband; local/file clones do not.
	fmt.Fprintln(logger.GetLogger("git").V(2), "Counting objects: 3, done.")

	stop()
	out := readOut()

	want := []string{
		"clone url=",
		"branch=main",
		"depth=1",
		"dir=" + dst,
		"checked out ",
		"Counting objects: 3, done.",
		"(git)",
	}
	for _, w := range want {
		if !strings.Contains(out, w) {
			t.Errorf("trace output missing %q\n--- captured ---\n%s", w, out)
		}
	}
}

// TestGitCloneDefaultLevelIsQuiet verifies that without log.level.git=trace,
// the named logger suppresses V(1)+ structured lines — proving the level gate
// both opens (TestGitCloneTraceLogging) and closes.
func TestGitCloneDefaultLevelIsQuiet(t *testing.T) {
	root := t.TempDir()
	remote := makeBareRepo(t, root)

	logger.UseSlog()

	readOut, stop := captureStderr(t)
	defer stop()

	properties.Set("log.level.git", "info")
	t.Cleanup(func() { properties.Set("log.level.git", "info") })

	dst := filepath.Join(root, "checkout")
	gitClient := &GitClient{URL: remote, Branch: "main", Depth: 1}

	ctx := context.NewContext(gocontext.TODO())
	if _, err := gitClient.Clone(ctx, dst); err != nil {
		t.Fatalf("clone: %v", err)
	}

	// Writing a progress line at V(2) when the level is "info" must be a
	// no-op — the writer short-circuits in slog.Verbose.Write.
	fmt.Fprintln(logger.GetLogger("git").V(2), "Counting objects: 3, done.")

	stop()
	out := readOut()

	// The info-level "checked out" line is fine to see; V(1)+ and V(3) must not.
	forbidden := []string{
		"clone url=",
		"Counting objects",
	}
	for _, f := range forbidden {
		if strings.Contains(out, f) {
			t.Errorf("default level leaked trace-only line %q\n--- captured ---\n%s", f, out)
		}
	}
}
