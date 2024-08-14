// Copyright (C) 2022 Couchbase, Inc.
//
// Use of this software is subject to the Couchbase Inc. License Agreement
// which may be found at https://www.couchbase.com/LA03012021.

package fluentbit

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sync"

	"go.uber.org/zap"

	"github.com/couchbaselabs/cbmultimanager/agent/pkg/bootstrap"
)

type FluentBit struct {
	logger             *zap.Logger
	executable         string
	node               *bootstrap.Node
	cbInstallationRoot string
	hazelnutPort       int
}

func findFluentBit(logger *zap.SugaredLogger) (flbExe string, err error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get CWD: %w", err)
	}
	flbExe = filepath.Join(cwd, fmt.Sprintf("fluent-bit-%s-%s", runtime.GOOS, runtime.GOARCH))
	logger.Debugw("Looking for Fluent Bit", "in", flbExe)
	var stat os.FileInfo
	if stat, err = os.Stat(flbExe); err == nil && stat.Size() > 0 {
		return
	}
	flbExe = filepath.Join(cwd, "fluent-bit")
	logger.Debugw("Looking for Fluent Bit", "in", flbExe)
	if stat, err = os.Stat(flbExe); err == nil && stat.Size() > 0 {
		return
	}
	flbExe = filepath.Join(cwd, "td-agent-bit")
	logger.Debugw("Looking for Fluent Bit", "in", flbExe)
	if stat, err = os.Stat(flbExe); err == nil && stat.Size() > 0 {
		return
	}
	logger.Debugw("Looking for Fluent Bit", "in", "$PATH/fluent-bit")
	if flbExe, err = exec.LookPath("fluent-bit"); err == nil {
		return
	}
	logger.Debugw("Looking for Fluent Bit", "in", "$PATH/td-agent-bit")
	if flbExe, err = exec.LookPath("td-agent-bit"); err == nil {
		return
	}

	// Ensure we tell the user something useful
	logger.Errorf("Could not find Fluent Bit executable. Ensure it is named either fluent-bit, "+
		"fluent-bit-%s-%s, or td-agent-bit, and located either in %s or on the system $PATH.",
		runtime.GOOS, runtime.GOARCH, cwd)
	return "", fmt.Errorf("could not find fluent-bit executable on path")
}

func NewFluentBit(node *bootstrap.Node, cbRoot string, hazelnutPort int) (*FluentBit, error) {
	flbExe, err := findFluentBit(zap.S().Named("fluent-bit"))
	if err != nil {
		return nil, err
	}

	return &FluentBit{
		logger:             zap.L().Named("fluent-bit"),
		executable:         flbExe,
		node:               node,
		cbInstallationRoot: cbRoot,
		hazelnutPort:       hazelnutPort,
	}, nil
}

type lineReader struct {
	reader io.Reader
	err    error
}

func (l *lineReader) Read(ctx context.Context) <-chan string {
	br := bufio.NewReader(l.reader)
	ch := make(chan string)
	go func() {
		defer close(ch)
		for {
			if ctx.Err() != nil {
				l.err = ctx.Err()
				return
			}
			line, err := br.ReadString('\n')
			if err != nil {
				l.err = err
				return
			}
			ch <- line
		}
	}()
	return ch
}

func (l *lineReader) Err() error {
	return l.err
}

func (f *FluentBit) Start(ctx context.Context, wg *sync.WaitGroup) error {
	defer wg.Done()

	us, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable: %w", err)
	}
	baseDir := filepath.Dir(us)

	cmd := exec.Command(f.executable, "-c", filepath.Join(baseDir, "etc", "fluent-bit/fluent-bit.conf"))
	cmd.Env = os.Environ()
	cmd.Env = append(cmd.Env, []string{
		fmt.Sprintf("FLB_CONFIG_ROOT=%s", filepath.Join(baseDir, "etc", "fluent-bit")),
		fmt.Sprintf("HOSTNAME=%s", f.node.Hostname()),
		fmt.Sprintf("couchbase_node=%s", f.node.Hostname()),
		fmt.Sprintf("couchbase_cluster=%s", f.node.Cluster().Name),
		fmt.Sprintf("COUCHBASE_LOGS=%s", filepath.Join(f.cbInstallationRoot, "var", "lib", "couchbase", "logs")),
		fmt.Sprintf("HAZELNUT_PORT=%d", f.hazelnutPort),
	}...)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("stdout pipe: %w", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("stderr pipe: %w", err)
	}

	f.logger.Debug("Starting Fluent Bit", zap.String("command", cmd.String()), zap.Any("env", cmd.Env))

	err = cmd.Start()
	if err != nil {
		return fmt.Errorf("failed to start: %w", err)
	}

	go func() {
		stdout := lineReader{reader: stdout}
		stderr := lineReader{reader: stderr}
		stdoutCh := stdout.Read(ctx)
		stderrCh := stderr.Read(ctx)
		for {
			select {
			case line, ok := <-stdoutCh:
				if !ok {
					stdoutCh = nil
				}
				f.logger.Debug(line)
			case line, ok := <-stderrCh:
				if !ok {
					stderrCh = nil
				}
				f.logger.Warn(line)
			}
			if stdoutCh == nil && stderrCh == nil {
				f.logger.Warn("All channels closed!", zap.NamedError("stdout", stdout.Err()),
					zap.NamedError("stderr", stderr.Err()))
				return
			}
		}
	}()

	go func() {
		<-ctx.Done()
		if cmd.Process != nil {
			f.logger.Info("Interrupting")
			if err := cmd.Process.Signal(os.Interrupt); err != nil {
				f.logger.Warn("Failed to interrupt", zap.Error(err))
			}
		}
	}()

	return nil
}
