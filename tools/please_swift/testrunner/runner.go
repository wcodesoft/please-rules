package testrunner

import (
	"bufio"
	"bytes"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"
	"tools/please_swift/linkorder"
)

// JSON Event Stream structs
type StreamRecord struct {
	Kind    string          `json:"kind"`
	Payload json.RawMessage `json:"payload"`
	Version any             `json:"version"`
}

type TestMetadata struct {
	ID              string `json:"id"`
	Name            string `json:"name"`
	DisplayName     string `json:"displayName"`
	Kind            string `json:"kind"`
	IsParameterized bool   `json:"isParameterized"`
}

type paramCaseInfo struct {
	args      string
	startTime float64
	endTime   float64
	issues    []string
}

var testCasePassingRe = regexp.MustCompile(`^Test case passing (?:\d+\s+arguments?\s+)?(.+?)\s+to\s+(?:\"[^\"]+\"|.+?)\s+started\.$`)

func extractTestCaseArgs(msg string) string {
	if m := testCasePassingRe.FindStringSubmatch(msg); len(m) > 1 {
		return strings.TrimSpace(m[1])
	}
	idx := strings.Index(msg, "passing ")
	if idx != -1 {
		rest := msg[idx+len("passing "):]
		if toIdx := strings.LastIndex(rest, " to "); toIdx != -1 {
			args := rest[:toIdx]
			args = regexp.MustCompile(`^\d+\s+arguments?\s+`).ReplaceAllString(args, "")
			return strings.TrimSpace(args)
		}
	}
	return ""
}

func formatSubtestName(testName, args string) string {
	cleanArgs := strings.TrimSpace(args)
	if cleanArgs == "" {
		return testName
	}
	if strings.HasPrefix(cleanArgs, "(") && strings.HasSuffix(cleanArgs, ")") {
		return fmt.Sprintf("%s %s", testName, cleanArgs)
	}
	return fmt.Sprintf("%s (%s)", testName, cleanArgs)
}

type EventPayload struct {
	Kind    string `json:"kind"`
	TestID  string `json:"testID"`
	Instant struct {
		Absolute float64 `json:"absolute"`
	} `json:"instant"`
	Comments []string `json:"_comments"`
	Messages []struct {
		Symbol string `json:"symbol"`
		Text   string `json:"text"`
	} `json:"messages"`
}

// RunOptions configures the execution of the Swift test runner.
type RunOptions struct {
	Swiftc       string
	LlvmProfdata string
	LlvmCov      string
	Framework    string
	Srcs         []string
	Deps         []string
	Out          string
	Binary       string
	CompileOnly  bool
	ResultsFile  string
	Coverage     bool
	CoverageFile string
	LcovFile     string // raw lcov export (functions and branches), written empty without coverage
	TestArgs     []string
	Flags        []string
}

// JUnit XML structs
type JUnitTestSuites struct {
	XMLName  xml.Name         `xml:"testsuites"`
	Suites   []JUnitTestSuite `xml:"testsuite"`
	Tests    int              `xml:"tests,attr"`
	Failures int              `xml:"failures,attr"`
	Errors   int              `xml:"errors,attr"`
	Skipped  int              `xml:"skipped,attr,omitempty"`
	Time     string           `xml:"time,attr"`
}

type JUnitTestSuite struct {
	Name      string          `xml:"name,attr"`
	Tests     int             `xml:"tests,attr"`
	Failures  int             `xml:"failures,attr"`
	Errors    int             `xml:"errors,attr"`
	Skipped   int             `xml:"skipped,attr,omitempty"`
	Time      string          `xml:"time,attr"`
	TestCases []JUnitTestCase `xml:"testcase"`
}

type JUnitTestCase struct {
	Name      string        `xml:"name,attr"`
	ClassName string        `xml:"classname,attr"`
	Time      string        `xml:"time,attr"`
	Skipped   *JUnitSkipped `xml:"skipped,omitempty"`
	Failure   *JUnitFailure `xml:"failure,omitempty"`
}

type JUnitSkipped struct {
	Message string `xml:"message,attr,omitempty"`
}

type JUnitFailure struct {
	Message  string `xml:"message,attr"`
	Type     string `xml:"type,attr,omitempty"`
	Contents string `xml:",chardata"`
}

type ParsedTestCase struct {
	Suite    string
	Name     string
	Duration float64
	Passed   bool
	Skipped  bool
	Failure  string
}

func resolveTool(tool string) string {
	if tool == "" {
		return tool
	}
	if filepath.IsAbs(tool) {
		return tool
	}
	if p, err := exec.LookPath(tool); err == nil {
		return p
	}
	commonDirs := []string{
		"/home/linuxbrew/.linuxbrew/bin",
		"/usr/local/bin",
		"/usr/bin",
		"/opt/swift/usr/bin",
	}
	for _, dir := range commonDirs {
		candidate := filepath.Join(dir, tool)
		if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
			return candidate
		}
	}
	return tool
}

// Run coordinates building, executing, and reporting tests.
func Run(opts RunOptions) error {
	if opts.Swiftc == "" {
		opts.Swiftc = "swiftc"
	}
	opts.Swiftc = resolveTool(opts.Swiftc)
	if opts.LlvmProfdata == "" {
		opts.LlvmProfdata = "llvm-profdata"
	}
	opts.LlvmProfdata = resolveTool(opts.LlvmProfdata)
	if opts.LlvmCov == "" {
		opts.LlvmCov = "llvm-cov"
	}
	opts.LlvmCov = resolveTool(opts.LlvmCov)
	if opts.ResultsFile == "" {
		opts.ResultsFile = "test.results"
	}
	if opts.Framework == "" {
		opts.Framework = "swift-testing"
	}

	coverageActive := opts.Coverage || os.Getenv("COVERAGE") != "" || os.Getenv("COVERAGE_FILE") != ""
	if opts.CoverageFile == "" {
		if covEnv := os.Getenv("COVERAGE_FILE"); covEnv != "" {
			opts.CoverageFile = covEnv
		} else {
			opts.CoverageFile = "test.coverage"
		}
	}

	// The raw lcov export is a declared test output: create it up front so it exists
	// (empty) when coverage is off or the report cannot be produced.
	if !opts.CompileOnly {
		if err := writeRawLcov(opts.LcovFile, nil); err != nil {
			return fmt.Errorf("failed to reset lcov file: %w", err)
		}
	}

	tmpDir, err := os.MkdirTemp("", "please_swift_test_*")
	if err != nil {
		return fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	testBinary := opts.Out
	if testBinary == "" {
		testBinary = filepath.Join(tmpDir, "test_binary")
	}

	var linkIncDirs, linkArchives []string
	if opts.Binary != "" && !coverageActive {
		testBinary = opts.Binary
	} else {
		// 1. Check for @main in source files
		hasMain := false
		for _, src := range opts.Srcs {
			if data, err := os.ReadFile(src); err == nil {
				if strings.Contains(string(data), "@main") {
					hasMain = true
					break
				}
			}
		}

		compileSrcs := append([]string{}, opts.Srcs...)
		extraFlags := append([]string{}, opts.Flags...)

		if !hasMain && opts.Framework == "swift-testing" {
			runnerMainPath := filepath.Join(tmpDir, "__runner_main.swift")
			runnerCode := `import Testing
#if canImport(Glibc)
import Glibc
#elseif canImport(Darwin)
import Darwin
#endif

@main
struct __PleaseTestRunner {
    static func main() async {
        var args = Testing.__CommandLineArguments_v0()
        if let eventsPath = getenv("PLEASE_SWIFT_EVENTS_OUTPUT") {
            args.eventStreamOutputPath = String(cString: eventsPath)
        }
        let exitCode: CInt = await Testing.__swiftPMEntryPoint(passing: args)
        exit(exitCode)
    }
}
`
			if err := os.WriteFile(runnerMainPath, []byte(runnerCode), 0644); err != nil {
				return fmt.Errorf("failed to write test runner main: %w", err)
			}
			compileSrcs = append(compileSrcs, runnerMainPath)
			extraFlags = append(extraFlags, "-parse-as-library", "-lTesting")
		}

		if coverageActive {
			extraFlags = append(extraFlags, "-profile-generate", "-profile-coverage-mapping")
		}

		// 2. Discover dependency paths
		incDirs, archives := discoverDependencies(opts.Deps)
		archives = linkorder.Sort(archives)
		linkIncDirs, linkArchives = incDirs, archives

		var compileArgs []string
		compileArgs = append(compileArgs, "-o", testBinary)
		for _, inc := range incDirs {
			compileArgs = append(compileArgs, "-I", inc)
		}
		compileArgs = append(compileArgs, extraFlags...)
		compileArgs = append(compileArgs, compileSrcs...)

		if runtime.GOOS == "linux" && len(archives) > 0 {
			compileArgs = append(compileArgs, "-Xlinker", "--start-group")
			for _, arch := range archives {
				compileArgs = append(compileArgs, arch)
			}
			compileArgs = append(compileArgs, "-Xlinker", "--end-group")
		} else {
			for _, arch := range archives {
				compileArgs = append(compileArgs, arch)
			}
		}

		var compileErr bytes.Buffer
		cCmd := exec.Command(opts.Swiftc, compileArgs...)
		cCmd.Stderr = &compileErr
		if err := cCmd.Run(); err != nil {
			return fmt.Errorf("failed to compile test executable: %v\n%s", err, compileErr.String())
		}
	}

	if opts.CompileOnly {
		// Keep the transitive link inputs next to the binary: the coverage run
		// rebuilds the test with instrumentation but only has the direct deps staged.
		if opts.Out != "" {
			if err := saveLinkInputs(opts.Out+".deps", linkIncDirs, linkArchives); err != nil {
				return fmt.Errorf("failed to save link inputs: %w", err)
			}
		}
		return nil
	}

	// 3. Execute test binary
	profDir := filepath.Join(tmpDir, "profiles")
	_ = os.MkdirAll(profDir, 0755)
	profPattern := filepath.Join(profDir, "test_%p.profraw")

	eventsPath := filepath.Join(tmpDir, "events.ndjson")
	testCmd := exec.Command(testBinary, opts.TestArgs...)
	testCmd.Env = append(os.Environ(),
		"LLVM_PROFILE_FILE="+profPattern,
		"PLEASE_SWIFT_EVENTS_OUTPUT="+eventsPath,
	)

	var stdoutBuf, stderrBuf bytes.Buffer
	testCmd.Stdout = io.MultiWriter(os.Stdout, &stdoutBuf)
	testCmd.Stderr = io.MultiWriter(os.Stderr, &stderrBuf)

	startTime := time.Now()
	testErr := testCmd.Run()
	elapsed := time.Since(startTime).Seconds()

	fullOutput := stdoutBuf.String() + "\n" + stderrBuf.String()

	// 4. Parse test results from JSON event stream (or fall back to output parser)
	var cases []ParsedTestCase
	if info, err := os.Stat(eventsPath); err == nil && info.Size() > 0 {
		if parsed, err := parseEventStream(eventsPath, fullOutput); err == nil && len(parsed) > 0 {
			cases = parsed
		}
	}

	if len(cases) == 0 {
		cases = parseTestOutput(fullOutput, opts.Framework)
		if len(cases) == 0 {
			passed := testErr == nil
			failMsg := ""
			if !passed {
				failMsg = fmt.Sprintf("Test execution failed: %v", testErr)
			}
			cases = append(cases, ParsedTestCase{
				Suite:    "SwiftTestSuite",
				Name:     "Execution",
				Duration: elapsed,
				Passed:   passed,
				Failure:  failMsg,
			})
		}
	}

	if err := writeJUnitResults(opts.ResultsFile, cases, elapsed); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to write JUnit test results: %v\n", err)
	}

	// 5. Handle coverage if enabled
	if coverageActive {
		if err := processCoverage(opts, testBinary, profDir, tmpDir); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to process coverage: %v\n", err)
		}
	}

	if testErr != nil {
		return fmt.Errorf("test execution failed: %w", testErr)
	}

	for _, c := range cases {
		if !c.Passed && !c.Skipped {
			return fmt.Errorf("one or more tests failed")
		}
	}

	return nil
}

func parseEventStream(path string, output ...string) ([]ParsedTestCase, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	testsByID := make(map[string]TestMetadata)
	suitesByID := make(map[string]TestMetadata)
	startTimes := make(map[string]float64)
	issuesByID := make(map[string][]string)
	paramCasesByID := make(map[string][]*paramCaseInfo)
	var results []ParsedTestCase

	// Parse issues from output if available
	issuesByNameAndArgs := make(map[string][]string)
	if len(output) > 0 && output[0] != "" {
		paramIssueRe := regexp.MustCompile(`✘\s+Test\s+(.+?)\s+recorded an issue with\s+(?:(?:\d+)\s+arguments?\s+)?(.+?)\s+at\s+(.*)`)
		for _, m := range paramIssueRe.FindAllStringSubmatch(output[0], -1) {
			tName := cleanSwiftTestName(m[1])
			args := strings.TrimSpace(m[2])
			detail := strings.TrimSpace(m[3])
			issuesByNameAndArgs[tName+"||"+args] = append(issuesByNameAndArgs[tName+"||"+args], detail)
		}
	}

	resolveSuiteName := func(meta TestMetadata) string {
		for suiteID, sMeta := range suitesByID {
			if strings.HasPrefix(meta.ID, suiteID+"/") {
				if sMeta.DisplayName != "" {
					return cleanSwiftTestName(sMeta.DisplayName)
				}
				return cleanSwiftTestName(sMeta.Name)
			}
		}
		return "SwiftTesting"
	}

	resolveTestName := func(meta TestMetadata) string {
		testName := cleanSwiftTestName(meta.DisplayName)
		if testName == "" {
			testName = cleanSwiftTestName(meta.Name)
		}
		return testName
	}

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		var rec StreamRecord
		if err := json.Unmarshal(scanner.Bytes(), &rec); err != nil {
			continue
		}

		switch rec.Kind {
		case "test":
			var meta TestMetadata
			if err := json.Unmarshal(rec.Payload, &meta); err == nil {
				if meta.Kind == "suite" {
					suitesByID[meta.ID] = meta
				} else {
					testsByID[meta.ID] = meta
				}
			}

		case "event":
			var ev EventPayload
			if err := json.Unmarshal(rec.Payload, &ev); err != nil {
				continue
			}

			switch ev.Kind {
			case "testStarted":
				startTimes[ev.TestID] = ev.Instant.Absolute

			case "testCaseStarted":
				args := ""
				for _, m := range ev.Messages {
					if m.Text != "" {
						if a := extractTestCaseArgs(m.Text); a != "" {
							args = a
							break
						}
					}
				}
				if args == "" {
					args = fmt.Sprintf("case %d", len(paramCasesByID[ev.TestID])+1)
				}
				paramCasesByID[ev.TestID] = append(paramCasesByID[ev.TestID], &paramCaseInfo{
					args:      args,
					startTime: ev.Instant.Absolute,
				})

			case "testCaseEnded":
				cases := paramCasesByID[ev.TestID]
				for _, c := range cases {
					if c.endTime == 0 {
						c.endTime = ev.Instant.Absolute
						break
					}
				}

			case "issueRecorded":
				var parts []string
				for _, m := range ev.Messages {
					if m.Text != "" {
						parts = append(parts, m.Text)
					}
				}
				for _, c := range ev.Comments {
					if c != "" {
						alreadyPresent := false
						for _, p := range parts {
							if strings.Contains(p, c) {
								alreadyPresent = true
								break
							}
						}
						if !alreadyPresent {
							parts = append(parts, c)
						}
					}
				}
				if len(parts) > 0 {
					issueText := strings.Join(parts, " - ")
					issuesByID[ev.TestID] = append(issuesByID[ev.TestID], issueText)
					cases := paramCasesByID[ev.TestID]
					if len(cases) > 0 {
						matched := false
						for _, c := range cases {
							if strings.Contains(issueText, c.args) {
								c.issues = append(c.issues, issueText)
								matched = true
								break
							}
						}
						if !matched {
							cases[len(cases)-1].issues = append(cases[len(cases)-1].issues, issueText)
						}
					}
				}

			case "testSkipped":
				meta, ok := testsByID[ev.TestID]
				if ok && meta.Kind == "function" {
					reason := "Test skipped"
					if len(ev.Comments) > 0 {
						reason = strings.Join(ev.Comments, "; ")
					}
					if meta.IsParameterized && len(paramCasesByID[ev.TestID]) > 0 {
						for _, c := range paramCasesByID[ev.TestID] {
							results = append(results, ParsedTestCase{
								Suite:    resolveSuiteName(meta),
								Name:     formatSubtestName(resolveTestName(meta), c.args),
								Duration: 0.0,
								Passed:   true,
								Skipped:  true,
								Failure:  reason,
							})
						}
					} else {
						results = append(results, ParsedTestCase{
							Suite:    resolveSuiteName(meta),
							Name:     resolveTestName(meta),
							Duration: 0.0,
							Passed:   true,
							Skipped:  true,
							Failure:  reason,
						})
					}
				}

			case "testEnded":
				meta, ok := testsByID[ev.TestID]
				if !ok || meta.Kind != "function" {
					continue
				}

				cases := paramCasesByID[ev.TestID]
				if meta.IsParameterized && len(cases) > 0 {
					testName := resolveTestName(meta)
					for _, c := range cases {
						duration := 0.0
						if c.endTime >= c.startTime && c.startTime > 0 {
							duration = c.endTime - c.startTime
						} else if start, ok := startTimes[ev.TestID]; ok && ev.Instant.Absolute >= start {
							duration = (ev.Instant.Absolute - start) / float64(len(cases))
						}

						var caseIssues []string
						if issues, ok := issuesByNameAndArgs[testName+"||"+c.args]; ok && len(issues) > 0 {
							caseIssues = append(caseIssues, issues...)
						}
						if len(caseIssues) == 0 && len(c.issues) > 0 {
							caseIssues = append(caseIssues, c.issues...)
						}

						if len(caseIssues) == 0 && len(issuesByID[ev.TestID]) > 0 {
							anyMatched := false
							for _, other := range cases {
								if len(other.issues) > 0 || len(issuesByNameAndArgs[testName+"||"+other.args]) > 0 {
									anyMatched = true
									break
								}
							}
							if !anyMatched {
								caseIssues = append(caseIssues, issuesByID[ev.TestID]...)
							}
						}

						hasFailed := len(caseIssues) > 0
						failMsg := ""
						if hasFailed {
							failMsg = strings.Join(caseIssues, "\n")
						}

						results = append(results, ParsedTestCase{
							Suite:    resolveSuiteName(meta),
							Name:     formatSubtestName(testName, c.args),
							Duration: duration,
							Passed:   !hasFailed,
							Failure:  failMsg,
						})
					}
				} else {
					duration := 0.0
					if start, ok := startTimes[ev.TestID]; ok && ev.Instant.Absolute >= start {
						duration = ev.Instant.Absolute - start
					}

					issues := issuesByID[ev.TestID]
					hasFailed := len(issues) > 0
					for _, m := range ev.Messages {
						if m.Symbol == "fail" {
							hasFailed = true
							if len(issues) == 0 {
								issues = append(issues, m.Text)
							}
						}
					}

					failMsg := ""
					if hasFailed {
						failMsg = strings.Join(issues, "\n")
						if failMsg == "" {
							failMsg = "Test failed"
						}
					}

					results = append(results, ParsedTestCase{
						Suite:    resolveSuiteName(meta),
						Name:     resolveTestName(meta),
						Duration: duration,
						Passed:   !hasFailed,
						Failure:  failMsg,
					})
				}
			}
		}
	}

	return results, scanner.Err()
}

func cleanSwiftTestName(raw string) string {
	name := strings.TrimSpace(raw)
	name = strings.Trim(name, `"'`+"`")
	name = strings.TrimSuffix(name, "()")
	name = strings.Trim(name, `"'`+"`")
	return strings.TrimSpace(name)
}

func isTestRunSummary(raw string) bool {
	trimmed := strings.TrimSpace(raw)
	return trimmed == "run" || strings.HasPrefix(trimmed, "run with ") || strings.HasPrefix(trimmed, "run ")
}

func parseTestOutput(output, framework string) []ParsedTestCase {
	var results []ParsedTestCase

	ansiRegex := regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]`)
	cleanOutput := ansiRegex.ReplaceAllString(output, "")

	// Detect suite name if present (e.g. "◇ Suite Foo started" or "✔ Suite Foo passed")
	suiteName := "SwiftTesting"
	suiteRe := regexp.MustCompile(`(?:◇|✔)\s+Suite\s+(.+?)\s+(?:started|passed)`)
	allSuites := suiteRe.FindAllStringSubmatch(cleanOutput, -1)
	uniqueSuites := make(map[string]bool)
	for _, m := range allSuites {
		if len(m) > 1 {
			s := cleanSwiftTestName(m[1])
			if s != "" {
				uniqueSuites[s] = true
			}
		}
	}
	if len(uniqueSuites) == 1 {
		for s := range uniqueSuites {
			suiteName = s
		}
	}

	// swift-testing format:
	// ✔ Test <name> passed after X.XXX seconds.
	// ✘ Test <name> failed after X.XXX seconds with N issues.
	// ✘ Test <name> recorded an issue at <file>:<line>:<col>: <msg>
	// ➜ Test <name> skipped: <reason>
	swiftTestingPassedRe := regexp.MustCompile(`✔\s+Test\s+(.+?)\s+passed after\s+([0-9.]+)\s+seconds`)
	swiftTestingFailedRe := regexp.MustCompile(`✘\s+Test\s+(.+?)\s+failed after\s+([0-9.]+)\s+seconds`)
	swiftTestingIssueRe := regexp.MustCompile(`✘\s+Test\s+(.+?)\s+recorded an issue at\s+(.*)`)
	swiftTestingSkippedRe := regexp.MustCompile(`➜\s+Test\s+(.+?)\s+skipped(?::\s*(.*))?`)

	swiftTestingParamCaseRe := regexp.MustCompile(`◇\s+Test case passing\s+(?:(?:\d+)\s+arguments?\s+)?(.+?)\s+to\s+(.+?)\s+started\.`)
	swiftTestingParamIssueRe := regexp.MustCompile(`✘\s+Test\s+(.+?)\s+recorded an issue with\s+(?:(?:\d+)\s+arguments?\s+)?(.+?)\s+at\s+(.*)`)
	swiftTestingParamPassedRe := regexp.MustCompile(`✔\s+Test\s+(.+?)\s+with\s+(\d+)\s+test cases\s+passed after\s+([0-9.]+)\s+seconds`)
	swiftTestingParamFailedRe := regexp.MustCompile(`✘\s+Test\s+(.+?)\s+with\s+(\d+)\s+test cases\s+failed after\s+([0-9.]+)\s+seconds`)

	type paramCase struct {
		args string
	}
	paramCasesByTest := make(map[string][]paramCase)
	for _, match := range swiftTestingParamCaseRe.FindAllStringSubmatch(cleanOutput, -1) {
		args := strings.TrimSpace(match[1])
		testName := cleanSwiftTestName(match[2])
		paramCasesByTest[testName] = append(paramCasesByTest[testName], paramCase{args: args})
	}

	issuesByTestAndArgs := make(map[string][]string)
	for _, match := range swiftTestingParamIssueRe.FindAllStringSubmatch(cleanOutput, -1) {
		testName := cleanSwiftTestName(match[1])
		args := strings.TrimSpace(match[2])
		detail := strings.TrimSpace(match[3])
		key := testName + "||" + args
		issuesByTestAndArgs[key] = append(issuesByTestAndArgs[key], detail)
	}

	issuesByTest := make(map[string][]string)
	for _, match := range swiftTestingIssueRe.FindAllStringSubmatch(cleanOutput, -1) {
		if isTestRunSummary(match[1]) {
			continue
		}
		testName := cleanSwiftTestName(match[1])
		issuesByTest[testName] = append(issuesByTest[testName], match[2])
	}

	// Parameterized passed
	paramHandledTests := make(map[string]bool)
	for _, match := range swiftTestingParamPassedRe.FindAllStringSubmatch(cleanOutput, -1) {
		testName := cleanSwiftTestName(match[1])
		count, _ := strconv.Atoi(match[2])
		duration, _ := strconv.ParseFloat(match[3], 64)
		paramHandledTests[testName] = true

		cases := paramCasesByTest[testName]
		if len(cases) > 0 {
			subDuration := duration / float64(len(cases))
			for _, c := range cases {
				results = append(results, ParsedTestCase{
					Suite:    suiteName,
					Name:     formatSubtestName(testName, c.args),
					Duration: subDuration,
					Passed:   true,
				})
			}
		} else if count > 0 {
			subDuration := duration / float64(count)
			for i := 1; i <= count; i++ {
				results = append(results, ParsedTestCase{
					Suite:    suiteName,
					Name:     fmt.Sprintf("%s (case %d)", testName, i),
					Duration: subDuration,
					Passed:   true,
				})
			}
		}
	}

	// Parameterized failed
	for _, match := range swiftTestingParamFailedRe.FindAllStringSubmatch(cleanOutput, -1) {
		testName := cleanSwiftTestName(match[1])
		count, _ := strconv.Atoi(match[2])
		duration, _ := strconv.ParseFloat(match[3], 64)
		paramHandledTests[testName] = true

		cases := paramCasesByTest[testName]
		if len(cases) > 0 {
			subDuration := duration / float64(len(cases))
			for _, c := range cases {
				issues := issuesByTestAndArgs[testName+"||"+c.args]
				passed := len(issues) == 0
				failMsg := ""
				if !passed {
					failMsg = strings.Join(issues, "\n")
				}
				results = append(results, ParsedTestCase{
					Suite:    suiteName,
					Name:     formatSubtestName(testName, c.args),
					Duration: subDuration,
					Passed:   passed,
					Failure:  failMsg,
				})
			}
		} else if count > 0 {
			subDuration := duration / float64(count)
			generalIssues := strings.Join(issuesByTest[testName], "\n")
			for i := 1; i <= count; i++ {
				results = append(results, ParsedTestCase{
					Suite:    suiteName,
					Name:     fmt.Sprintf("%s (case %d)", testName, i),
					Duration: subDuration,
					Passed:   false,
					Failure:  generalIssues,
				})
			}
		}
	}

	for _, match := range swiftTestingPassedRe.FindAllStringSubmatch(cleanOutput, -1) {
		if isTestRunSummary(match[1]) || (strings.Contains(match[1], " with ") && strings.Contains(match[1], " test cases")) {
			continue
		}
		name := cleanSwiftTestName(match[1])
		if paramHandledTests[name] {
			continue
		}
		duration, _ := strconv.ParseFloat(match[2], 64)
		results = append(results, ParsedTestCase{
			Suite:    suiteName,
			Name:     name,
			Duration: duration,
			Passed:   true,
		})
	}

	for _, match := range swiftTestingFailedRe.FindAllStringSubmatch(cleanOutput, -1) {
		if isTestRunSummary(match[1]) || (strings.Contains(match[1], " with ") && strings.Contains(match[1], " test cases")) {
			continue
		}
		name := cleanSwiftTestName(match[1])
		if paramHandledTests[name] {
			continue
		}
		duration, _ := strconv.ParseFloat(match[2], 64)
		issueDetails := strings.Join(issuesByTest[name], "\n")
		if issueDetails == "" {
			issueDetails = "Test failed"
		}
		results = append(results, ParsedTestCase{
			Suite:    suiteName,
			Name:     name,
			Duration: duration,
			Passed:   false,
			Failure:  issueDetails,
		})
	}

	for _, match := range swiftTestingSkippedRe.FindAllStringSubmatch(cleanOutput, -1) {
		if isTestRunSummary(match[1]) {
			continue
		}
		name := cleanSwiftTestName(match[1])
		reason := "Test skipped"
		if len(match) > 2 && strings.TrimSpace(match[2]) != "" {
			reason = strings.Trim(strings.TrimSpace(match[2]), `"`)
		}
		results = append(results, ParsedTestCase{
			Suite:    suiteName,
			Name:     name,
			Duration: 0.0,
			Passed:   true,
			Skipped:  true,
			Failure:  reason,
		})
	}

	if len(results) > 0 {
		return results
	}

	// XCTest format fallback:
	// Test Case '-[Suite.Test testMethod]' passed (0.001 seconds).
	// Test Case '-[Suite.Test testMethod]' failed (0.002 seconds).
	xctestPassedRe := regexp.MustCompile(`Test Case '-\[([^ ]+) ([^\]]+)\]' passed \(([0-9.]+) seconds\)`)
	xctestFailedRe := regexp.MustCompile(`Test Case '-\[([^ ]+) ([^\]]+)\]' failed \(([0-9.]+) seconds\)`)

	for _, match := range xctestPassedRe.FindAllStringSubmatch(cleanOutput, -1) {
		duration, _ := strconv.ParseFloat(match[3], 64)
		results = append(results, ParsedTestCase{
			Suite:    match[1],
			Name:     match[2],
			Duration: duration,
			Passed:   true,
		})
	}

	for _, match := range xctestFailedRe.FindAllStringSubmatch(cleanOutput, -1) {
		duration, _ := strconv.ParseFloat(match[3], 64)
		results = append(results, ParsedTestCase{
			Suite:    match[1],
			Name:     match[2],
			Duration: duration,
			Passed:   false,
			Failure:  "XCTest failed",
		})
	}

	return results
}

func writeJUnitResults(path string, cases []ParsedTestCase, totalTime float64) error {
	_ = os.MkdirAll(filepath.Dir(path), 0755)

	suitesMap := make(map[string][]ParsedTestCase)
	for _, c := range cases {
		suitesMap[c.Suite] = append(suitesMap[c.Suite], c)
	}

	var suites []JUnitTestSuite
	totalTests := 0
	totalFailures := 0
	totalSkipped := 0

	for suiteName, suiteCases := range suitesMap {
		s := JUnitTestSuite{
			Name: suiteName,
		}
		suiteTime := 0.0
		for _, c := range suiteCases {
			tc := JUnitTestCase{
				Name:      c.Name,
				ClassName: c.Suite,
				Time:      fmt.Sprintf("%.4f", c.Duration),
			}
			suiteTime += c.Duration
			totalTests++
			if c.Skipped {
				tc.Skipped = &JUnitSkipped{
					Message: c.Failure,
				}
				s.Skipped++
				totalSkipped++
			} else if !c.Passed {
				tc.Failure = &JUnitFailure{
					Message:  c.Failure,
					Contents: c.Failure,
				}
				s.Failures++
				totalFailures++
			}
			s.TestCases = append(s.TestCases, tc)
		}
		s.Tests = len(suiteCases)
		s.Time = fmt.Sprintf("%.4f", suiteTime)
		suites = append(suites, s)
	}

	ts := JUnitTestSuites{
		Suites:   suites,
		Tests:    totalTests,
		Failures: totalFailures,
		Errors:   0,
		Skipped:  totalSkipped,
		Time:     fmt.Sprintf("%.4f", totalTime),
	}

	data, err := xml.MarshalIndent(ts, "", "  ")
	if err != nil {
		return err
	}
	content := []byte(xml.Header + string(data) + "\n")
	return os.WriteFile(path, content, 0644)
}

func processCoverage(opts RunOptions, testBinary, profDir, tmpDir string) error {
	entries, err := os.ReadDir(profDir)
	if err != nil {
		return err
	}

	var profFiles []string
	for _, entry := range entries {
		if strings.HasSuffix(entry.Name(), ".profraw") {
			profFiles = append(profFiles, filepath.Join(profDir, entry.Name()))
		}
	}

	if len(profFiles) == 0 {
		return fmt.Errorf("no .profraw files found in %s", profDir)
	}

	mergedProf := filepath.Join(tmpDir, "test.profdata")
	mergeArgs := append([]string{"merge", "-sparse"}, profFiles...)
	mergeArgs = append(mergeArgs, "-o", mergedProf)

	if err := exec.Command(opts.LlvmProfdata, mergeArgs...).Run(); err != nil {
		return fmt.Errorf("llvm-profdata merge failed: %w", err)
	}

	exportCmd := exec.Command(opts.LlvmCov, "export", "-format=lcov", testBinary, "-instr-profile="+mergedProf)
	var lcovOut bytes.Buffer
	exportCmd.Stdout = &lcovOut
	if err := exportCmd.Run(); err != nil {
		return fmt.Errorf("llvm-cov export failed: %w", err)
	}

	cwd, _ := os.Getwd()
	coberturaXML := LcovToCoberturaXML(lcovOut.Bytes(), cwd)

	if opts.LcovFile != "" {
		if err := exportRawLcov(opts, testBinary, mergedProf, lcovOut.Bytes(), cwd); err != nil {
			fmt.Fprintf(os.Stderr, "raw lcov export error: %v\n", err)
		}
	}

	if err := os.MkdirAll(filepath.Dir(opts.CoverageFile), 0755); err != nil {
		return err
	}
	if err := os.WriteFile(opts.CoverageFile, coberturaXML, 0644); err != nil {
		return err
	}

	// Also write to $COVERAGE_FILE if set
	if covEnv := os.Getenv("COVERAGE_FILE"); covEnv != "" && covEnv != opts.CoverageFile {
		_ = os.MkdirAll(filepath.Dir(covEnv), 0755)
		_ = os.WriteFile(covEnv, coberturaXML, 0644)
	}

	return nil
}

type coverageLine struct {
	Number int
	Hits   int
}

type coverageFileRecord struct {
	Filename string
	Lines    []coverageLine
}

// coveragePath maps a source path found in an lcov report to a repository-relative
// path. It returns "" for the synthetic test runner entry point, which is not part of
// the project.
func coveragePath(path, cleanCwd string) string {
	path = strings.TrimPrefix(path, "file://")
	if strings.Contains(path, "__runner_main.swift") {
		return ""
	}

	// Normalize paths to repository-relative
	if idx := strings.Index(path, "._test/"); idx != -1 {
		path = path[idx+len("._test/"):]
		if strings.HasPrefix(path, "run_") {
			if slashIdx := strings.Index(path, "/"); slashIdx != -1 {
				path = path[slashIdx+1:]
			}
		}
	} else if idx := strings.Index(path, "._build/"); idx != -1 {
		path = path[idx+len("._build/"):]
	}

	if strings.HasPrefix(path, cleanCwd) {
		path = strings.TrimPrefix(path, cleanCwd)
		path = strings.TrimPrefix(path, "/")
	}
	return strings.TrimPrefix(path, "./")
}

// LcovToCoberturaXML transforms LCOV data into standard Cobertura XML format.
func LcovToCoberturaXML(lcovData []byte, cwd string) []byte {
	lines := strings.Split(string(lcovData), "\n")
	cleanCwd := filepath.Clean(cwd)

	var files []coverageFileRecord
	var currentFile *coverageFileRecord

	for _, rawLine := range lines {
		line := strings.TrimSpace(rawLine)
		if strings.HasPrefix(line, "SF:") {
			if currentFile != nil && len(currentFile.Lines) > 0 {
				files = append(files, *currentFile)
			}
			path := coveragePath(strings.TrimPrefix(line, "SF:"), cleanCwd)
			if path == "" {
				// Synthetic test runner file
				currentFile = nil
				continue
			}

			currentFile = &coverageFileRecord{Filename: path}
		} else if strings.HasPrefix(line, "DA:") && currentFile != nil {
			da := strings.TrimPrefix(line, "DA:")
			parts := strings.Split(da, ",")
			if len(parts) >= 2 {
				lineNum, err1 := strconv.Atoi(strings.TrimSpace(parts[0]))
				hits, err2 := strconv.Atoi(strings.TrimSpace(parts[1]))
				if err1 == nil && err2 == nil {
					currentFile.Lines = append(currentFile.Lines, coverageLine{
						Number: lineNum,
						Hits:   hits,
					})
				}
			}
		} else if line == "end_of_record" && currentFile != nil {
			if len(currentFile.Lines) > 0 {
				files = append(files, *currentFile)
			}
			currentFile = nil
		}
	}
	if currentFile != nil && len(currentFile.Lines) > 0 {
		files = append(files, *currentFile)
	}

	if len(files) == 0 {
		return []byte("<?xml version=\"1.0\" ?>\n<coverage>\n  <packages/>\n</coverage>\n")
	}

	var sb strings.Builder
	sb.WriteString("<?xml version=\"1.0\" ?>\n")
	sb.WriteString("<coverage>\n")
	sb.WriteString("  <packages>\n")
	sb.WriteString("    <package name=\"swift\">\n")
	sb.WriteString("      <classes>\n")

	for _, f := range files {
		sb.WriteString(fmt.Sprintf("        <class name=%q filename=%q>\n", filepath.Base(f.Filename), f.Filename))
		sb.WriteString("          <lines>\n")
		sort.Slice(f.Lines, func(i, j int) bool {
			return f.Lines[i].Number < f.Lines[j].Number
		})
		for _, l := range f.Lines {
			sb.WriteString(fmt.Sprintf("            <line number=\"%d\" hits=\"%d\"/>\n", l.Number, l.Hits))
		}
		sb.WriteString("          </lines>\n")
		sb.WriteString("        </class>\n")
	}

	sb.WriteString("      </classes>\n")
	sb.WriteString("    </package>\n")
	sb.WriteString("  </packages>\n")
	sb.WriteString("</coverage>\n")

	return []byte(sb.String())
}

func discoverDependencies(deps []string) (includeDirs []string, archives []string) {
	incSet := make(map[string]bool)
	archSet := make(map[string]bool)

	searchRoots := deps
	if len(searchRoots) == 0 {
		searchRoots = []string{"."}
	}

	for _, root := range searchRoots {
		info, err := os.Stat(root)
		if err == nil && info.IsDir() {
			if !incSet[root] {
				incSet[root] = true
				includeDirs = append(includeDirs, root)
			}
		}

		_ = filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return nil
			}
			if strings.HasSuffix(path, ".swiftmodule") {
				dir := filepath.Dir(path)
				if !incSet[dir] {
					incSet[dir] = true
					includeDirs = append(includeDirs, dir)
				}
			}
			if strings.HasSuffix(path, ".a") {
				if !archSet[path] {
					archSet[path] = true
					archives = append(archives, path)
				}
			}
			return nil
		})
	}
	return includeDirs, archives
}

// saveLinkInputs copies the module directories and archives found while compiling
// into dest, keeping their paths relative to the working directory so that
// discoverDependencies can find them again. dest is created even when empty.
func saveLinkInputs(dest string, incDirs, archives []string) error {
	if err := os.MkdirAll(dest, 0755); err != nil {
		return err
	}
	copyFile := func(src string) error {
		if filepath.IsAbs(src) {
			return nil
		}
		target := filepath.Join(dest, src)
		if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
			return err
		}
		data, err := os.ReadFile(src)
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, 0644)
	}
	for _, dir := range incDirs {
		if dir == "." || filepath.IsAbs(dir) {
			continue
		}
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, e := range entries {
			if !e.IsDir() {
				if err := copyFile(filepath.Join(dir, e.Name())); err != nil {
					return err
				}
			}
		}
	}
	for _, a := range archives {
		if err := copyFile(a); err != nil {
			return err
		}
	}
	return nil
}
