package testrunner

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBuildJvmArgs(t *testing.T) {
	opts := RunOptions{
		TestJar:     "test.jar",
		Deps:        []string{"dep1.jar", "dep2.jar"},
		TestClass:   "org.example.JUnit5Test",
		JunitRunner: "junit-standalone.jar",
		ExtraArgs:   []string{"arg1", "arg2"},
		JacocoAgent: "jacocoagent.jar",
	}

	args := buildJvmArgs(opts, "/tmp/jacoco.exec", "/tmp/reports")
	joined := strings.Join(args, " ")

	if !strings.Contains(joined, "-javaagent:jacocoagent.jar=destfile=/tmp/jacoco.exec") {
		t.Errorf("missing javaagent arg: %s", joined)
	}
	if !strings.Contains(joined, "junit-standalone.jar") {
		t.Errorf("missing junit runner in classpath: %s", joined)
	}
	if !strings.Contains(joined, "org.junit.platform.console.ConsoleLauncher execute --reports-dir /tmp/reports --select-class org.example.JUnit5Test") {
		t.Errorf("missing ConsoleLauncher invocation: %s", joined)
	}
	if !strings.Contains(joined, "arg1 arg2") {
		t.Errorf("missing extra args: %s", joined)
	}
}

func TestResolveCoverage(t *testing.T) {
	opts := RunOptions{CoverageActive: true}
	active, file := resolveCoverage(opts)
	if !active {
		t.Errorf("expected active = true")
	}
	if file != "test.coverage" {
		t.Errorf("got %q, want test.coverage", file)
	}
}

func TestRunMandatoryTestClass(t *testing.T) {
	err := Run(RunOptions{Java: "java"})
	if err == nil || !strings.Contains(err.Error(), "test-class is mandatory") {
		t.Fatalf("expected mandatory test-class error, got %v", err)
	}
}

func TestRunMockJava(t *testing.T) {
	tmpDir := t.TempDir()
	mockJava := filepath.Join(tmpDir, "fake_java")
	script := "#!/bin/sh\necho 'Running Test1... ok'\nexit 0\n"
	if err := os.WriteFile(mockJava, []byte(script), 0755); err != nil {
		t.Fatalf("failed to write mock script: %v", err)
	}

	resultsFile := filepath.Join(tmpDir, "test.results")
	opts := RunOptions{
		Java:        mockJava,
		TestClass:   "pkg.MyTest",
		ResultsFile: resultsFile,
	}

	if err := Run(opts); err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	if _, err := os.Stat(resultsFile); os.IsNotExist(err) {
		t.Errorf("expected %s to be created", resultsFile)
	}
}
