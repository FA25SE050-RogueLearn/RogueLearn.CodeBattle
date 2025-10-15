package executor

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/FA25SE050-RogueLearn/RogueLearn.CodeBattle/internal/store"
)

const (
	QueryTimeOutSecond   = 30 * time.Second
	CodeRunTimeOutSecond = 15 * time.Second
)

type Job struct {
	Language  store.Language
	Code      string
	TestCases []store.TestCase // we will run all test cases in a job
	Result    chan Result
}

type CodeErr error

var CompileError CodeErr = errors.New("Failed to compile code")
var RunTimeError CodeErr = errors.New("Failed to compile code")
var FailTestCase CodeErr = errors.New("Test case failed")

type Result struct {
	Stdout        string
	Stderr        string
	Message       string
	Success       bool
	Error         CodeErr
	ExecutionTime string
}

type ExecuteCommandResult struct {
	Stdout   string
	Stderr   string
	Err      error
	Duration time.Duration
}

type WorkerPool struct {
	cm           *DockerContainerManager
	queries      *store.Queries
	logger       *slog.Logger
	jobs         chan Job
	wg           sync.WaitGroup
	shutdownChan chan any
}

type WorkerPoolOptions struct {
	MaxWorkers       int
	MemoryLimitBytes int64
	MaxJobCount      int
	CpuNanoLimit     int64
}

func NewWorkerPool(logger *slog.Logger, queries *store.Queries, opts *WorkerPoolOptions) (*WorkerPool, error) {
	cm, err := NewDockerContainerManager(opts.MaxWorkers, opts.MemoryLimitBytes, opts.CpuNanoLimit)
	if err != nil {
		return nil, err
	}

	err = cm.InitializePool()
	if err != nil {
		return nil, err
	}

	w := &WorkerPool{
		cm:           cm,
		queries:      queries,
		logger:       logger,
		jobs:         make(chan Job, opts.MaxJobCount),
		shutdownChan: make(chan any),
	}

	for i := range opts.MaxWorkers {
		w.wg.Add(1)
		go w.worker(i + 1)
	}

	w.logger.Info("Initialized worker pool with max workers",
		"max_worker", w.cm.maxWorkers)

	return w, err
}

func (w *WorkerPool) worker(id int) {
	defer w.wg.Done()
	w.logger.Info("Worker started", "id", id)

	for {
		select {
		case j, ok := <-w.jobs:
			if !ok {
				w.logger.Info("Worker shutting down due to channel closed",
					"worker_id", id)
				return
			}
			w.executeJob(id, j)

		case <-w.shutdownChan:
			w.logger.Info("Worker received shutdown signal", "worker_id", id)
			return
		}
	}
}

// ExecuteJob submits the job for execution
// input as a pointer so we could either set it or make it null
func (w *WorkerPool) ExecuteJob(lang store.Language, code string, tcs []store.TestCase) Result {
	w.logger.Info("Submitting job...",
		"language", lang)

	result := make(chan Result, 1)
	select {
	case w.jobs <- Job{Language: lang, Code: code, TestCases: tcs, Result: result}:
		return <-result
	default:
		w.logger.Warn("Job queue is full, rejecting job...",
			"language", lang,
			"maxJobCount", w.cm.maxWorkers)
		return Result{}
	}
}

// executeInContainer is a generic function to run a specific shell command in a container
func (w *WorkerPool) executeInContainer(ctx context.Context, containerID, command string, stdin io.Reader) ExecuteCommandResult {
	dockerArgs := []string{"exec", "-i", containerID, "sh", "-c", command}
	cmd := exec.CommandContext(ctx, "docker", dockerArgs...)

	var stdout, stderr bytes.Buffer
	cmd.Stdin = stdin
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	start := time.Now()
	err := cmd.Run()
	duration := time.Since(start)

	return ExecuteCommandResult{
		Stdout:   stdout.String(),
		Stderr:   stderr.String(),
		Err:      err,
		Duration: duration,
	}
}

// executeJob handle the execution of a single job
func (w *WorkerPool) executeJob(workerID int, job Job) error {
	ctx, cancel := context.WithTimeout(context.Background(), CodeRunTimeOutSecond)
	defer cancel()

	w.logger.Info("Job has been picked",
		"worker_id", workerID,
		"job", job)

	containerID, err := w.cm.GetAvailableContainer()
	if err != nil {
		w.logger.Error("Failed to get available Container",
			"err", err)
		return err
	}

	err = w.cm.SetContainerState(containerID, StateBusy)
	if err != nil {
		return err
	}

	defer func() {
		err = w.cm.SetContainerState(containerID, StateIdle)
		if err != nil {
			w.logger.Error("failed to set container state to idle",
				"container_id", containerID,
				"err", err)
		}
	}()

	start := time.Now()

	// Step 1: Copy code to container filesystem
	err = w.cm.copyCodeToContainer(ctx, containerID, job.Language.TempFileDir.String, job.Language.TempFileName.String, []byte(job.Code))
	if err != nil {
		w.logger.Error("Failed to copy code to container", "err", err)
		job.Result <- Result{Error: err, Success: false, Message: "Failed to set up execution environment."}
		return err
	}

	w.logger.Info("Code copied to container",
		"container_id", containerID,
		"code", job.Code)

	// Step 2: Run the code
	// If compiled lang -> compiled first.
	if job.Language.CompileCmd != "" { // case Compiled Lang
		// Create compile command
		compileCmd := strings.ReplaceAll(job.Language.CompileCmd, tempFileDirHolder, job.Language.TempFileDir.String)
		w.logger.Info("Compiling code...", "container_id", containerID, "command", compileCmd)

		compileResult := w.executeInContainer(ctx, containerID, compileCmd, nil)
		if compileResult.Err != nil {
			w.logger.Warn("Compilation failed",
				"err", compileResult.Err,
				"stderr", compileResult.Stderr,
				"stdout", compileResult.Stdout)
			job.Result <- Result{
				Error:   CompileError,
				Success: false,
				Stdout:  compileResult.Stdout,
				Stderr:  compileResult.Stderr,
				Message: "Compiled failed",
			}
			return err
		}
		w.logger.Info("Compilation successful", "duration", compileResult.Duration.Milliseconds())
	}

	// Step 4: Run all test case
	finalRunCmd := strings.ReplaceAll(job.Language.RunCmd, tempFileDirHolder, job.Language.TempFileDir.String)
	finalRunCmd = strings.ReplaceAll(finalRunCmd, tempFileNameHolder, job.Language.TempFileName.String)

	w.logger.Info("Preparing to run test cases", "command", finalRunCmd, "count", len(job.TestCases))

	totalExecutionTime := int64(0)
	for _, tc := range job.TestCases {
		runCtx, runCancel := context.WithTimeout(ctx, CodeRunTimeOutSecond)

		runResult := w.executeInContainer(runCtx, containerID, finalRunCmd, strings.NewReader(tc.Input))
		totalExecutionTime += runResult.Duration.Milliseconds()

		if runResult.Err != nil {
			w.logger.Warn("Runtime error", "test_case_id", tc.ID, "err", runResult.Err, "stderr", runResult.Stderr)
			job.Result <- Result{
				Error:   RunTimeError,
				Success: false,
				Stdout:  runResult.Stdout,
				Stderr:  runResult.Stderr,
				Message: "Runtime error",
			}
			runCancel()
			return nil
		}

		actualOutput := strings.TrimSpace(runResult.Stdout)
		w.logger.Info("Test case executed", "actual_output", actualOutput)
		expectedOutput := strings.TrimSpace(tc.ExpectedOutput)
		w.logger.Info("Test case executed", "expected_output", expectedOutput)

		if actualOutput != expectedOutput {
			w.logger.Warn("Wrong answer",
				"test_case_id", tc.ID,
				"actual_output", actualOutput,
				"expected_output", expectedOutput,
			)

			message := fmt.Sprintf("Wrong Answer on test case.\nInput:\n%s\n\nExpected Output:\n%s\n\nYour Output:\n%s", tc.Input, expectedOutput, actualOutput)
			job.Result <- Result{
				Success: false,
				Stdout:  runResult.Stdout,
				Stderr:  runResult.Stderr,
				Error:   FailTestCase,
				Message: message,
			}
			runCancel()
			return err
		}

		runCancel()
	}

	duration := time.Since(start)
	w.logger.Info("full process done", "took", duration)

	// Step 4: Send Result
	w.logger.Info("All test cases passed!", "worker_id", workerID)
	job.Result <- Result{
		Success:       true,
		Message:       "All test cases passed!",
		ExecutionTime: fmt.Sprintf("%dms", totalExecutionTime),
	}

	if err != nil {
		w.logger.Error("Worker job failed",
			"worker_id", workerID,
			"container_id", containerID,
			"duration", duration.Milliseconds(),
			"lang", job.Language,
			"err", err)
	} else {
		w.logger.Info("Worker job completed",
			"worker_id", workerID,
			"container_id", containerID,
			"duration", duration.Milliseconds(),
			"lang", job.Language)
	}

	return nil
}

// executeCode run the code in a specific Container
func (w *WorkerPool) executeCode(lang store.Language, containerID, code string, tcs []store.TestCase) (string, bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), CodeRunTimeOutSecond)
	defer cancel()

	// TODO: add input
	var stdout, stderr bytes.Buffer
	runCmd := generateRunCmd(lang.RunCmd, code)

	// -i for interactive
	cmd := exec.CommandContext(ctx, "docker", "exec", "-i", containerID, "sh", "-c", runCmd)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	start := time.Now()
	err := cmd.Run()
	duration := time.Since(start)

	// run all test cases
	w.logger.Info("executing test cases...",
		"number_of_test_case", len(tcs))

	//

	if err != nil {
		w.logger.Error("Failed to execute code",
			"container_id", containerID,
			"duration", duration,
			"err", err,
			"stdout", stdout.String(),
			"stderr", stderr.String())
		return stderr.String(), false, err
	}

	w.logger.Info("Code Execution Completed",
		"container_id", containerID,
		"duration", duration)

	return stdout.String(), true, nil
}

// generateCodeRunCmd will generate a run command for the code
func generateRunCmd(runCmd, finalCode string) string {
	formattedCode := strings.ReplaceAll(finalCode, "'", "'\\''")
	return fmt.Sprintf(runCmd, formattedCode)
}
