package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/liut-coder/game-helper-server/internal/modules/appreleases"
)

const workerAgentName = "releasectl-worker"

type workerRunConfig struct {
	BaseURL           string
	Token             string
	WorkerKey         string
	Name              string
	EndpointURL       string
	Labels            []string
	Capacity          int
	Workdir           string
	Execute           bool
	Once              bool
	PollInterval      time.Duration
	HeartbeatInterval time.Duration
	TaskTimeout       time.Duration
}

type workerCommandSpec struct {
	Source        string
	Args          []string
	Shell         string
	Display       string
	Env           map[string]string
	EnvSources    map[string]string
	CredentialRef string
}

func runWorkerRun(args []string) error {
	fs := flag.NewFlagSet("worker-run", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	baseURL := fs.String("base-url", env("GAME_HELPER_RELEASE_BASE_URL", "http://127.0.0.1:18080"), "server base URL")
	token := fs.String("token", firstNonBlankEnv("GAME_HELPER_WORKER_TOKEN", "GAME_HELPER_CI_TOKEN", "GAME_HELPER_ADMIN_TOKEN"), "worker or CI bearer token")
	workerKey := fs.String("worker-key", env("RELEASE_CENTER_WORKER_KEY", defaultWorkerKey()), "stable worker key")
	name := fs.String("name", env("RELEASE_CENTER_WORKER_NAME", ""), "worker display name")
	endpointURL := fs.String("endpoint-url", env("RELEASE_CENTER_WORKER_ENDPOINT", ""), "optional worker endpoint URL")
	labels := fs.String("labels", env("RELEASE_CENTER_WORKER_LABELS", defaultWorkerLabels()), "comma separated worker labels")
	capacity := fs.Int("capacity", 1, "reported worker capacity")
	workdir := fs.String("workdir", env("RELEASE_CENTER_WORKER_WORKDIR", ""), "command working directory")
	execute := fs.Bool("execute", false, "execute task command/prepared_command on this machine")
	once := fs.Bool("once", false, "claim and process at most one task")
	pollInterval := fs.Duration("poll-interval", 5*time.Second, "poll interval when no task is available")
	heartbeatInterval := fs.Duration("heartbeat-interval", 30*time.Second, "heartbeat interval")
	taskTimeout := fs.Duration("task-timeout", time.Hour, "per task command timeout")
	if err := fs.Parse(args); err != nil {
		return err
	}
	cfg := workerRunConfig{
		BaseURL:           strings.TrimRight(strings.TrimSpace(*baseURL), "/"),
		Token:             strings.TrimSpace(*token),
		WorkerKey:         strings.TrimSpace(*workerKey),
		Name:              strings.TrimSpace(*name),
		EndpointURL:       strings.TrimSpace(*endpointURL),
		Labels:            splitCSV(*labels),
		Capacity:          *capacity,
		Workdir:           strings.TrimSpace(*workdir),
		Execute:           *execute,
		Once:              *once,
		PollInterval:      *pollInterval,
		HeartbeatInterval: *heartbeatInterval,
		TaskTimeout:       *taskTimeout,
	}
	return runWorkerAgent(context.Background(), cfg)
}

func runWorkerAgent(ctx context.Context, cfg workerRunConfig) error {
	if cfg.BaseURL == "" {
		return fmt.Errorf("-base-url is required")
	}
	if cfg.Token == "" {
		return fmt.Errorf("-token is required")
	}
	if cfg.WorkerKey == "" {
		return fmt.Errorf("-worker-key is required")
	}
	if cfg.Name == "" {
		cfg.Name = cfg.WorkerKey
	}
	if cfg.Capacity <= 0 {
		cfg.Capacity = 1
	}
	if cfg.PollInterval <= 0 {
		cfg.PollInterval = 5 * time.Second
	}
	if cfg.HeartbeatInterval <= 0 {
		cfg.HeartbeatInterval = 30 * time.Second
	}
	if cfg.TaskTimeout <= 0 {
		cfg.TaskTimeout = time.Hour
	}
	if len(cfg.Labels) == 0 {
		cfg.Labels = splitCSV(defaultWorkerLabels())
	}
	if err := workerRegister(ctx, cfg); err != nil {
		return err
	}
	if err := workerHeartbeat(ctx, cfg, "online", 0); err != nil {
		return err
	}

	nextHeartbeat := time.Now().Add(cfg.HeartbeatInterval)
	for {
		task, err := workerNextTask(ctx, cfg)
		if err != nil {
			return err
		}
		if task == nil {
			if cfg.Once {
				printJSON(map[string]any{"ok": true, "message_zh": "暂无可领取任务"})
				return nil
			}
			if time.Now().After(nextHeartbeat) {
				if err := workerHeartbeat(ctx, cfg, "online", 0); err != nil {
					return err
				}
				nextHeartbeat = time.Now().Add(cfg.HeartbeatInterval)
			}
			time.Sleep(cfg.PollInterval)
			continue
		}
		if err := workerHeartbeat(ctx, cfg, "busy", 1); err != nil {
			return err
		}
		if err := executeWorkerTask(ctx, cfg, *task); err != nil {
			return err
		}
		if err := workerHeartbeat(ctx, cfg, "online", 0); err != nil {
			return err
		}
		if cfg.Once {
			return nil
		}
		nextHeartbeat = time.Now().Add(cfg.HeartbeatInterval)
	}
}

func executeWorkerTask(ctx context.Context, cfg workerRunConfig, task appreleases.WorkerTaskAdmin) error {
	spec, err := workerTaskCommand(task)
	if err != nil {
		return workerFailTask(ctx, cfg, task, err.Error(), map[string]any{
			"worker_agent": workerAgentName,
		})
	}
	if !cfg.Execute {
		lines := []string{"command execution disabled; rerun worker with -execute to process this task"}
		if err := workerAppendLogs(ctx, cfg, task, lines); err != nil {
			return err
		}
		return workerFailTask(ctx, cfg, task, lines[0], map[string]any{
			"worker_agent":           workerAgentName,
			"worker_command":         spec.Display,
			"worker_command_source":  spec.Source,
			"worker_execution_block": "execute_flag_required",
		})
	}

	taskCtx, cancel := context.WithTimeout(ctx, cfg.TaskTimeout)
	defer cancel()
	started := time.Now()
	out, exitCode, runErr := runWorkerCommand(taskCtx, cfg, spec)
	duration := time.Since(started)
	lines := workerOutputLines(out)
	if len(lines) == 0 {
		lines = []string{fmt.Sprintf("command finished with exit code %d", exitCode)}
	}
	if err := workerAppendLogs(ctx, cfg, task, lines); err != nil {
		return err
	}

	metadata := workerExecutionMetadata(spec, duration, exitCode, out)
	if runErr != nil {
		if errors.Is(taskCtx.Err(), context.DeadlineExceeded) {
			runErr = fmt.Errorf("worker task timed out after %s", cfg.TaskTimeout)
		}
		return workerFailTask(ctx, cfg, task, runErr.Error(), metadata)
	}
	return workerCompleteTask(ctx, cfg, task, metadata)
}

func runWorkerCommand(ctx context.Context, cfg workerRunConfig, spec workerCommandSpec) ([]byte, int, error) {
	var cmd *exec.Cmd
	if spec.Shell != "" {
		cmd = exec.CommandContext(ctx, shellBinary(), shellFlag(), spec.Shell)
	} else {
		if len(spec.Args) == 0 {
			return nil, -1, fmt.Errorf("empty worker command")
		}
		cmd = exec.CommandContext(ctx, spec.Args[0], spec.Args[1:]...)
	}
	if cfg.Workdir != "" {
		cmd.Dir = cfg.Workdir
	}
	cmd.Env = workerCommandEnv(os.Environ(), spec.Env)
	out, err := cmd.CombinedOutput()
	exitCode := 0
	if err != nil {
		exitCode = -1
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			exitCode = exitErr.ExitCode()
		}
	}
	return out, exitCode, err
}

func workerTaskCommand(task appreleases.WorkerTaskAdmin) (workerCommandSpec, error) {
	var metadata map[string]any
	if len(task.Metadata) > 0 {
		_ = json.Unmarshal(task.Metadata, &metadata)
	}
	if metadata == nil {
		metadata = map[string]any{}
	}
	for _, key := range []string{"prepared_command", "command"} {
		if spec, ok := commandSpecFromValue(key, metadata[key]); ok {
			return enrichWorkerCommandSpec(task, metadata, spec)
		}
	}
	if commands, ok := metadata["commands"].(map[string]any); ok {
		for _, key := range []string{task.Action, task.TaskType, "default"} {
			if spec, ok := commandSpecFromValue("commands."+key, commands[key]); ok {
				return enrichWorkerCommandSpec(task, metadata, spec)
			}
		}
	}
	return workerCommandSpec{}, fmt.Errorf("worker task does not include metadata.command or metadata.prepared_command")
}

func commandSpecFromValue(source string, value any) (workerCommandSpec, bool) {
	switch typed := value.(type) {
	case string:
		command := strings.TrimSpace(typed)
		if command == "" {
			return workerCommandSpec{}, false
		}
		return workerCommandSpec{Source: source, Shell: command, Display: command}, true
	case []string:
		args := trimCommandArgs(typed)
		if len(args) == 0 {
			return workerCommandSpec{}, false
		}
		return workerCommandSpec{Source: source, Args: args, Display: strings.Join(args, " ")}, true
	case []any:
		args := make([]string, 0, len(typed))
		for _, item := range typed {
			arg, ok := item.(string)
			if !ok {
				continue
			}
			args = append(args, arg)
		}
		args = trimCommandArgs(args)
		if len(args) == 0 {
			return workerCommandSpec{}, false
		}
		return workerCommandSpec{Source: source, Args: args, Display: strings.Join(args, " ")}, true
	default:
		return workerCommandSpec{}, false
	}
}

func enrichWorkerCommandSpec(task appreleases.WorkerTaskAdmin, metadata map[string]any, spec workerCommandSpec) (workerCommandSpec, error) {
	if !workerTaskIsCloudflare(task, metadata) {
		return spec, nil
	}
	credentialRef := metadataString(metadata, "credential_ref")
	token, tokenSource := lookupCloudflareAPIToken(credentialRef)
	if credentialRef != "" && token == "" {
		return workerCommandSpec{}, fmt.Errorf("cloudflare credential_ref %q is set but no matching token env was found", credentialRef)
	}
	if token != "" {
		workerSpecSetEnv(&spec, "CLOUDFLARE_API_TOKEN", token, tokenSource)
	}
	if accountID := metadataString(metadata, "cloudflare_account_id"); accountID != "" {
		workerSpecSetEnv(&spec, "CLOUDFLARE_ACCOUNT_ID", accountID, "task_metadata.cloudflare_account_id")
	}
	if credentialRef != "" {
		spec.CredentialRef = credentialRef
	}
	return spec, nil
}

func workerTaskIsCloudflare(task appreleases.WorkerTaskAdmin, metadata map[string]any) bool {
	return strings.HasPrefix(strings.ToLower(strings.TrimSpace(task.Action)), "cloudflare_") ||
		strings.HasPrefix(strings.ToLower(metadataString(metadata, "provider")), "cloudflare_")
}

func workerSpecSetEnv(spec *workerCommandSpec, key, value, source string) {
	key = strings.TrimSpace(key)
	value = strings.TrimSpace(value)
	if key == "" || value == "" {
		return
	}
	if spec.Env == nil {
		spec.Env = map[string]string{}
	}
	if spec.EnvSources == nil {
		spec.EnvSources = map[string]string{}
	}
	spec.Env[key] = value
	spec.EnvSources[key] = source
}

func lookupCloudflareAPIToken(credentialRef string) (string, string) {
	for _, key := range cloudflareCredentialEnvNames(credentialRef) {
		if value := strings.TrimSpace(os.Getenv(key)); value != "" {
			return value, key
		}
	}
	return "", ""
}

func cloudflareCredentialEnvNames(credentialRef string) []string {
	ref := credentialEnvSuffix(credentialRef)
	names := []string{}
	if ref != "" {
		names = append(names,
			"RELEASE_CENTER_CREDENTIAL_"+ref,
			"CLOUDFLARE_API_TOKEN_"+ref,
			ref,
		)
	}
	names = append(names, "CLOUDFLARE_API_TOKEN")
	return names
}

func credentialEnvSuffix(value string) string {
	value = strings.ToUpper(strings.TrimSpace(value))
	if value == "" {
		return ""
	}
	var b strings.Builder
	lastUnderscore := false
	for _, r := range value {
		if (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
			lastUnderscore = false
			continue
		}
		if !lastUnderscore {
			b.WriteByte('_')
			lastUnderscore = true
		}
	}
	return strings.Trim(b.String(), "_")
}

func metadataString(metadata map[string]any, key string) string {
	value, ok := metadata[key]
	if !ok {
		return ""
	}
	if value == nil {
		return ""
	}
	switch typed := value.(type) {
	case string:
		return strings.TrimSpace(typed)
	case fmt.Stringer:
		return strings.TrimSpace(typed.String())
	default:
		return strings.TrimSpace(fmt.Sprint(typed))
	}
}

func workerExecutionMetadata(spec workerCommandSpec, duration time.Duration, exitCode int, output []byte) map[string]any {
	metadata := map[string]any{
		"worker_agent":          workerAgentName,
		"worker_command":        spec.Display,
		"worker_command_source": spec.Source,
		"worker_duration_ms":    duration.Milliseconds(),
		"worker_exit_code":      exitCode,
	}
	if deploymentURL := firstURL(string(output)); deploymentURL != "" {
		metadata["deployment_url"] = deploymentURL
	}
	if spec.CredentialRef != "" {
		metadata["worker_credential_ref"] = spec.CredentialRef
	}
	if len(spec.Env) > 0 {
		metadata["worker_env_keys"] = sortedMapKeys(spec.Env)
	}
	if len(spec.EnvSources) > 0 {
		metadata["worker_env_sources"] = spec.EnvSources
	}
	return metadata
}

func workerRegister(ctx context.Context, cfg workerRunConfig) error {
	var resp appreleases.WorkerActionResponse
	return workerDoJSON(ctx, cfg, http.MethodPost, "/api/v1/workers/register", appreleases.WorkerRegisterRequest{
		WorkerKey:   cfg.WorkerKey,
		Name:        cfg.Name,
		EndpointURL: cfg.EndpointURL,
		Labels:      cfg.Labels,
		Capacity:    cfg.Capacity,
		Metadata: map[string]any{
			"agent":   workerAgentName,
			"goos":    runtime.GOOS,
			"goarch":  runtime.GOARCH,
			"execute": cfg.Execute,
		},
	}, &resp)
}

func workerHeartbeat(ctx context.Context, cfg workerRunConfig, status string, runningTasks int) error {
	var resp appreleases.WorkerActionResponse
	return workerDoJSON(ctx, cfg, http.MethodPost, "/api/v1/workers/heartbeat", appreleases.WorkerHeartbeatRequest{
		WorkerKey:    cfg.WorkerKey,
		Status:       status,
		Labels:       cfg.Labels,
		Capacity:     cfg.Capacity,
		RunningTasks: runningTasks,
		Metadata: map[string]any{
			"agent":   workerAgentName,
			"goos":    runtime.GOOS,
			"goarch":  runtime.GOARCH,
			"execute": cfg.Execute,
		},
	}, &resp)
}

func workerNextTask(ctx context.Context, cfg workerRunConfig) (*appreleases.WorkerTaskAdmin, error) {
	var resp appreleases.WorkerTaskNextResponse
	path := "/api/v1/workers/tasks/next?worker_key=" + url.QueryEscape(cfg.WorkerKey)
	if err := workerDoJSON(ctx, cfg, http.MethodGet, path, nil, &resp); err != nil {
		return nil, err
	}
	return resp.Task, nil
}

func workerAppendLogs(ctx context.Context, cfg workerRunConfig, task appreleases.WorkerTaskAdmin, lines []string) error {
	var resp appreleases.WorkerActionResponse
	path := "/api/v1/workers/tasks/" + url.PathEscape(task.ID) + "/logs"
	return workerDoJSON(ctx, cfg, http.MethodPost, path, appreleases.WorkerTaskLogsRequest{
		WorkerKey:  cfg.WorkerKey,
		LeaseToken: task.LeaseToken,
		Lines:      lines,
	}, &resp)
}

func workerCompleteTask(ctx context.Context, cfg workerRunConfig, task appreleases.WorkerTaskAdmin, metadata map[string]any) error {
	var resp appreleases.WorkerActionResponse
	path := "/api/v1/workers/tasks/" + url.PathEscape(task.ID) + "/complete"
	if err := workerDoJSON(ctx, cfg, http.MethodPost, path, appreleases.WorkerTaskCompleteRequest{
		WorkerKey:  cfg.WorkerKey,
		LeaseToken: task.LeaseToken,
		Metadata:   metadata,
	}, &resp); err != nil {
		return err
	}
	printJSON(resp)
	return nil
}

func workerFailTask(ctx context.Context, cfg workerRunConfig, task appreleases.WorkerTaskAdmin, errorMessage string, metadata map[string]any) error {
	var resp appreleases.WorkerActionResponse
	path := "/api/v1/workers/tasks/" + url.PathEscape(task.ID) + "/fail"
	if err := workerDoJSON(ctx, cfg, http.MethodPost, path, appreleases.WorkerTaskFailRequest{
		WorkerKey:    cfg.WorkerKey,
		LeaseToken:   task.LeaseToken,
		ErrorMessage: strings.TrimSpace(errorMessage),
		Metadata:     metadata,
	}, &resp); err != nil {
		return err
	}
	printJSON(resp)
	return nil
}

func workerDoJSON(ctx context.Context, cfg workerRunConfig, method, path string, body any, out any) error {
	var reader io.Reader
	if body != nil {
		encoded, err := json.Marshal(body)
		if err != nil {
			return err
		}
		reader = bytes.NewReader(encoded)
	}
	req, err := http.NewRequestWithContext(ctx, method, joinURL(cfg.BaseURL, path), reader)
	if err != nil {
		return err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Authorization", "Bearer "+cfg.Token)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		msg, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("%s %s failed: %s %s", method, path, resp.Status, strings.TrimSpace(string(msg)))
	}
	if out == nil {
		return nil
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

func workerOutputLines(output []byte) []string {
	text := strings.ReplaceAll(string(output), "\r\n", "\n")
	text = strings.TrimRight(text, "\n")
	if text == "" {
		return []string{}
	}
	lines := strings.Split(text, "\n")
	for i := range lines {
		lines[i] = strings.TrimRight(lines[i], "\r")
	}
	if len(lines) > 200 {
		lines = lines[len(lines)-200:]
	}
	return lines
}

func trimCommandArgs(args []string) []string {
	out := make([]string, 0, len(args))
	for _, arg := range args {
		arg = strings.TrimSpace(arg)
		if arg != "" {
			out = append(out, arg)
		}
	}
	return out
}

func workerCommandEnv(base []string, extra map[string]string) []string {
	if len(extra) == 0 {
		return base
	}
	keys := sortedMapKeys(extra)
	override := map[string]struct{}{}
	for _, key := range keys {
		override[key] = struct{}{}
	}
	out := make([]string, 0, len(base)+len(extra))
	for _, item := range base {
		key, _, ok := strings.Cut(item, "=")
		if ok {
			if _, exists := override[key]; exists {
				continue
			}
		}
		out = append(out, item)
	}
	for _, key := range keys {
		out = append(out, key+"="+extra[key])
	}
	return out
}

func sortedMapKeys(values map[string]string) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func splitCSV(value string) []string {
	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	seen := map[string]struct{}{}
	for _, part := range parts {
		part = strings.ToLower(strings.TrimSpace(part))
		if part == "" {
			continue
		}
		if _, ok := seen[part]; ok {
			continue
		}
		seen[part] = struct{}{}
		out = append(out, part)
	}
	return out
}

func defaultWorkerLabels() string {
	return runtime.GOOS + "," + runtime.GOARCH
}

func defaultWorkerKey() string {
	host, _ := os.Hostname()
	host = strings.TrimSpace(host)
	if host == "" {
		return runtime.GOOS + "-" + runtime.GOARCH
	}
	return host
}

func firstNonBlankEnv(keys ...string) string {
	for _, key := range keys {
		if value := strings.TrimSpace(os.Getenv(key)); value != "" {
			return value
		}
	}
	return ""
}

func shellBinary() string {
	if runtime.GOOS == "windows" {
		return "cmd"
	}
	return "sh"
}

func shellFlag() string {
	if runtime.GOOS == "windows" {
		return "/C"
	}
	return "-lc"
}

var urlPattern = regexp.MustCompile(`https?://[^\s"'<>]+`)

func firstURL(text string) string {
	match := urlPattern.FindString(text)
	return strings.TrimRight(match, ".,);]")
}
