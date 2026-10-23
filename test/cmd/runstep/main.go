// Command runstep runs a CustomPromotionStep locally, the way the Kargo
// Enterprise promotion executor does, and checks the result against a fixture.
//
//   - command and env are rendered with Kargo's own template engine
//     (github.com/akuity/kargo/pkg/expressions), so the re-parsing gotchas
//     documented in the README reproduce exactly.
//   - The image runs as UID/GID 65532 with all capabilities dropped and no
//     privilege escalation, with the step's working directory mounted at /workdir.
//   - Output is collected per spec.output (Pipe/Stdout/Stderr/File and
//     KeyValue/JSON/YAML/Text) and transformed with expr-lang, as the executor does.
//
// Usage:
//
//	runstep -step <step.yaml> -fixture <fixture.yaml> [-repo <repo root>] [-v]
package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"regexp"
	"sort"
	"strings"

	"github.com/akuity/kargo/pkg/expressions"
	"github.com/expr-lang/expr"
	"go.yaml.in/yaml/v3"
)

const (
	stepUID            = "65532"
	workdirMount       = "/workdir"
	kargoOutputPath    = "/kargo-output/output"
	maxOutputBytes     = 256 * 1024
	maxMessageBytes    = 16 * 1024
	statusSucceeded    = "Succeeded"
	statusFailed       = "Failed"
	statusErrored      = "Errored"
	outputKey          = "output"
	defaultFixtureStep = "step"
)

// step mirrors the parts of ee.kargo.akuity.io/v1alpha1 CustomPromotionStep
// that affect execution.
type step struct {
	Kind     string `yaml:"kind"`
	Metadata struct {
		Name string `yaml:"name"`
	} `yaml:"metadata"`
	Spec struct {
		Image   string   `yaml:"image"`
		Command []string `yaml:"command"`
		Env     []struct {
			Name  string `yaml:"name"`
			Value string `yaml:"value"`
		} `yaml:"env"`
		Output *struct {
			Source struct {
				Type   string `yaml:"type"`
				Path   string `yaml:"path"`
				Format string `yaml:"format"`
			} `yaml:"source"`
			Transform map[string]string `yaml:"transform"`
		} `yaml:"output"`
	} `yaml:"spec"`
}

// fixture describes one test case for a step.
type fixture struct {
	Description string         `yaml:"description"`
	Config      map[string]any `yaml:"config"`
	Ctx         map[string]any `yaml:"ctx"`
	// Files maps a path inside the workdir to a path relative to the repo
	// root. Directories are copied recursively. This stands in for the
	// built-in steps (git-clone, helm-template, ...) that would normally
	// populate the workdir before the custom step runs.
	Files map[string]string `yaml:"files"`
	// Env adds container environment variables, e.g. ones the executor
	// injects such as ARGOCD_NAMESPACE.
	Env map[string]string `yaml:"env"`
	// Docker adds raw `docker run` flags, e.g. --network for Tier B tests.
	Docker []string `yaml:"docker"`
	Expect struct {
		Status          string         `yaml:"status"`
		Output          map[string]any `yaml:"output"`
		OutputKeys      []string       `yaml:"outputKeys"`
		MessageContains []string       `yaml:"messageContains"`
	} `yaml:"expect"`
}

func main() {
	stepPath := flag.String("step", "", "path to the CustomPromotionStep YAML")
	fixturePath := flag.String("fixture", "", "path to the fixture YAML")
	repoRoot := flag.String("repo", ".", "repository root, for resolving fixture files")
	verbose := flag.Bool("v", false, "print the rendered command, stdout and stderr")
	showMessage := flag.Bool("message", false, "also print the step message the promotion would show")
	flag.Parse()
	if *stepPath == "" || *fixturePath == "" {
		flag.Usage()
		os.Exit(2)
	}
	if err := run(*stepPath, *fixturePath, *repoRoot, *verbose, *showMessage); err != nil {
		fmt.Fprintf(os.Stderr, "FAIL %s: %v\n", *fixturePath, err)
		os.Exit(1)
	}
}

func run(stepPath, fixturePath, repoRoot string, verbose, showMessage bool) error {
	var s step
	if err := readYAML(stepPath, &s); err != nil {
		return err
	}
	if s.Kind != "CustomPromotionStep" {
		return fmt.Errorf("%s: kind is %q, want CustomPromotionStep", stepPath, s.Kind)
	}
	var f fixture
	if err := readYAML(fixturePath, &f); err != nil {
		return err
	}

	exprEnv := map[string]any{
		"config": nonNil(f.Config),
		"ctx":    buildCtx(f.Ctx),
	}
	command, err := renderAll(s.Spec.Command, exprEnv)
	if err != nil {
		return fmt.Errorf("cannot render command template: %w", err)
	}
	if len(command) == 0 {
		return errors.New("step command should not be empty")
	}
	var env []string
	for _, e := range s.Spec.Env {
		v, err := render(e.Value, exprEnv)
		if err != nil {
			return fmt.Errorf("cannot render env %s: %w", e.Name, err)
		}
		env = append(env, e.Name+"="+v)
	}

	workdir, err := os.MkdirTemp("", "runstep-workdir-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(workdir)
	outDir, err := os.MkdirTemp("", "runstep-output-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(outDir)
	if err := os.WriteFile(filepath.Join(outDir, "output"), nil, 0o666); err != nil {
		return err
	}
	for dest, src := range f.Files {
		if err := copyPath(filepath.Join(repoRoot, src), filepath.Join(workdir, dest)); err != nil {
			return fmt.Errorf("copying fixture file %s: %w", src, err)
		}
	}
	// The step runs as 65532, which does not own the temp dirs.
	if err := chmodAll(workdir); err != nil {
		return err
	}
	if err := chmodAll(outDir); err != nil {
		return err
	}

	container := fmt.Sprintf("runstep-%d", os.Getpid())
	args := []string{
		"run", "--name", container,
		"--user", stepUID + ":" + stepUID,
		"--cap-drop", "ALL",
		"--security-opt", "no-new-privileges",
		"-v", workdir + ":" + workdirMount,
		"-w", workdirMount,
		"-v", outDir + ":" + filepath.Dir(kargoOutputPath),
		"-e", "KARGO_OUTPUT=" + kargoOutputPath,
		"-e", "STEP_DIRECTORY=/coordination",
	}
	for _, kv := range env {
		args = append(args, "-e", kv)
	}
	for _, k := range sortedKeys(f.Env) {
		args = append(args, "-e", k+"="+f.Env[k])
	}
	args = append(args, f.Docker...)
	args = append(args, "--entrypoint", command[0], s.Spec.Image)
	args = append(args, command[1:]...)

	if verbose {
		fmt.Fprintf(os.Stderr, "--- rendered command:\n%q\n--- rendered env:\n%s\n", command, strings.Join(env, "\n"))
	}
	var stdout, stderr bytes.Buffer
	cmd := exec.Command("docker", args...)
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	runErr := cmd.Run()
	defer exec.Command("docker", "rm", "-f", container).Run() //nolint:errcheck

	var exitErr *exec.ExitError
	if runErr != nil && !errors.As(runErr, &exitErr) {
		return fmt.Errorf("docker run: %w", runErr)
	}
	stdoutStr, stderrStr := sanitize(stdout.String()), sanitize(stderr.String())
	if verbose {
		fmt.Fprintf(os.Stderr, "--- stdout:\n%s\n--- stderr:\n%s\n", stdoutStr, stderrStr)
	}
	if exitErr != nil && exitErr.ExitCode() == 125 {
		return fmt.Errorf("docker could not start the container: %s", stderrStr)
	}

	status := statusSucceeded
	if exitErr != nil {
		status = statusFailed
	}
	output, collectErr := collectOutput(s, container, stdoutStr, stderrStr, filepath.Join(outDir, "output"))
	if collectErr != nil && status == statusSucceeded {
		status = statusErrored
	}
	prefix := "Step succeeded"
	if status != statusSucceeded {
		prefix = "Step failed"
	}
	message := buildMessage(prefix, stdoutStr, stderrStr)
	if collectErr != nil {
		message += fmt.Sprintf(" (output error: %v)", collectErr)
	}

	pretty, _ := json.MarshalIndent(output, "", "  ")
	fmt.Printf("%s %s [%s]\n%s\n", status, fixturePath, f.Description, pretty)
	if showMessage {
		fmt.Printf("MESSAGE: %s\n", message)
	}
	return check(f, status, output, message)
}

func buildCtx(override map[string]any) map[string]any {
	ctx := map[string]any{
		"uiBaseUrl": "https://kargo.example.com",
		"project":   "custom-steps-test",
		"promotion": "test-promotion",
		"stage":     "test",
		"targetFreight": map[string]any{
			"name":   "0123456789abcdef0123456789abcdef01234567",
			"alias":  "test-freight",
			"origin": map[string]any{"name": "test-warehouse"},
		},
		"meta": map[string]any{
			"promotion": map[string]any{"actor": "admin", "rollback": false},
			"step":      map[string]any{"alias": defaultFixtureStep},
		},
	}
	mergeInto(ctx, override)
	return ctx
}

func mergeInto(dst, src map[string]any) {
	for k, v := range src {
		if sv, ok := v.(map[string]any); ok {
			if dv, ok := dst[k].(map[string]any); ok {
				mergeInto(dv, sv)
				continue
			}
		}
		dst[k] = v
	}
}

func renderAll(templates []string, env map[string]any) ([]string, error) {
	out := make([]string, 0, len(templates))
	for _, t := range templates {
		v, err := render(t, env)
		if err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, nil
}

// render matches the executor: EvaluateTemplate, then fmt.Sprint.
func render(template string, env map[string]any) (string, error) {
	v, err := expressions.EvaluateTemplate(template, env)
	if err != nil {
		return "", err
	}
	return fmt.Sprint(v), nil
}

func collectOutput(s step, container, stdout, stderr, pipePath string) (map[string]any, error) {
	srcType, srcPath, format := "Pipe", "", "KeyValue"
	var transform map[string]string
	if o := s.Spec.Output; o != nil {
		srcType, srcPath, format, transform = o.Source.Type, o.Source.Path, o.Source.Format, o.Transform
		if srcType == "" {
			srcType = "Pipe"
		}
		if format == "" {
			switch srcType {
			case "Pipe":
				format = "KeyValue"
			case "File":
				format = inferFormat(srcPath)
			default:
				format = "Text"
			}
		}
	}

	var raw string
	switch srcType {
	case "Stdout":
		raw = stdout
	case "Stderr":
		raw = stderr
	case "Pipe":
		data, err := os.ReadFile(pipePath)
		if err != nil {
			return nil, fmt.Errorf("failed to read output pipe file: %w", err)
		}
		raw = string(data)
	case "File":
		data, err := exec.Command("docker", "cp", container+":"+srcPath, "-").Output()
		if err != nil {
			return nil, fmt.Errorf("failed to read output file %q: %w", srcPath, err)
		}
		raw, err = untarSingle(data)
		if err != nil {
			return nil, fmt.Errorf("failed to read output file %q: %w", srcPath, err)
		}
	default:
		return nil, fmt.Errorf("unsupported output source type %q", srcType)
	}

	var parsed any
	var outMap map[string]any
	switch format {
	case "JSON", "YAML":
		var v any
		var err error
		if format == "JSON" {
			err = json.Unmarshal([]byte(raw), &v)
		} else {
			err = yaml.Unmarshal([]byte(raw), &v)
		}
		if err != nil {
			return nil, fmt.Errorf("failed to parse %s output: %w", format, err)
		}
		if m, ok := v.(map[string]any); ok {
			outMap, parsed = m, m
		} else {
			outMap, parsed = map[string]any{outputKey: v}, v
		}
	case "KeyValue":
		m, err := parseKeyValue(raw)
		if err != nil {
			return nil, err
		}
		outMap, parsed = m, m
	case "Text":
		outMap, parsed = map[string]any{outputKey: raw}, raw
	default:
		return nil, fmt.Errorf("unsupported output format %q", format)
	}

	if len(transform) > 0 {
		env := map[string]any{outputKey: parsed}
		outMap = make(map[string]any, len(transform))
		for k, e := range transform {
			program, err := expr.Compile(e, expr.AllowUndefinedVariables(), expr.AsAny())
			if err != nil {
				return nil, fmt.Errorf("failed to compile transform expression for key %q: %w", k, err)
			}
			v, err := expr.Run(program, env)
			if err != nil {
				return nil, fmt.Errorf("failed to evaluate transform expression for key %q: %w", k, err)
			}
			outMap[k] = v
		}
	}
	encoded, err := json.Marshal(outMap)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal output: %w", err)
	}
	if len(encoded) > maxOutputBytes {
		return nil, fmt.Errorf("output size %d bytes exceeds the 256 KiB limit", len(encoded))
	}
	// Round-trip through JSON so fixture comparisons see plain JSON types.
	var normalized map[string]any
	_ = json.Unmarshal(encoded, &normalized)
	return normalized, nil
}

// parseKeyValue follows the GitHub Actions GITHUB_OUTPUT format, like the executor.
func parseKeyValue(content string) (map[string]any, error) {
	result := map[string]any{}
	var key, eof string
	var value strings.Builder
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimRight(line, "\r")
		if eof == "" {
			if line == "" {
				continue
			}
			if k, v, ok := strings.Cut(line, "="); ok {
				result[k] = v
			} else if k, tag, ok := strings.Cut(line, "<<"); ok {
				key, eof = k, tag
				value.Reset()
			}
			continue
		}
		if strings.TrimSpace(line) == eof {
			result[key] = value.String()
			eof = ""
			continue
		}
		value.WriteString(line)
		value.WriteByte('\n')
	}
	if eof != "" {
		return nil, fmt.Errorf("unexpected end of content when scanning multiline value %q", key)
	}
	return result, nil
}

func inferFormat(path string) string {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".json":
		return "JSON"
	case ".yaml", ".yml":
		return "YAML"
	default:
		return "KeyValue"
	}
}

func check(f fixture, status string, output map[string]any, message string) error {
	var problems []string
	want := f.Expect.Status
	if want == "" {
		want = statusSucceeded
	}
	if status != want {
		problems = append(problems, fmt.Sprintf("status = %s, want %s\n  message: %s", status, want, message))
	}
	for k, wantV := range f.Expect.Output {
		gotV, ok := output[k]
		if !ok {
			problems = append(problems, fmt.Sprintf("output key %q missing", k))
			continue
		}
		if !jsonEqual(gotV, wantV) {
			problems = append(problems, fmt.Sprintf("output %q = %v, want %v", k, gotV, wantV))
		}
	}
	for _, k := range f.Expect.OutputKeys {
		if _, ok := output[k]; !ok {
			problems = append(problems, fmt.Sprintf("output key %q missing", k))
		}
	}
	for _, sub := range f.Expect.MessageContains {
		if !strings.Contains(message, sub) {
			problems = append(problems, fmt.Sprintf("message does not contain %q\n  message: %s", sub, message))
		}
	}
	if len(problems) > 0 {
		return errors.New(strings.Join(problems, "\n"))
	}
	return nil
}

func jsonEqual(a, b any) bool {
	ja, _ := json.Marshal(a)
	jb, _ := json.Marshal(b)
	var na, nb any
	_ = json.Unmarshal(ja, &na)
	_ = json.Unmarshal(jb, &nb)
	return reflect.DeepEqual(na, nb)
}

func buildMessage(prefix, stdout, stderr string) string {
	msg := prefix
	if stdout != "" {
		msg += fmt.Sprintf(" (stdout: %v)", truncateLast(stdout, maxMessageBytes))
	}
	if stderr != "" {
		msg += fmt.Sprintf(" (stderr: %v)", truncateLast(stderr, maxMessageBytes))
	}
	return msg
}

func truncateLast(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[len(s)-n:]
}

var ansiRegex = regexp.MustCompile(`\x1b(?:\[[0-?]*[ -\/]*[@-~]|\][^\x07\x1b]*\x07|[^\[\]])`)

func sanitize(s string) string {
	return ansiRegex.ReplaceAllString(strings.ToValidUTF8(s, ""), "")
}

func readYAML(path string, v any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if err := yaml.Unmarshal(data, v); err != nil {
		return fmt.Errorf("%s: %w", path, err)
	}
	return nil
}

func nonNil(m map[string]any) map[string]any {
	if m == nil {
		return map[string]any{}
	}
	return m
}

func sortedKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func copyPath(src, dst string) error {
	info, err := os.Stat(src)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return copyFile(src, dst)
	}
	return filepath.WalkDir(src, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(src, p)
		target := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o777)
		}
		return copyFile(p, target)
	})
}

func copyFile(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o777); err != nil {
		return err
	}
	return os.WriteFile(dst, data, 0o666)
}

func chmodAll(root string) error {
	return filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return os.Chmod(p, 0o777)
		}
		return os.Chmod(p, 0o666)
	})
}
