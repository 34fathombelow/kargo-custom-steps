// Command lint checks the repository conventions that Argo CD and Kargo
// will not catch until something breaks at promotion time:
//
//   - kargo/ holds only Kargo control-plane resources, never Secrets
//   - every demo Stage and Warehouse is pinned to one shard (spec.shard and label agree,
//     and every demo uses the same shard; scripts/fork.sh changes it)
//   - every demo resource lives in its group's custom-steps-<group> namespace
//   - every CustomPromotionStep uses an image this repo builds, at the tag its
//     Dockerfile declares, and never templates ${{ }} into a script body
//   - every `uses:` names a built-in step or a step defined in this repo, and
//     every `task:` names a PromotionTask in the same project
//   - every step has a README and at least two test fixtures
//   - no file tells anyone to run kubectl or the kargo CLI: everything is GitOps
//
// Usage: lint -repo <repo root>
package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"go.yaml.in/yaml/v3"
)

const (
	nsPrefix        = "custom-steps-"
	imageRegistry   = "ghcr.io/34fathombelow/"
	projectLabel    = "kargo.akuity.io/project"
	shardLabel      = "kargo.akuity.io/shard"
)

// builtinSteps are the step kinds that ship with Kargo CE and Kargo Enterprise.
// A custom step must never reuse one of these names (the built-in wins and the
// custom step is silently ignored), and `uses:` may reference any of them.
var builtinSteps = set(
	// Kargo CE
	"argocd-update", "argocd-wait", "compose-output", "copy", "delete", "fail", "file-write",
	"git-clear", "git-clone", "git-commit", "git-merge-pr", "git-open-pr", "git-push", "git-tag",
	"git-wait-for-pr", "github-push", "helm-template", "helm-update-chart", "http", "http-download",
	"json-parse", "json-update", "kustomize-build", "kustomize-set-image", "oci-download", "oci-push",
	"set-freight-alias", "set-metadata", "tar", "toml-parse", "toml-update", "untar", "yaml-merge",
	"yaml-parse", "yaml-update",
	// Kargo Enterprise
	"get-user-input", "gh-create-issue", "gh-issue-add-comment", "gh-issue-delete-comment",
	"gh-issue-update-comment", "gh-search-issues", "gh-update-issue", "gh-wait-for-issue-state",
	"gha-dispatch-workflow", "gha-wait-for-workflow", "hcl-update", "jfrog-evidence", "jira",
	"record-audit-event", "send-message", "snow-create", "snow-delete", "snow-query-records",
	"snow-update", "snow-wait-for-condition", "tf-apply", "tf-output", "tf-plan", "wait-for-approval",
)

var projectKinds = set("Namespace", "Project", "ProjectConfig", "Warehouse", "Stage", "PromotionTask")

type object struct {
	file string
	doc  map[string]any
}

func (o object) kind() string      { return str(o.doc, "kind") }
func (o object) name() string      { return str(o.doc, "metadata", "name") }
func (o object) namespace() string { return str(o.doc, "metadata", "namespace") }

type linter struct {
	repo     string
	problems []string
	shards   map[string]string // shard -> first file that used it
}

func (l *linter) errorf(file, format string, args ...any) {
	rel, _ := filepath.Rel(l.repo, file)
	l.problems = append(l.problems, fmt.Sprintf("%s: %s", rel, fmt.Sprintf(format, args...)))
}

func main() {
	repo := flag.String("repo", ".", "repository root")
	flag.Parse()
	l := &linter{repo: *repo, shards: map[string]string{}}
	if err := l.run(); err != nil {
		fmt.Fprintln(os.Stderr, "lint:", err)
		os.Exit(2)
	}
	if len(l.problems) > 0 {
		sort.Strings(l.problems)
		for _, p := range l.problems {
			fmt.Println(p)
		}
		fmt.Printf("\n%d problem(s)\n", len(l.problems))
		os.Exit(1)
	}
	fmt.Println("lint: OK")
}

func (l *linter) run() error {
	steps, err := l.loadDir(filepath.Join(l.repo, "kargo", "custom-steps"))
	if err != nil {
		return err
	}
	customSteps := map[string]bool{}
	for _, o := range steps {
		if o.kind() != "CustomPromotionStep" {
			l.errorf(o.file, "only CustomPromotionStep belongs under kargo/custom-steps, found %s", o.kind())
			continue
		}
		if builtinSteps[o.name()] {
			l.errorf(o.file, "step %q collides with a built-in step and would be ignored", o.name())
		}
		customSteps[o.name()] = true
		l.checkStep(o)
	}

	if err := l.checkNoCLI(); err != nil {
		return err
	}

	projectsDir := filepath.Join(l.repo, "kargo", "projects")
	entries, err := os.ReadDir(projectsDir)
	if err != nil {
		return err
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		objs, err := l.loadDir(filepath.Join(projectsDir, e.Name()))
		if err != nil {
			return err
		}
		l.checkProject(e.Name(), objs, customSteps)
	}
	return nil
}

func (l *linter) loadDir(dir string) ([]object, error) {
	var objs []object
	err := filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		ext := filepath.Ext(p)
		if ext != ".yaml" && ext != ".yml" {
			return nil
		}
		data, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		dec := yaml.NewDecoder(bytes.NewReader(data))
		for {
			var doc map[string]any
			if err := dec.Decode(&doc); err != nil {
				if errors.Is(err, io.EOF) {
					break
				}
				l.errorf(p, "invalid YAML: %v", err)
				break
			}
			if doc != nil {
				objs = append(objs, object{file: p, doc: doc})
			}
		}
		return nil
	})
	return objs, err
}

func (l *linter) checkStep(o object) {
	name := o.name()
	base := strings.TrimSuffix(filepath.Base(o.file), filepath.Ext(o.file))
	if base != name || filepath.Base(filepath.Dir(o.file)) != name {
		l.errorf(o.file, "step %q must live at kargo/custom-steps/<group>/%s/%s.yaml", name, name, name)
	}
	image := str(o.doc, "spec", "image")
	if !strings.HasPrefix(image, imageRegistry) {
		l.errorf(o.file, "image %q must be built by this repo (%s<tool>:<tag>)", image, imageRegistry)
	} else {
		tool, tag, _ := strings.Cut(strings.TrimPrefix(image, imageRegistry), ":")
		cmd := exec.Command(filepath.Join(l.repo, "scripts", "image-tag.sh"), tool)
		cmd.Dir = l.repo
		want, err := cmd.Output()
		if err != nil {
			l.errorf(o.file, "image %q has no docker/%s/Dockerfile with ARG %s_VERSION", image, tool, strings.ToUpper(tool))
		} else if strings.TrimSpace(string(want)) != tag {
			l.errorf(o.file, "image tag %q does not match docker/%s (%s)", tag, tool, strings.TrimSpace(string(want)))
		}
	}
	command, _ := get(o.doc, "spec", "command").([]any)
	if len(command) == 0 {
		l.errorf(o.file, "spec.command is empty")
	}
	for _, c := range command {
		s, _ := c.(string)
		if strings.Contains(s, "\n") && (strings.Contains(s, "${{") || strings.Contains(s, "${%")) {
			l.errorf(o.file, "script body contains a template expression; pass values through spec.env instead")
		}
	}
	env, _ := get(o.doc, "spec", "env").([]any)
	for _, e := range env {
		m, _ := e.(map[string]any)
		v, _ := m["value"].(string)
		if strings.Contains(v, "asJSON(") && !(strings.HasPrefix(v, "${{") && strings.HasSuffix(v, "}}")) {
			l.errorf(o.file, "env %v: asJSON() must be the entire value or it leaves stray quotes", m["name"])
		}
	}
	dir := filepath.Dir(o.file)
	if _, err := os.Stat(filepath.Join(dir, "README.md")); err != nil {
		l.errorf(o.file, "missing README.md next to the step")
	}
	fixtures, _ := filepath.Glob(filepath.Join(l.repo, "test", "fixtures", name, "*.yaml"))
	if len(fixtures) < 2 {
		l.errorf(o.file, "needs at least two fixtures in test/fixtures/%s/ (one passing, one failing)", name)
	}
}

func (l *linter) checkProject(dir string, objs []object, customSteps map[string]bool) {
	ns := nsPrefix + dir
	tasks := map[string]bool{}
	for _, o := range objs {
		if o.kind() == "PromotionTask" {
			tasks[o.name()] = true
		}
	}
	var sawNamespace, sawProject bool
	for _, o := range objs {
		kind := o.kind()
		switch {
		case kind == "Secret":
			l.errorf(o.file, "Secrets must never be synced from git; create them in the Kargo UI and document them in secrets/README.md")
			continue
		case !projectKinds[kind]:
			l.errorf(o.file, "%s does not belong in a Kargo demo project", kind)
			continue
		}
		switch kind {
		case "Namespace":
			sawNamespace = true
			if o.name() != ns {
				l.errorf(o.file, "Namespace must be named %s", ns)
			}
			if str(o.doc, "metadata", "labels", projectLabel) != "true" {
				l.errorf(o.file, "Namespace needs label %s: \"true\" so the Project adopts it", projectLabel)
			}
		case "Project":
			sawProject = true
			if o.name() != ns {
				l.errorf(o.file, "Project must be named %s", ns)
			}
		default:
			if o.namespace() != ns {
				l.errorf(o.file, "%s/%s must be in namespace %s", kind, o.name(), ns)
			}
		}
		if kind == "Stage" || kind == "Warehouse" {
			specShard, labelShard := str(o.doc, "spec", "shard"), str(o.doc, "metadata", "labels", shardLabel)
			switch {
			case specShard == "" || specShard != labelShard:
				l.errorf(o.file, "%s/%s must set spec.shard and label %s to the same shard", kind, o.name(), shardLabel)
			case len(l.shards) > 0 && l.shards[specShard] == "":
				l.errorf(o.file, "%s/%s uses shard %q but other demos use another; run scripts/fork.sh --shard <name>", kind, o.name(), specShard)
			default:
				if l.shards[specShard] == "" {
					l.shards[specShard] = o.file
				}
			}
		}
		l.checkStepRefs(o, customSteps, tasks)
	}
	if !sawNamespace || !sawProject {
		l.errorf(filepath.Join(l.repo, "kargo", "projects", dir), "needs project.yaml with a Namespace and a Project named %s", ns)
	}
}

// checkStepRefs validates `uses:` and `task:` references in PromotionTasks
// and Stage promotion templates.
func (l *linter) checkStepRefs(o object, customSteps, tasks map[string]bool) {
	var steps []any
	switch o.kind() {
	case "PromotionTask":
		steps, _ = get(o.doc, "spec", "steps").([]any)
	case "Stage":
		steps, _ = get(o.doc, "spec", "promotionTemplate", "spec", "steps").([]any)
	}
	for _, s := range steps {
		m, _ := s.(map[string]any)
		if uses, ok := m["uses"].(string); ok && !builtinSteps[uses] && !customSteps[uses] {
			l.errorf(o.file, "%s/%s uses unknown step %q", o.kind(), o.name(), uses)
		}
		if task, ok := m["task"].(map[string]any); ok {
			name, _ := task["name"].(string)
			if kind, _ := task["kind"].(string); kind != "ClusterPromotionTask" && !tasks[name] {
				l.errorf(o.file, "%s/%s references PromotionTask %q, which is not in this project", o.kind(), o.name(), name)
			}
		}
	}
}

func get(m map[string]any, path ...string) any {
	var cur any = m
	for _, p := range path {
		mm, ok := cur.(map[string]any)
		if !ok {
			return nil
		}
		cur = mm[p]
	}
	return cur
}

func str(m map[string]any, path ...string) string {
	s, _ := get(m, path...).(string)
	return s
}

func set(items ...string) map[string]bool {
	m := make(map[string]bool, len(items))
	for _, i := range items {
		m[i] = true
	}
	return m
}

// cliCommand matches an instruction to run kubectl or the kargo CLI. Nobody has
// access to the Kargo control plane, so everything must happen through Argo CD
// and the Kargo / Argo CD UIs.
var cliCommand = regexp.MustCompile(
	`\bkubectl\s+(apply|create|delete|patch|scale|get|edit|exec|rollout|config|label|annotate|replace|run|cp|logs|wait|port-forward|-n|--)` +
		`|\bkargo\s+(promote|apply|create|delete|login|get|update|approve|refresh|verify)\b`)

// Local working files that are never published.
var cliCheckSkip = map[string]bool{".git": true, ".work": true, ".claude": true, "CLAUDE.md": true, "PLAN.md": true}

func (l *linter) checkNoCLI() error {
	return filepath.WalkDir(l.repo, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if cliCheckSkip[d.Name()] {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if d.IsDir() {
			return nil
		}
		data, err := os.ReadFile(p)
		if err != nil || bytes.IndexByte(data, 0) >= 0 { // skip unreadable and binary files
			return nil
		}
		for i, line := range strings.Split(string(data), "\n") {
			if m := cliCommand.FindString(line); m != "" {
				l.errorf(p, "line %d runs %q: use GitOps (argocd/apps) or the Argo CD / Kargo UI instead", i+1, strings.TrimSpace(m))
			}
		}
		return nil
	})
}
