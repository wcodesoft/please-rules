package registry

import (
	"crypto/sha1"
	"crypto/sha512"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func packument(t *testing.T, doc string) *Packument {
	t.Helper()
	var p Packument
	if err := json.Unmarshal([]byte(doc), &p); err != nil {
		t.Fatal(err)
	}
	return &p
}

const msPackument = `{
  "name": "ms",
  "dist-tags": {"latest": "2.1.3", "next": "3.0.0-beta.1"},
  "versions": {
    "1.0.0": {"version": "1.0.0"},
    "2.0.0": {"version": "2.0.0"},
    "2.1.2": {"version": "2.1.2"},
    "2.1.3": {"version": "2.1.3"},
    "2.2.0": {"version": "2.2.0"},
    "2.3.0": {"version": "2.3.0", "deprecated": "use 2.2.0"},
    "3.0.0-beta.1": {"version": "3.0.0-beta.1"}
  },
  "time": {
    "1.0.0": "2015-01-01T00:00:00.000Z", "2.0.0": "2017-01-01T00:00:00.000Z",
    "2.1.2": "2019-01-01T00:00:00.000Z", "2.1.3": "2020-01-01T00:00:00.000Z",
    "2.2.0": "2023-01-01T00:00:00.000Z", "2.3.0": "2024-01-01T00:00:00.000Z",
    "3.0.0-beta.1": "2024-06-01T00:00:00.000Z"
  }
}`

func TestResolve(t *testing.T) {
	p := packument(t, msPackument)
	date := func(s string) time.Time {
		d, err := time.Parse("2006-01-02", s)
		if err != nil {
			t.Fatal(err)
		}
		return d.Add(24*time.Hour - time.Second)
	}
	for _, tc := range []struct {
		name, spec string
		asOf       time.Time
		want       string
	}{
		{"the latest tag when it satisfies", "^2.0.0", time.Time{}, "2.1.3"},
		{"highest when latest does not satisfy", "^1.0.0", time.Time{}, "1.0.0"},
		{"highest satisfying, skipping a deprecated one", "~2.2.0", time.Time{}, "2.2.0"},
		{"exact", "2.1.2", time.Time{}, "2.1.2"},
		{"star prefers latest", "*", time.Time{}, "2.1.3"},
		{"empty prefers latest", "", time.Time{}, "2.1.3"},
		{"a range of alternatives", "1.0.0 || 2.0.0", time.Time{}, "2.0.0"},
		{"a dist-tag", "latest", time.Time{}, "2.1.3"},
		{"another dist-tag", "next", time.Time{}, "3.0.0-beta.1"},
		{"a range that allows a prerelease", ">=3.0.0-beta.0", time.Time{}, "3.0.0-beta.1"},
		{"as of a date ignores the latest tag", "^2.0.0", date("2019-06-01"), "2.1.2"},
		{"as of a date, the newest published by then", "^2.0.0", date("2023-06-01"), "2.2.0"},
		{"as of the day of a release includes it", "^2.0.0", date("2020-01-01"), "2.1.3"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := Resolve(p, tc.spec, tc.asOf)
			if err != nil || got.Version != tc.want {
				t.Errorf("Resolve(%q) = %q, %v; want %s", tc.spec, got.Version, err, tc.want)
			}
		})
	}
}

func TestResolveNeverGuesses(t *testing.T) {
	p := packument(t, msPackument)
	for name, tc := range map[string]struct{ spec, want string }{
		"no version satisfies":     {"^9.0.0", "no version of ms satisfies"},
		"only a prerelease exists": {"^3.0.0", "no version of ms satisfies"},
		"not a range or a tag":     {"banana", "neither a version range nor a dist-tag"},
		"git dependency":           {"git+https://github.com/x/y.git", "unsupported dependency specifier"},
		"github shorthand":         {"github:x/y", "unsupported dependency specifier"},
		"file dependency":          {"file:../x", "unsupported dependency specifier"},
		"npm alias":                {"npm:other@^1", "unsupported dependency specifier"},
		"workspace":                {"workspace:*", "unsupported dependency specifier"},
		"url":                      {"https://example.com/x.tgz", "unsupported dependency specifier"},
		"relative path":            {"./local", "unsupported dependency specifier"},
	} {
		t.Run(name, func(t *testing.T) {
			if got, err := Resolve(p, tc.spec, time.Time{}); err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Errorf("Resolve(%q) = %q, %v; want an error containing %q", tc.spec, got.Version, err, tc.want)
			}
		})
	}
}

func TestResolveAsOfErrors(t *testing.T) {
	p := packument(t, msPackument)
	asOf := time.Date(2016, 1, 1, 0, 0, 0, 0, time.UTC)
	if _, err := Resolve(p, "^2.0.0", asOf); err == nil || !strings.Contains(err.Error(), "published by 2016-01-01") {
		t.Errorf("nothing was published by then, got %v", err)
	}
	if _, err := Resolve(p, "latest", asOf); err == nil || !strings.Contains(err.Error(), "published after") {
		t.Errorf("a dist-tag newer than the date must be an error, got %v", err)
	}
	abbreviated := packument(t, `{"name":"ms","versions":{"1.0.0":{"version":"1.0.0"}}}`)
	if _, err := Resolve(abbreviated, "^1.0.0", asOf); err == nil || !strings.Contains(err.Error(), "no publish time") {
		t.Errorf("an as-of date needs publish times, got %v", err)
	}
}

func TestResolveFallsBackToADeprecatedVersionOnlyWhenNothingElseQualifies(t *testing.T) {
	p := packument(t, `{"name":"x","versions":{"1.0.0":{"version":"1.0.0","deprecated":"old"},"1.1.0":{"version":"1.1.0","deprecated":"older"}}}`)
	got, err := Resolve(p, "^1.0.0", time.Time{})
	if err != nil || got.Version != "1.1.0" {
		t.Errorf("Resolve = %q, %v; want the highest deprecated version", got.Version, err)
	}
}

func integrityOf(data []byte) string {
	s := sha512.Sum512(data)
	return "sha512-" + base64.StdEncoding.EncodeToString(s[:])
}

func TestVerify(t *testing.T) {
	data := []byte("tarball bytes")
	sha1sum := sha1.Sum(data)
	shasum := hex.EncodeToString(sha1sum[:])
	for _, tc := range []struct {
		name string
		dist Dist
		ok   bool
	}{
		{"sha512 integrity", Dist{Integrity: integrityOf(data)}, true},
		{"integrity wins over a wrong shasum", Dist{Integrity: integrityOf(data), Shasum: "00"}, true},
		{"several integrity entries, the strongest matches", Dist{Integrity: "sha256-AAAA " + integrityOf(data)}, true},
		{"the strongest does not match", Dist{Integrity: "sha256-AAAA sha512-AAAA"}, false},
		{"wrong integrity", Dist{Integrity: "sha512-AAAA", Shasum: shasum}, false},
		{"shasum alone", Dist{Shasum: shasum}, true},
		{"uppercase shasum", Dist{Shasum: strings.ToUpper(shasum)}, true},
		{"wrong shasum", Dist{Shasum: "0000"}, false},
		{"unsupported algorithm", Dist{Integrity: "md5-AAAA"}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := Verify("x", VersionInfo{Version: "1.0.0", Dist: tc.dist}, data)
			if (err == nil) != tc.ok {
				t.Errorf("Verify = %v, want ok=%v", err, tc.ok)
			}
		})
	}
}

func newRegistry(t *testing.T, handler func(w http.ResponseWriter, r *http.Request)) *Client {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(handler))
	t.Cleanup(srv.Close)
	return New(srv.URL)
}

func TestPackumentFetchesAndEscapesScopedNames(t *testing.T) {
	var gotPath, gotAccept string
	c := newRegistry(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath, gotAccept = r.URL.EscapedPath(), r.Header.Get("Accept")
		fmt.Fprint(w, msPackument)
	})
	p, err := c.Packument("@scope/ms", true)
	if err != nil || p.Name != "ms" || len(p.Versions) != 7 {
		t.Fatalf("Packument = %+v, %v", p, err)
	}
	if gotPath != "/@scope%2Fms" && gotPath != "/@scope/ms" {
		t.Errorf("path = %q", gotPath)
	}
	if gotAccept != "application/json" {
		t.Errorf("a full packument must ask for plain JSON, Accept = %q", gotAccept)
	}
	if _, err := c.Packument("ms", false); err != nil || !strings.Contains(gotAccept, "install-v1") {
		t.Errorf("an abbreviated packument must ask for the install format, Accept = %q, %v", gotAccept, err)
	}
}

func TestPackumentErrors(t *testing.T) {
	c := newRegistry(t, func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/bad") {
			fmt.Fprint(w, "{")
			return
		}
		http.NotFound(w, r)
	})
	if _, err := c.Packument("missing", false); err == nil || !strings.Contains(err.Error(), "404") {
		t.Errorf("a 404 must be reported, got %v", err)
	}
	if _, err := c.Packument("bad", false); err == nil {
		t.Error("invalid JSON must be an error")
	}
}

func TestDownloadVerifiesTheTarball(t *testing.T) {
	data := []byte("the real tarball")
	var served = data
	c := newRegistry(t, func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write(served) })

	good := VersionInfo{Version: "1.0.0", Dist: Dist{Tarball: c.BaseURL + "/x/-/x-1.0.0.tgz", Integrity: integrityOf(data)}}
	if got, err := c.Download("x", good); err != nil || string(got) != string(data) {
		t.Fatalf("Download = %q, %v", got, err)
	}

	served = []byte("a different tarball")
	if _, err := c.Download("x", good); err == nil || !strings.Contains(err.Error(), "does not match") {
		t.Errorf("tampered content must fail verification, got %v", err)
	}

	noIntegrity := VersionInfo{Version: "1.0.0", Dist: Dist{Tarball: c.BaseURL + "/x/-/x-1.0.0.tgz"}}
	if _, err := c.Download("x", noIntegrity); err == nil || !strings.Contains(err.Error(), "no integrity") {
		t.Errorf("a version without integrity data must be refused, got %v", err)
	}
}

func TestDownloadBuildsTheTarballURLWhenTheRegistryOmitsIt(t *testing.T) {
	data := []byte("x")
	var path string
	c := newRegistry(t, func(w http.ResponseWriter, r *http.Request) { path = r.URL.Path; _, _ = w.Write(data) })
	v := VersionInfo{Version: "2.0.0", Dist: Dist{Integrity: integrityOf(data)}}
	if _, err := c.Download("@scope/pkg", v); err != nil || path != "/@scope/pkg/-/pkg-2.0.0.tgz" {
		t.Errorf("path = %q, err = %v", path, err)
	}
}

func TestDownloadReportsHTTPFailures(t *testing.T) {
	c := newRegistry(t, func(w http.ResponseWriter, r *http.Request) { http.Error(w, "boom", http.StatusInternalServerError) })
	v := VersionInfo{Version: "1.0.0", Dist: Dist{Tarball: c.BaseURL + "/x.tgz", Shasum: "00"}}
	if _, err := c.Download("x", v); err == nil || !strings.Contains(err.Error(), "500") {
		t.Errorf("got %v", err)
	}
}
