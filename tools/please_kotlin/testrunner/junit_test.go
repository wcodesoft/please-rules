package testrunner

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// Reports written by the JUnit Platform console launcher (--reports-dir) for a Jupiter test
// class with three tests, one of which raised an exception. The <properties> block is left
// out; the rest is as the launcher wrote it. The launcher writes one file per engine, and
// engines that found no test still get a report.
const jupiterReport = `<?xml version="1.0" encoding="UTF-8"?>
<testsuite name="JUnit Jupiter" tests="3" skipped="0" failures="0" errors="1" time="0.048" hostname="host" timestamp="2026-10-04T13:39:38">
<properties>
</properties>
<testcase name="testAdd()" classname="test.kotlin.lib.LibTest" time="0.002">
<system-out><![CDATA[
unique-id: [engine:junit-jupiter]/[class:test.kotlin.lib.LibTest]/[method:testAdd()]
display-name: testAdd()
]]></system-out>
</testcase>
<testcase name="testMavenAnnotations()" classname="test.kotlin.lib.LibTest" time="0.016">
<error message="kotlin/jvm/internal/Intrinsics" type="java.lang.NoClassDefFoundError"><![CDATA[java.lang.NoClassDefFoundError: kotlin/jvm/internal/Intrinsics
	at test.kotlin.lib.LibKt.annotateMessage(lib.kt)
	at test.kotlin.lib.LibTest.testMavenAnnotations(lib_test.kt:20)
]]></error>
<system-out><![CDATA[
unique-id: [engine:junit-jupiter]/[class:test.kotlin.lib.LibTest]/[method:testMavenAnnotations()]
]]></system-out>
</testcase>
<testcase name="testMultiply()" classname="test.kotlin.lib.LibTest" time="0">
</testcase>
<system-out><![CDATA[
unique-id: [engine:junit-jupiter]
]]></system-out>
</testsuite>
`

const vintageReport = `<?xml version="1.0" encoding="UTF-8"?>
<testsuite name="JUnit Vintage" tests="0" skipped="0" failures="0" errors="0" time="0" hostname="host" timestamp="2026-10-04T13:39:38">
<properties>
</properties>
</testsuite>
`

// Not written by the launcher here: a JUnit 4 class run by the vintage engine, with an
// assertion failure and a skipped test, in the format the launcher uses for those.
const vintageWithTests = `<?xml version="1.0" encoding="UTF-8"?>
<testsuite name="JUnit Vintage" tests="2" skipped="1" failures="1" errors="0" time="0.01" hostname="host" timestamp="2026-10-04T13:39:38">
<testcase name="checksTotal" classname="legacy.TotalTest" time="0.004">
<failure message="expected: &lt;5&gt; but was: &lt;6&gt;" type="java.lang.AssertionError"><![CDATA[java.lang.AssertionError: expected: <5> but was: <6>
	at legacy.TotalTest.checksTotal(TotalTest.java:12)
]]></failure>
</testcase>
<testcase name="notYet" classname="legacy.TotalTest" time="0">
<skipped message="not implemented"/>
</testcase>
</testsuite>
`

func writeReports(t *testing.T, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func TestReadJUnitReportsReadsTheLauncherOutput(t *testing.T) {
	dir := writeReports(t, map[string]string{
		"TEST-junit-jupiter.xml":        jupiterReport,
		"TEST-junit-vintage.xml":        vintageReport,
		"TEST-junit-platform-suite.xml": vintageReport,
		"jacoco_report.xml":             "<report/>", // not a test report: must be ignored
	})
	suites, err := ReadJUnitReports(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(suites) != 1 {
		t.Fatalf("suites = %d, want only the Jupiter one (engines that ran nothing are dropped)", len(suites))
	}
	s := suites[0]
	if s.Name != "JUnit Jupiter" || s.Tests != 3 || s.Errors != 1 || s.Failures != 0 || s.Skipped != 0 {
		t.Errorf("suite = %+v", s)
	}
	names := []string{}
	for _, tc := range s.TestCases {
		names = append(names, tc.Name)
	}
	if strings.Join(names, ",") != "testAdd(),testMavenAnnotations(),testMultiply()" {
		t.Errorf("test cases = %v", names)
	}
	errCase := s.TestCases[1]
	if errCase.Error == nil || errCase.Error.Type != "java.lang.NoClassDefFoundError" || !strings.Contains(errCase.Error.Contents, "LibTest.testMavenAnnotations") {
		t.Errorf("the error of testMavenAnnotations was lost: %+v", errCase.Error)
	}
	if !Failed(suites) {
		t.Error("an errored test case is a failure")
	}
}

func TestReadJUnitReportsKeepsEveryEngineThatRanTests(t *testing.T) {
	// The runner used to copy only the first .xml, losing the results of the other engines.
	dir := writeReports(t, map[string]string{
		"TEST-junit-jupiter.xml": strings.Replace(jupiterReport, `errors="1"`, `errors="0"`, 1),
		"TEST-junit-vintage.xml": vintageWithTests,
	})
	suites, err := ReadJUnitReports(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(suites) != 2 || suites[0].Name != "JUnit Jupiter" || suites[1].Name != "JUnit Vintage" {
		t.Fatalf("suites = %+v, want Jupiter then Vintage", suites)
	}
	v := suites[1]
	if v.Tests != 2 || v.Failures != 1 || v.Skipped != 1 || v.Errors != 0 {
		t.Errorf("vintage suite = %+v", v)
	}
	if f := v.TestCases[0].Failure; f == nil || f.Type != "java.lang.AssertionError" || !strings.Contains(f.Message, "expected: <5> but was: <6>") {
		t.Errorf("failure = %+v", f)
	}
	if v.TestCases[1].Skipped == nil || v.TestCases[1].Skipped.Message != "not implemented" {
		t.Errorf("skipped = %+v", v.TestCases[1].Skipped)
	}
}

func TestReadJUnitReportsRecountsFromTheTestCases(t *testing.T) {
	lying := strings.Replace(jupiterReport, `tests="3" skipped="0" failures="0" errors="1"`, `tests="99" skipped="9" failures="9" errors="0"`, 1)
	suites, err := ReadJUnitReports(writeReports(t, map[string]string{"TEST-a.xml": lying}))
	if err != nil {
		t.Fatal(err)
	}
	if s := suites[0]; s.Tests != 3 || s.Errors != 1 || s.Failures != 0 || s.Skipped != 0 {
		t.Errorf("counts must come from the test cases: %+v", s)
	}
}

func TestReadJUnitReportsAcceptsATestsuitesRoot(t *testing.T) {
	wrapped := `<?xml version="1.0"?><testsuites>` + strings.SplitN(vintageWithTests, "?>", 2)[1] + `</testsuites>`
	suites, err := ReadJUnitReports(writeReports(t, map[string]string{"TEST-wrapped.xml": wrapped}))
	if err != nil || len(suites) != 1 || suites[0].Tests != 2 {
		t.Errorf("suites = %+v, err = %v", suites, err)
	}
}

func TestReadJUnitReportsWithoutReports(t *testing.T) {
	suites, err := ReadJUnitReports(t.TempDir())
	if err != nil || len(suites) != 0 {
		t.Errorf("an empty directory gives no suites and no error, got %v, %v", suites, err)
	}
}

func TestReadJUnitReportsRejectsMalformedReports(t *testing.T) {
	for name, content := range map[string]string{
		"truncated":  `<?xml version="1.0"?><testsuite name="x"><testcase name="a"`,
		"empty file": "",
		"other root": `<?xml version="1.0"?><report/>`,
		"not xml":    "tests passed",
	} {
		_, err := ReadJUnitReports(writeReports(t, map[string]string{"TEST-bad.xml": content}))
		if err == nil || !strings.Contains(err.Error(), "TEST-bad.xml") {
			t.Errorf("%s: got %v, want an error naming the file", name, err)
		}
	}
}

func TestProcessSuite(t *testing.T) {
	ok := ProcessSuite("pkg.MyTest", "all good", 250*time.Millisecond, true)
	if ok.Tests != 1 || ok.Failures != 0 || ok.Errors != 0 || ok.Time != "0.250" || ok.Name != "pkg.MyTest" {
		t.Errorf("successful process = %+v", ok)
	}
	bad := ProcessSuite("pkg.MyTest", "boom\nstack", time.Second, false)
	if bad.Tests != 1 || bad.Errors != 1 || bad.TestCases[0].Error == nil || bad.TestCases[0].Error.Contents != "boom\nstack" {
		t.Errorf("failed process = %+v", bad)
	}
	if !Failed([]JUnitTestSuite{bad}) || Failed([]JUnitTestSuite{ok}) {
		t.Error("Failed disagrees with the suites")
	}
}

func TestWriteJUnitResultsWritesEverySuiteAndCase(t *testing.T) {
	dir := writeReports(t, map[string]string{
		"TEST-junit-jupiter.xml": jupiterReport,
		"TEST-junit-vintage.xml": vintageWithTests,
	})
	suites, err := ReadJUnitReports(dir)
	if err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(t.TempDir(), "sub", "test.results")
	if err := WriteJUnitResults(out, suites...); err != nil {
		t.Fatalf("WriteJUnitResults failed: %v", err)
	}
	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"<testsuites>", `name="JUnit Jupiter"`, `name="JUnit Vintage"`, "<error ", "NoClassDefFoundError", "<failure ", "<skipped "} {
		if !strings.Contains(string(data), want) {
			t.Errorf("results lack %q:\n%s", want, data)
		}
	}

	// What was written reads back as the same suites.
	back, err := parseJUnitReport(data)
	if err != nil || len(back) != 2 || back[0].Tests != 3 || back[1].Tests != 2 {
		t.Errorf("round trip = %+v, %v", back, err)
	}
}
