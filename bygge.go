package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"text/template"
	"time"
)

type bygge struct {
	lastError error
	output    io.Writer

	targets map[string]target
	vars    map[string]string
	env     map[string]string
	visited map[string]bool
	tmpl    *template.Template

	cfg config
}

type target struct {
	name          string
	buildCommands []string
	dependencies  []string
	resolved      bool
	force         bool
	modifiedAt    time.Time
}

func verifyVersion(byggFile string) error {
	if Tag == "" {
		return nil
	}

	f, err := os.Open(byggFile)
	if err != nil {
		return err
	}
	scanner := bufio.NewScanner(f)
	scanner.Scan()
	firstLine := scanner.Text()
	if strings.HasPrefix(firstLine, "##") {
		firstLine = strings.TrimSpace(firstLine[2:])
		if strings.HasPrefix(firstLine, "bygg:") {
			ok, err := isVersionCompatible(firstLine, Tag)
			if err != nil {
				return fmt.Errorf("failed to check file version: %w", err)
			}
			if !ok {
				return fmt.Errorf("Incompatible bygg version %q, required %q", Tag, firstLine)
			}
		}
	}
	return nil
}

func newBygge(cfg config) (*bygge, error) {
	pwd, _ := os.Getwd()
	if err := os.Chdir(cfg.baseDir); err != nil {
		return nil, err
	}
	defer os.Chdir(pwd)

	result := &bygge{
		targets: map[string]target{},
		vars:    builtins,
		env:     map[string]string{},
		visited: map[string]bool{},
		output:  os.Stdout,
		cfg:     cfg,
	}

	if err := verifyVersion(cfg.byggFil); err != nil {
		return nil, err
	}

	for _, pair := range os.Environ() {
		parts := strings.SplitN(pair, "=", 2)
		result.env[parts[0]] = parts[1]
	}

	genExec := func(b *bygge, validate bool) func(string, ...interface{}) (string, error) {
		return func(prog string, args ...interface{}) (string, error) {
			argStrings := []string{}
			for _, arg := range args {
				switch v := arg.(type) {
				case string:
					argStrings = append(argStrings, v)
				case []string:
					argStrings = append(argStrings, v...)
				default:
					return "", fmt.Errorf("unsupported arg: %q", v)
				}
			}
			for i, v := range argStrings {
				argStrings[i] = strings.TrimSpace(v)
			}
			cmd := exec.Command(prog, argStrings...)
			cmd.Env = b.envList()
			var output []byte
			output, b.lastError = cmd.Output()
			if b.lastError != nil && validate {
				if exitError, ok := b.lastError.(*exec.ExitError); ok {
					b.verbose("Template executed %v %v, result=%s", prog, argStrings, string(exitError.Stderr))
				}
				return "", b.lastError
			}
			b.verbose("Template executed %v %v, result=%v", prog, argStrings, b.lastError)
			return string(output), nil
		}
	}

	getFunctions := func(b *bygge) template.FuncMap {
		return template.FuncMap{
			"env": func(args ...string) (interface{}, error) {
				switch len(args) {
				case 0:
					return b.env, nil
				case 1:
					return b.env[args[0]], nil
				case 2:
					value := args[1]
					b.env[args[0]] = value
					return value, nil
				default:
					return "", fmt.Errorf("Too many arguments to 'env'")
				}
			},
			"exec":     genExec(b, false),
			"mustexec": genExec(b, true),
			"ok": func() bool {
				return b.lastError == nil
			},
			"date": func(layout string) string {
				return time.Now().Format(layout)
			},
			"split": func(unsplit string, splitArg ...string) []string {
				if len(splitArg) > 0 {
					return strings.Split(unsplit, splitArg[0])
				}
				return strings.Fields(unsplit)
			},
			"join": func(array []string, joinArg ...string) string {
				joiner := " "
				if len(joinArg) > 0 {
					joiner = joinArg[0]
				}
				return strings.Join(array, joiner)
			},
			"glob": func(patterns ...string) []string {
				result := []string{}
				for _, pattern := range patterns {
					if matches, err := filepath.Glob(pattern); err == nil {
						result = append(result, matches...)
					}
				}
				return result
			},
			"replace": func(pattern, replacement string, operands interface{}) interface{} {
				re, err := regexp.Compile(pattern)
				if err != nil {
					return nil
				}
				if one, ok := operands.(string); ok {
					return re.ReplaceAllString(one, replacement)
				}
				if many, ok := operands.([]string); ok {
					replaced := make([]string, len(many))
					for i, operand := range many {
						replaced[i] = re.ReplaceAllString(operand, replacement)
					}
					return replaced
				}
				return ""
			},
		}
	}

	result.tmpl = template.New(cfg.byggFil)
	result.tmpl.Funcs(getFunctions(result))

	result.verbose("Parsing template")
	if !exists(cfg.byggFil) {
		return nil, fmt.Errorf("bygg file %q not found", cfg.byggFil)
	}
	var err error

	if result.tmpl, err = result.tmpl.ParseFiles(cfg.byggFil); err != nil {
		return nil, fmt.Errorf("failed to parse templates: %w", err)
	}
	return result, nil
}

func (b *bygge) buildTarget(tgt string) error {
	pwd, _ := os.Getwd()
	if err := os.Chdir(b.cfg.baseDir); err != nil {
		return err
	}
	defer os.Chdir(pwd)

	data := map[string]interface{}{
		"env": b.env,
	}

	addBuiltins(data)

	b.verbose("Executing template")
	var buf bytes.Buffer
	if err := b.tmpl.Execute(&buf, data); err != nil {
		return err
	}

	if b.cfg.veryVerbose {
		b.verbose(fmt.Sprintf("Script:[\n%s\n]", string(buf.Bytes())))
	}

	var joined bytes.Buffer
	lineJoiner := strings.NewReplacer("\\\n", "")
	lineJoiner.WriteString(&joined, buf.String())

	b.verbose("Loading build script")
	if err := b.loadBuildScript(&joined); err != nil {
		return err
	}

	if tgt, ok := b.targets[tgt]; ok {
		for {
			err := b.resolve(tgt)
			if b.cfg.watch {
				b.unresolve()
				if err != nil {
					fmt.Printf("%v\n", err)
				}
				fmt.Println("Waiting for changes")
				err = b.waitForChange(tgt)
				if err != nil {
					return err
				}
				fmt.Println("Detected change, rebuilding...")
			} else {
				return err
			}
		}
	}

	return fmt.Errorf("no such target %q", tgt)
}

func (b *bygge) loadBuildScript(scriptSource io.Reader) error {
	scanner := bufio.NewScanner(scriptSource)

	// Handle dependencies, build commands and assignments, with
	// or without spaces around the operators.
	//
	// Examples:
	// all: foo splat
	// all <- gcc -o all all.c
	// bar=baz
	// bar += yes
	commandExp := regexp.MustCompile(`([\w._\-/${}]+)\s*([:=]|\+=|<-|<<)\s*(.*)`)

	for scanner.Scan() {
		line := scanner.Text()
		// Skip initial whitespace
		line = strings.TrimLeft(line, " \t")
		// Skip comments
		if strings.HasPrefix(line, "#") {
			continue
		}
		// Skip empty lines
		if line == "" {
			continue
		}
		// Handle message lines
		if strings.HasPrefix(line, "<<") {
			fmt.Fprintln(b.output, b.expand(strings.Trim(line[2:], " \t")))
			continue
		}

		matches := commandExp.FindStringSubmatch(line)
		if matches == nil {
			return fmt.Errorf("parse error: %q", line)
		}

		lvalue := matches[1]
		operator := matches[2]
		rvalue := matches[3]

		lvalue = b.expand(lvalue)
		rvalue = b.expand(rvalue)

		var err error
		switch operator {
		case ":":
			err = b.handleDependencies(lvalue, rvalue)
		case "=":
			err = b.handleAssignment(lvalue, rvalue, false)
		case "+=":
			err = b.handleAssignment(lvalue, rvalue, true)
		case "<<":
			rvalue = operator + " " + rvalue
			fallthrough
		case "<-":
			b.handleBuildCommand(lvalue, rvalue)
		default:
			return fmt.Errorf("unexpected operator %q", operator)
		}

		if err != nil {
			return err
		}
	}

	return nil
}

func (b *bygge) handleDependencies(lvalue, rvalue string) error {
	clean := cleanPaths(lvalue)[0]
	t := b.targets[clean]
	t.name = clean
	rvalue = strings.TrimLeft(rvalue, " \t")
	if strings.HasPrefix(rvalue, "!") {
		t.force = true
		rvalue = strings.TrimLeft(rvalue, "!")
	}
	dependencies, err := splitQuoted(rvalue)
	if err != nil {
		return err
	}
	dependencies = cleanPaths(dependencies...)
	t.dependencies = append(t.dependencies, dependencies...)
	b.targets[clean] = t

	return nil
}

func (b *bygge) handleAssignment(lvalue, rvalue string, add bool) error {
	if strings.Contains(lvalue, ".") {
		parts := strings.SplitN(lvalue, ".", 2)
		context := parts[0]
		name := parts[1]
		if context == "env" {
			if oldValue, isSet := b.env[name]; isSet && add {
				rvalue = oldValue + " " + rvalue
			}
			b.env[name] = rvalue
		} else {
			return fmt.Errorf("unknown variable context %q", context)
		}
	} else {
		if add {
			rvalue = b.vars[lvalue] + " " + rvalue
		}
		b.vars[lvalue] = rvalue
	}

	return nil
}

func (b *bygge) handleBuildCommand(lvalue, rvalue string) {
	clean := cleanPaths(lvalue)[0]
	t := b.targets[clean]
	t.name = clean
	t.buildCommands = append(t.buildCommands, rvalue)
	b.targets[clean] = t
}

// Permissive variable expansion
func (b *bygge) expand(expr string) string {
	return os.Expand(expr, func(varExpr string) string {
		varExpr = strings.Trim(varExpr, " \t")
		if strings.Contains(varExpr, ".") {
			parts := strings.SplitN(varExpr, ".", 2)
			context := parts[0]
			name := parts[1]

			if context == "env" {
				if local, ok := b.env[name]; ok {
					return local
				}
			}
			return ""
		}
		return b.vars[varExpr]
	})
}

func (b *bygge) resolve(t target) error {
	if t.resolved {
		return nil
	}

	b.verbose("Resolving target %q", t.name)
	if b.visited[t.name] {
		return fmt.Errorf("cyclic dependency resolving %q", t.name)
	}
	b.visited[t.name] = true
	defer func() {
		b.visited[t.name] = false
	}()

	dependencies := t.dependencies

	var mostRecentUpdate time.Time

	for _, depName := range dependencies {
		dep, ok := b.targets[depName]
		if !ok {
			if exists(depName) {
				dep = target{
					name: depName,
				}
			} else {
				return fmt.Errorf("target %q has unknown dependency %q", t.name, depName)
			}
		}
		if err := b.resolve(dep); err != nil {
			return err
		}
		dep = b.targets[depName]
		if dep.modifiedAt.After(mostRecentUpdate) {
			mostRecentUpdate = dep.modifiedAt
		}
	}

	if t.force || !exists(t.name) || getFileDate(t.name).Before(mostRecentUpdate) {
		if len(t.buildCommands) == 0 {
			b.verbose("No build command for target %q, skipping build", t.name)
		}
		for _, cmd := range t.buildCommands {
			if err := b.runBuildCommand(t.name, cmd); err != nil {
				return err
			}
		}
	}

	t.resolved = true

	if exists(t.name) {
		t.modifiedAt = getFileDate(t.name)
	} else {
		t.modifiedAt = time.Now()
	}

	b.targets[t.name] = t

	return nil
}

func (b *bygge) runBuildCommand(tgt, command string) error {
	if b.cfg.dryRun {
		fmt.Printf("Not running command %q\n", command)
		return nil
	}
	parts, err := splitQuoted(command)
	if err != nil {
		return err
	}
	prog := parts[0]
	args := parts[1:]
	b.verbose("Running command %q with args %v", prog, args)
	if prog == "<<" {
		fmt.Fprintln(b.output, strings.Join(args, " "))
		return nil
	}
	if prog == "bygg" {
		cfg, err := parseConfig(args)
		if err != nil {
			return err
		}
		bb, err := newBygge(cfg)
		if err != nil {
			return err
		}
		bb.output = b.output
		return bb.buildTarget(cfg.target)
	}
	if strings.HasPrefix(prog, "http") {
		return b.handleDownload(tgt, prog, args...)
	}
	if strings.HasPrefix(prog, "clean:") {
		return b.handleClean(prog, args...)
	}
	if strings.HasPrefix(prog, "mkdir:") {
		return b.handleMakeDir(tgt, prog, args...)
	}
	if strings.HasPrefix(prog, "copy:") {
		return b.handleCopy(tgt, prog, args...)
	}

	cmd := exec.Command(prog, args...)
	cmd.Env = b.envList()
	cmd.Stderr = b.output
	cmd.Stdout = b.output
	err = cmd.Run()
	return err
}

func (b *bygge) envList() []string {
	env := []string{}
	for k, v := range b.env {
		if k == "" {
			continue
		}
		env = append(env, fmt.Sprintf("%s=%s", k, v))
	}
	return env
}

var builtins = map[string]string{
	"GOVERSION": runtime.Version(),
	"GOOS":      runtime.GOOS,
	"GOARCH":    runtime.GOARCH,
}

func addBuiltins(vars map[string]interface{}) {
	for k, v := range builtins {
		vars[k] = v
	}
}

func (b *bygge) verbose(pattern string, args ...interface{}) {
	if b.cfg.verbose {
		fmt.Printf("bygg: "+pattern+"\n", args...)
	}
}
