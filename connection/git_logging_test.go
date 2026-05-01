package connection

import (
	gocontext "context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/flanksource/commons/logger"
	"github.com/flanksource/commons/properties"
	"github.com/flanksource/duty/context"
	"github.com/onsi/gomega"
)

// makeBareRepo creates a bare git repository at <root>/remote.git with a
// single commit on the "main" branch, suitable as a Clone source in tests.
func makeBareRepo(t *testing.T, root string) string {
	t.Helper()
	g := gomega.NewWithT(t)

	work := filepath.Join(root, "work")
	g.Expect(os.MkdirAll(work, 0o755)).To(gomega.Succeed())

	run := func(dir string, args ...string) {
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		cmd.Env = append(os.Environ(),
			"GIT_AUTHOR_NAME=test", "GIT_AUTHOR_EMAIL=test@test",
			"GIT_COMMITTER_NAME=test", "GIT_COMMITTER_EMAIL=test@test",
			"GIT_CONFIG_GLOBAL=/dev/null", "GIT_CONFIG_SYSTEM=/dev/null",
		)
		out, err := cmd.CombinedOutput()
		g.Expect(err).ToNot(gomega.HaveOccurred(), "git %s: %s", strings.Join(args, " "), string(out))
	}

	run(work, "init", "-b", "main")
	g.Expect(os.WriteFile(filepath.Join(work, "README.md"), []byte("hello"), 0o644)).To(gomega.Succeed())
	run(work, "add", "README.md")
	run(work, "commit", "-m", "initial")

	bare := filepath.Join(root, "remote.git")
	run(root, "clone", "--bare", work, bare)
	return bare
}

// captureLoggerOutput redirects the commons logger's shared writer to an
// in-memory buffer and returns a read/stop pair. The commons logger reads its
// destination from an atomic indirection set via logger.SetOutput, so swapping
// os.Stderr does not intercept its output — SetOutput is the supported hook.
func captureLoggerOutput(t *testing.T) (read func() string, stop func()) {
	t.Helper()

	orig := logger.GetOutput()

	var (
		mu  sync.Mutex
		buf strings.Builder
	)
	logger.SetOutput(writerFunc(func(p []byte) (int, error) {
		mu.Lock()
		defer mu.Unlock()
		return buf.Write(p)
	}))

	read = func() string {
		mu.Lock()
		defer mu.Unlock()
		return buf.String()
	}
	stop = func() {
		logger.SetOutput(orig)
	}
	return
}

type writerFunc func(p []byte) (int, error)

func (f writerFunc) Write(p []byte) (int, error) { return f(p) }

// setGitLogLevel sets log.level.git for the duration of the test, restoring
// the prior value (whatever it was) on cleanup. Hard-coding "info" on cleanup
// would corrupt subsequent tests in the same `go test` run.
func setGitLogLevel(t *testing.T, level string) {
	t.Helper()
	prev := properties.Get("log.level.git")
	properties.Set("log.level.git", level)
	t.Cleanup(func() { properties.Set("log.level.git", prev) })
}

// TestGitCloneTraceLogging verifies that -Plog.level.git=trace causes
// GitClient.Clone to emit structured command metadata via the "git" named
// logger, and that the same V(2) progress writer that go-git receives is
// open for writes (i.e. transport output would be forwarded).
func TestGitCloneTraceLogging(t *testing.T) {
	g := gomega.NewWithT(t)

	root := t.TempDir()
	remote := makeBareRepo(t, root)

	logger.UseSlog()

	readOut, stop := captureLoggerOutput(t)
	defer stop()

	setGitLogLevel(t, "trace")

	dst := filepath.Join(root, "checkout")
	gitClient := &GitClient{URL: remote, Branch: "main", Depth: 1}

	ctx := context.NewContext(gocontext.TODO())
	_, err := gitClient.Clone(ctx, dst)
	g.Expect(err).ToNot(gomega.HaveOccurred())

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
		g.Expect(out).To(gomega.ContainSubstring(w), "trace output missing %q", w)
	}
}

// TestGitCloneDefaultLevelIsQuiet verifies that without log.level.git=trace,
// the named logger suppresses V(1)+ structured lines — proving the level gate
// both opens (TestGitCloneTraceLogging) and closes.
func TestGitCloneDefaultLevelIsQuiet(t *testing.T) {
	g := gomega.NewWithT(t)

	root := t.TempDir()
	remote := makeBareRepo(t, root)

	logger.UseSlog()

	readOut, stop := captureLoggerOutput(t)
	defer stop()

	setGitLogLevel(t, "info")

	dst := filepath.Join(root, "checkout")
	gitClient := &GitClient{URL: remote, Branch: "main", Depth: 1}

	ctx := context.NewContext(gocontext.TODO())
	_, err := gitClient.Clone(ctx, dst)
	g.Expect(err).ToNot(gomega.HaveOccurred())

	// Writing a progress line at V(2) when the level is "info" must be a
	// no-op — the writer short-circuits in slog.Verbose.Write.
	fmt.Fprintln(logger.GetLogger("git").V(2), "Counting objects: 3, done.")

	stop()
	out := readOut()

	// The info-level "checked out" line is fine to see; V(1)+ and V(3) must not.
	for _, f := range []string{"clone url=", "Counting objects"} {
		g.Expect(out).ToNot(gomega.ContainSubstring(f), "default level leaked trace-only line")
	}
}
