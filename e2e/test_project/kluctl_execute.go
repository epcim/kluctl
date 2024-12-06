package test_project

import (
	"bytes"
	"context"
	status2 "github.com/kluctl/kluctl/lib/status"
	"github.com/kluctl/kluctl/v2/cmd/kluctl/commands"
	"github.com/kluctl/kluctl/v2/pkg/utils"
	"strings"
	"sync"
	"testing"
)

func KluctlExecute(t *testing.T, ctx context.Context, logFn func(args ...any), args ...string) (string, string, error) {
	t.Logf("Runnning kluctl: %s", strings.Join(args, " "))

	var m sync.Mutex

	stdoutBuf := bytes.NewBuffer(nil)
	stderrBuf := bytes.NewBuffer(nil)

	stdout := status2.NewLineRedirector(func(line string) {
		m.Lock()
		defer m.Unlock()
		logFn(line)
		stdoutBuf.WriteString(line + "\n")
	})
	stderr := status2.NewLineRedirector(func(line string) {
		m.Lock()
		defer m.Unlock()
		logFn(line)
		stderrBuf.WriteString(line + "\n")
	})

	if utils.GetTmpBaseDirNoDefault(ctx) == "" {
		ctx = utils.WithTmpBaseDir(ctx, t.TempDir())
	}
	if utils.GetCacheDirNoDefault(ctx) == "" {
		ctx = utils.WithCacheDir(ctx, t.TempDir())
	}
	ctx = commands.WithStdStreams(ctx, stdout, stderr)
	sh := status2.NewSimpleStatusHandler(func(level status2.Level, message string) {
		_, _ = stderr.Write([]byte(message + "\n"))
	}, true)
	defer func() {
		if sh != nil {
			sh.Stop()
		}
	}()
	ctx = status2.NewContext(ctx, sh)
	err := commands.Execute(ctx, args, nil)
	sh.Stop()
	sh = nil

	_ = stdout.Close()
	_ = stderr.Close()

	<-stdout.Done()
	<-stderr.Done()

	m.Lock()
	defer m.Unlock()
	return stdoutBuf.String(), stderrBuf.String(), err
}
