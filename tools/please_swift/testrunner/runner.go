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

// stripLeadingArgCount removes a leading "<N> argument " or "<N> arguments " prefix.
func stripLeadingArgCount(s string) string {
	s = strings.TrimSpace(s)
	spaceIdx := strings.IndexByte(s, ' ')
	if spaceIdx <= 0 {
		return s
	}
	for i := 0; i < spaceIdx; i++ {
		if s[i] < '0' || s[i] > '9' {
			return s
		}
	}
	rest := strings.TrimSpace(s[spaceIdx+1:])
	if after, ok := strings.CutPrefix(rest, "arguments"); ok && len(after) > 0 && (after[0] == ' ' || after[0] == '\t') {
		return strings.TrimSpace(after)
	}
	if after, ok := strings.CutPrefix(rest, "argument"); ok && len(after) > 0 && (after[0] == ' ' || after[0] == '\t') {
		return strings.TrimSpace(after)
	}
	return s
}

func extractTestCaseArgs(msg string) string {
	args, _, _ := parseParamCaseStartedLine(msg)
	return strings.TrimSpace(args)
}

// parseParamCaseStartedLine parses "Test case passing [N arguments] <args> to <testName> started."
// (with or without a leading "◇ " symbol).
func parseParamCaseStartedLine(line string) (args, testName string, ok bool) {
	s := strings.TrimSpace(line)
	if after, hasSymbol := strings.CutPrefix(s, "◇"); hasSymbol {
		s = strings.TrimSpace(after)
	}
	idx := strings.Index(s, "passing ")
	if idx == -1 {
		return "", "", false
	}
	rest := strings.TrimSpace(s[idx+len("passing "):])
	rest = strings.TrimSuffix(rest, " started.")
	toIdx := strings.LastIndex(rest, " to ")
	if toIdx == -1 {
		return "", "", false
	}
	args = stripLeadingArgCount(rest[:toIdx])
	testName = cleanSwiftTestName(rest[toIdx+len(" to "):])
	return args, testName, args != "" && testName != ""
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
		for _, line := range strings.Split(output[0], "\n") {
			if tName, args, detail, ok := parseParamIssueLine(line); ok {
				issuesByNameAndArgs[tName+"||"+args] = append(issuesByNameAndArgs[tName+"||"+args], detail)
			}
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

var ansiRegex = regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]`)

func parseSuiteLine(line string) (string, bool) {
	s := strings.TrimSpace(line)
	if after, ok := strings.CutPrefix(s, "◇"); ok {
		s = strings.TrimSpace(after)
	} else if after, ok := strings.CutPrefix(s, "✔"); ok {
		s = strings.TrimSpace(after)
	} else {
		return "", false
	}
	rest, ok := strings.CutPrefix(s, "Suite ")
	if !ok {
		return "", false
	}
	if idx := strings.LastIndex(rest, " started"); idx != -1 {
		return cleanSwiftTestName(rest[:idx]), true
	}
	if idx := strings.LastIndex(rest, " passed"); idx != -1 {
		return cleanSwiftTestName(rest[:idx]), true
	}
	return "", false
}

// parseParamIssueLine parses "✘ Test <name> recorded an issue with [N arguments] <args> at <detail>".
func parseParamIssueLine(line string) (testName, args, detail string, ok bool) {
	s := strings.TrimSpace(line)
	after, hasPrefix := strings.CutPrefix(s, "✘")
	if !hasPrefix {
		return "", "", "", false
	}
	rest, hasTest := strings.CutPrefix(strings.TrimSpace(after), "Test ")
	if !hasTest {
		return "", "", "", false
	}
	rawName, afterIssue, found := strings.Cut(rest, " recorded an issue with ")
	if !found {
		return "", "", "", false
	}
	atIdx := strings.LastIndex(afterIssue, " at ")
	if atIdx == -1 {
		return "", "", "", false
	}
	testName = cleanSwiftTestName(rawName)
	args = stripLeadingArgCount(afterIssue[:atIdx])
	detail = strings.TrimSpace(afterIssue[atIdx+len(" at "):])
	return testName, args, detail, testName != "" && args != ""
}

// parseTestIssueLine parses "✘ Test <name> recorded an issue at <detail>".
func parseTestIssueLine(line string) (rawName, detail string, ok bool) {
	s := strings.TrimSpace(line)
	after, hasPrefix := strings.CutPrefix(s, "✘")
	if !hasPrefix {
		return "", "", false
	}
	rest, hasTest := strings.CutPrefix(strings.TrimSpace(after), "Test ")
	if !hasTest {
		return "", "", false
	}
	namePart, detailPart, found := strings.Cut(rest, " recorded an issue at ")
	if !found {
		return "", "", false
	}
	return strings.TrimSpace(namePart), strings.TrimSpace(detailPart), true
}

// parseTrailingSeconds extracts a leading float before " seconds" in s (e.g. "0.003 seconds.").
func parseTrailingSeconds(s string) (float64, bool) {
	s = strings.TrimSpace(s)
	secPart, _, ok := strings.Cut(s, " seconds")
	if !ok {
		return 0, false
	}
	dur, err := strconv.ParseFloat(strings.TrimSpace(secPart), 64)
	if err != nil {
		return 0, false
	}
	return dur, true
}

// parseParamOutcomeLine parses "✔/✘ Test <name> with <count> test cases passed/failed after <dur> seconds...".
func parseParamOutcomeLine(line, symbol, verb string) (testName string, count int, duration float64, ok bool) {
	s := strings.TrimSpace(line)
	after, hasSym := strings.CutPrefix(s, symbol)
	if !hasSym {
		return "", 0, 0, false
	}
	rest, hasTest := strings.CutPrefix(strings.TrimSpace(after), "Test ")
	if !hasTest {
		return "", 0, 0, false
	}
	marker := " test cases " + verb + " after "
	beforeMarker, afterMarker, found := strings.Cut(rest, marker)
	if !found {
		return "", 0, 0, false
	}
	withIdx := strings.LastIndex(beforeMarker, " with ")
	if withIdx == -1 {
		return "", 0, 0, false
	}
	n, err := strconv.Atoi(strings.TrimSpace(beforeMarker[withIdx+len(" with "):]))
	if err != nil {
		return "", 0, 0, false
	}
	dur, durOK := parseTrailingSeconds(afterMarker)
	if !durOK {
		return "", 0, 0, false
	}
	return cleanSwiftTestName(beforeMarker[:withIdx]), n, dur, true
}

// parseSimpleOutcomeLine parses "✔/✘ Test <name> passed/failed after <dur> seconds...".
func parseSimpleOutcomeLine(line, symbol, verb string) (rawName string, duration float64, ok bool) {
	s := strings.TrimSpace(line)
	after, hasSym := strings.CutPrefix(s, symbol)
	if !hasSym {
		return "", 0, false
	}
	rest, hasTest := strings.CutPrefix(strings.TrimSpace(after), "Test ")
	if !hasTest {
		return "", 0, false
	}
	marker := " " + verb + " after "
	idx := strings.LastIndex(rest, marker)
	if idx == -1 {
		return "", 0, false
	}
	dur, durOK := parseTrailingSeconds(rest[idx+len(marker):])
	if !durOK {
		return "", 0, false
	}
	return strings.TrimSpace(rest[:idx]), dur, true
}

// parseSkippedLine parses "➜ Test <name> skipped[: <reason>]".
func parseSkippedLine(line string) (rawName, reason string, ok bool) {
	s := strings.TrimSpace(line)
	after, hasSym := strings.CutPrefix(s, "➜")
	if !hasSym {
		return "", "", false
	}
	rest, hasTest := strings.CutPrefix(strings.TrimSpace(after), "Test ")
	if !hasTest {
		return "", "", false
	}
	idx := strings.LastIndex(rest, " skipped")
	if idx == -1 {
		return "", "", false
	}
	rawName = strings.TrimSpace(rest[:idx])
	tail := strings.TrimSpace(rest[idx+len(" skipped"):])
	if afterColon, hasColon := strings.CutPrefix(tail, ":"); hasColon {
		reason = strings.Trim(strings.TrimSpace(afterColon), `"`)
	}
	if reason == "" {
		reason = "Test skipped"
	}
	return rawName, reason, true
}

// parseXCTestCaseLine parses "Test Case '-[Suite testMethod]' passed/failed (0.001 seconds)."
func parseXCTestCaseLine(line, verb string) (suite, name string, duration float64, ok bool) {
	s := strings.TrimSpace(line)
	rest, hasPrefix := strings.CutPrefix(s, "Test Case '-[")
	if !hasPrefix {
		return "", "", 0, false
	}
	target, tail, hasBracket := strings.Cut(rest, fmt.Sprintf("]' %s (", verb))
	if !hasBracket {
		return "", "", 0, false
	}
	suitePart, methodPart, hasSpace := strings.Cut(strings.TrimSpace(target), " ")
	if !hasSpace || suitePart == "" || methodPart == "" {
		return "", "", 0, false
	}
	dur, durOK := parseTrailingSeconds(tail)
	if !durOK {
		return "", "", 0, false
	}
	return suitePart, methodPart, dur, true
}

func parseTestOutput(output, framework string) []ParsedTestCase {
	var results []ParsedTestCase

	cleanOutput := ansiRegex.ReplaceAllString(output, "")
	lines := strings.Split(cleanOutput, "\n")

	// Detect suite name if present (e.g. "◇ Suite Foo started" or "✔ Suite Foo passed")
	suiteName := "SwiftTesting"
	uniqueSuites := make(map[string]bool)
	for _, line := range lines {
		if s, ok := parseSuiteLine(line); ok && s != "" {
			uniqueSuites[s] = true
		}
	}
	if len(uniqueSuites) == 1 {
		for s := range uniqueSuites {
			suiteName = s
		}
	}

	type paramCase struct {
		args string
	}
	paramCasesByTest := make(map[string][]paramCase)
	issuesByTestAndArgs := make(map[string][]string)
	issuesByTest := make(map[string][]string)

	for _, line := range lines {
		if args, testName, ok := parseParamCaseStartedLine(line); ok && strings.HasPrefix(strings.TrimSpace(line), "◇") {
			paramCasesByTest[testName] = append(paramCasesByTest[testName], paramCase{args: args})
		}
		if testName, args, detail, ok := parseParamIssueLine(line); ok {
			key := testName + "||" + args
			issuesByTestAndArgs[key] = append(issuesByTestAndArgs[key], detail)
		}
		if rawName, detail, ok := parseTestIssueLine(line); ok && !isTestRunSummary(rawName) {
			testName := cleanSwiftTestName(rawName)
			issuesByTest[testName] = append(issuesByTest[testName], detail)
		}
	}

	// Parameterized passed
	paramHandledTests := make(map[string]bool)
	for _, line := range lines {
		testName, count, duration, ok := parseParamOutcomeLine(line, "✔", "passed")
		if !ok {
			continue
		}
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
	for _, line := range lines {
		testName, count, duration, ok := parseParamOutcomeLine(line, "✘", "failed")
		if !ok {
			continue
		}
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

	for _, line := range lines {
		rawName, duration, ok := parseSimpleOutcomeLine(line, "✔", "passed")
		if !ok || isTestRunSummary(rawName) || (strings.Contains(rawName, " with ") && strings.Contains(rawName, " test cases")) {
			continue
		}
		name := cleanSwiftTestName(rawName)
		if paramHandledTests[name] {
			continue
		}
		results = append(results, ParsedTestCase{
			Suite:    suiteName,
			Name:     name,
			Duration: duration,
			Passed:   true,
		})
	}

	for _, line := range lines {
		rawName, duration, ok := parseSimpleOutcomeLine(line, "✘", "failed")
		if !ok || isTestRunSummary(rawName) || (strings.Contains(rawName, " with ") && strings.Contains(rawName, " test cases")) {
			continue
		}
		name := cleanSwiftTestName(rawName)
		if paramHandledTests[name] {
			continue
		}
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

	for _, line := range lines {
		rawName, reason, ok := parseSkippedLine(line)
		if !ok || isTestRunSummary(rawName) {
			continue
		}
		name := cleanSwiftTestName(rawName)
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
	for _, line := range lines {
		if suite, name, duration, ok := parseXCTestCaseLine(line, "passed"); ok {
			results = append(results, ParsedTestCase{
				Suite:    suite,
				Name:     name,
				Duration: duration,
				Passed:   true,
			})
		}
	}
	for _, line := range lines {
		if suite, name, duration, ok := parseXCTestCaseLine(line, "failed"); ok {
			results = append(results, ParsedTestCase{
				Suite:    suite,
				Name:     name,
				Duration: duration,
				Passed:   false,
				Failure:  "XCTest failed",
			})
		}
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
