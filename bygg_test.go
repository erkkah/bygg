package main

import (
	"bytes"
	"fmt"
	"net"
	"net/http"
	"os"
	"runtime"
	"strings"
	"testing"
	"time"
)

var capture bytes.Buffer

func loadTestBuild(t *testing.T, file string) *bygge {
	var cfg = config{
		byggFil: file,
		baseDir: "tests",
	}
	b, err := newBygge(cfg)
	if err != nil {
		t.Fatal(err)
	}
	b.output = &capture
	capture.Reset()
	return b
}

func runTestBuild(t *testing.T, file string, target string) string {
	b := loadTestBuild(t, file)
	err := b.buildTarget(target)
	if err != nil {
		t.Fatal(err)
	}
	return string(capture.Bytes())
}

func verifyTestOutput(t *testing.T, file string, target string, expected string) {
	output := runTestBuild(t, file, target)
	if output != expected {
		t.Errorf("Expected: %q, got: %q", expected, output)
	}
}

func TestEmptyBuild(t *testing.T) {
	b := loadTestBuild(t, "empty.bygg")
	err := b.buildTarget("all")
	if err == nil {
		t.Fail()
	}
}

func TestLogging_Plain(t *testing.T) {
	verifyTestOutput(
		t, "logging.bygg", "A",
		"message:\n4711\n",
	)
}

func TestLogging_Newlines(t *testing.T) {
	verifyTestOutput(
		t, "logging.bygg", "B",
		"message:\n1\n2\n3\n",
	)
}

func TestVariables_A(t *testing.T) {
	verifyTestOutput(
		t, "variables.bygg", "A",
		"hello world\n",
	)
}

func TestVariables_B(t *testing.T) {
	verifyTestOutput(
		t, "variables.bygg", "B",
		"hello worldwide\n",
	)
}

func TestVariables_C(t *testing.T) {
	verifyTestOutput(
		t, "variables.bygg", "C",
		"I say: hello world to you all!\n",
	)
}

func TestEnvironmentVariable(t *testing.T) {
	home := os.Getenv("HOME")
	expected := home + " is where the heart is\n"
	verifyTestOutput(
		t, "variables.bygg", "D",
		expected,
	)
}

func TestEmptyEnvironmentVariable(t *testing.T) {
	verifyTestOutput(
		t, "variables.bygg", "E",
		"\n",
	)
}

func TestDependencyChain_A(t *testing.T) {
	verifyTestOutput(
		t, "dependencies.bygg", "A",
		"Här\noch\ndär\n",
	)
}

func TestDependencyVariableTarget_X(t *testing.T) {
	verifyTestOutput(
		t, "dependencies.bygg", "X",
		"bullseye\n",
	)
}

func TestForcedDependency(t *testing.T) {
	verifyTestOutput(
		t, "dependencies.bygg", "Forced",
		"Forced\n",
	)
}

func TestFileDependency(t *testing.T) {
	verifyTestOutput(
		t, "dependencies.bygg", "dependencies.bygg",
		"",
	)
}

func TestBuildCommand(t *testing.T) {
	output := runTestBuild(t, "buildcommands.bygg", "help")
	if !strings.Contains(output, "SWIG") {
		t.Fail()
	}
}

func TestBuildCommand_Download(t *testing.T) {
	listener, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		t.Error(err)
	}
	defer listener.Close()
	os.Setenv("BYGG_TEST_ADDR", listener.Addr().String())

	go http.Serve(listener, http.FileServer(http.Dir(".")))

	os.RemoveAll("tests/download")
	defer os.RemoveAll("tests/download")

	runTestBuild(t, "buildcommands.bygg", "download")
	_, err = os.Stat("tests/download/doubt.txt")
	if err != nil {
		t.Error(err)
	}
}

func TestChildBuild(t *testing.T) {
	verifyTestOutput(
		t, "buildcommands.bygg", "child",
		"I am a child\n",
	)
}

func TestTemplates_split(t *testing.T) {
	verifyTestOutput(
		t, "templates.bygg", "split",
		"[b]\n",
	)
}

func TestTemplates_exec(t *testing.T) {
	expected := fmt.Sprintf("[%s]\nOK\n", runtime.Version())
	verifyTestOutput(
		t, "templates.bygg", "exec",
		expected,
	)
}

func TestTemplates_date(t *testing.T) {
	expected := time.Now().Format("2006")
	verifyTestOutput(
		t, "templates.bygg", "date",
		expected+"\n",
	)
}

func TestTemplates_glob(t *testing.T) {
	expected := "templates.bygg\n"
	verifyTestOutput(
		t, "templates.bygg", "glob",
		expected,
	)
}
