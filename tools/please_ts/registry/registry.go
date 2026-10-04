// Package registry is a small npm registry client: it reads packuments, resolves a
// dependency specifier to exactly one version, and downloads tarballs verified against
// the integrity data the registry publishes.
package registry

import (
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	"tools/please_ts/semver"
)

// DefaultURL is the public npm registry.
const DefaultURL = "https://registry.npmjs.org"

// maxTarball bounds a download, so a misbehaving registry cannot exhaust memory.
const maxTarball = 512 << 20

// Dist is where a version's tarball is and how to verify it.
type Dist struct {
	Tarball   string `json:"tarball"`
	Integrity string `json:"integrity"` // "sha512-<base64>", possibly several, space separated
	Shasum    string `json:"shasum"`    // hex sha1, the older field
}

// VersionInfo is the part of a version's manifest that resolution needs. Dependencies are
// deliberately not read from here: they come from the package.json inside the verified
// tarball, so that a registry cannot claim different dependencies than the package has.
type VersionInfo struct {
	Version    string          `json:"version"`
	Dist       Dist            `json:"dist"`
	Deprecated json.RawMessage `json:"deprecated"`
}

func (v VersionInfo) deprecated() bool {
	d := strings.TrimSpace(string(v.Deprecated))
	return d != "" && d != "null" && d != "false" && d != `""`
}

// Packument lists the versions of a package.
type Packument struct {
	Name     string                 `json:"name"`
	DistTags map[string]string      `json:"dist-tags"`
	Versions map[string]VersionInfo `json:"versions"`
	Time     map[string]string      `json:"time"`
}

// Client talks to an npm registry.
type Client struct {
	BaseURL string
	HTTP    *http.Client
}

// New returns a client for the registry at baseURL (the public registry if empty).
func New(baseURL string) *Client {
	if baseURL == "" {
		baseURL = DefaultURL
	}
	return &Client{BaseURL: strings.TrimSuffix(baseURL, "/"), HTTP: &http.Client{Timeout: 120 * time.Second}}
}

// Packument fetches the packument of a package. The full document has publish times, which
// an as-of resolution needs; the abbreviated one is much smaller.
func (c *Client) Packument(name string, full bool) (*Packument, error) {
	u := c.BaseURL + "/" + strings.Replace(url.PathEscape(name), "%40", "@", 1)
	req, err := http.NewRequest("GET", u, nil)
	if err != nil {
		return nil, err
	}
	if full {
		req.Header.Set("Accept", "application/json")
	} else {
		req.Header.Set("Accept", "application/vnd.npm.install-v1+json; q=1.0, application/json; q=0.8, */*")
	}
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetching %s: %w", u, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("registry returned status %d for %s", resp.StatusCode, u)
	}
	var p Packument
	if err := json.NewDecoder(resp.Body).Decode(&p); err != nil {
		return nil, fmt.Errorf("decoding the packument of %s: %w", name, err)
	}
	return &p, nil
}

// Resolve picks the version of a package that a dependency specifier means: a version
// range or a dist-tag. Nothing is guessed: if no version satisfies the specifier it is an
// error. When asOf is set, versions published after it are ignored, which makes the result
// reproducible (this needs the full packument, with publish times).
//
// Among the versions that satisfy a range the choice follows npm: the "latest" dist-tag
// if it qualifies, otherwise the highest version, preferring releases over prereleases
// and versions that are not deprecated.
func Resolve(p *Packument, spec string, asOf time.Time) (VersionInfo, error) {
	spec = strings.TrimSpace(spec)
	if err := checkSpecifier(spec); err != nil {
		return VersionInfo{}, fmt.Errorf("%s: %w", p.Name, err)
	}

	published := func(v string) (bool, error) {
		if asOf.IsZero() {
			return true, nil
		}
		raw, ok := p.Time[v]
		if !ok {
			return false, fmt.Errorf("%s@%s has no publish time in the packument; an as-of date needs the full packument", p.Name, v)
		}
		t, err := time.Parse(time.RFC3339, raw)
		if err != nil {
			return false, fmt.Errorf("%s@%s: bad publish time %q", p.Name, v, raw)
		}
		return !t.After(asOf), nil
	}

	rng, rangeErr := semver.ParseRange(spec)
	if rangeErr != nil {
		// Not a range: it may be a dist-tag such as "latest" or "next".
		version, ok := p.DistTags[spec]
		if !ok {
			return VersionInfo{}, fmt.Errorf("%s: %q is neither a version range nor a dist-tag (%v)", p.Name, spec, rangeErr)
		}
		info, ok := p.Versions[version]
		if !ok {
			return VersionInfo{}, fmt.Errorf("%s: dist-tag %q points at %s, which is not in the packument", p.Name, spec, version)
		}
		if ok, err := published(version); err != nil {
			return VersionInfo{}, err
		} else if !ok {
			return VersionInfo{}, fmt.Errorf("%s: dist-tag %q is %s, published after %s", p.Name, spec, version, asOf.Format("2006-01-02"))
		}
		return info, nil
	}

	type candidate struct {
		v    semver.Version
		info VersionInfo
	}
	var cands []candidate
	for raw, info := range p.Versions {
		v, err := semver.Parse(raw)
		if err != nil || !rng.Satisfies(v) {
			continue
		}
		ok, err := published(raw)
		if err != nil {
			return VersionInfo{}, err
		}
		if ok {
			cands = append(cands, candidate{v, info})
		}
	}
	if len(cands) == 0 {
		suffix := ""
		if !asOf.IsZero() {
			suffix = " published by " + asOf.Format("2006-01-02")
		}
		return VersionInfo{}, fmt.Errorf("no version of %s satisfies %q%s", p.Name, spec, suffix)
	}
	sort.Slice(cands, func(i, j int) bool { return semver.Compare(cands[i].v, cands[j].v) > 0 })

	if asOf.IsZero() {
		if latest, ok := p.DistTags["latest"]; ok {
			for _, c := range cands {
				if c.info.Version == latest && !c.info.deprecated() {
					return c.info, nil
				}
			}
		}
	}
	for _, wantStable := range []bool{true, false} {
		for _, c := range cands {
			if c.info.deprecated() || (wantStable && len(c.v.Pre) > 0) {
				continue
			}
			return c.info, nil
		}
	}
	return cands[0].info, nil // everything that qualifies is deprecated
}

// checkSpecifier rejects the dependency forms that are not registry versions.
func checkSpecifier(spec string) error {
	for _, prefix := range []string{"git", "file:", "link:", "workspace:", "npm:", "github:", "http:", "https:", "/", "./", "../", "~/"} {
		if strings.HasPrefix(spec, prefix) {
			return fmt.Errorf("unsupported dependency specifier %q (only registry versions are supported)", spec)
		}
	}
	if strings.Contains(spec, "://") {
		return fmt.Errorf("unsupported dependency specifier %q (only registry versions are supported)", spec)
	}
	return nil
}

// Download fetches the tarball of a version and verifies it against the registry's integrity
// data. A registry that publishes none is an error: the download would be unverifiable.
func (c *Client) Download(name string, v VersionInfo) ([]byte, error) {
	if v.Dist.Integrity == "" && v.Dist.Shasum == "" {
		return nil, fmt.Errorf("%s@%s: the registry gives no integrity or shasum to verify the tarball with", name, v.Version)
	}
	tarball := v.Dist.Tarball
	if tarball == "" {
		base := name
		if i := strings.LastIndex(base, "/"); i >= 0 {
			base = base[i+1:]
		}
		tarball = fmt.Sprintf("%s/%s/-/%s-%s.tgz", c.BaseURL, name, base, v.Version)
	}
	resp, err := c.HTTP.Get(tarball)
	if err != nil {
		return nil, fmt.Errorf("downloading %s: %w", tarball, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("downloading %s: status %d", tarball, resp.StatusCode)
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, maxTarball+1))
	if err != nil {
		return nil, fmt.Errorf("downloading %s: %w", tarball, err)
	}
	if len(data) > maxTarball {
		return nil, fmt.Errorf("%s is larger than %d bytes", tarball, maxTarball)
	}
	if err := Verify(name, v, data); err != nil {
		return nil, err
	}
	return data, nil
}

// Verify checks data against the integrity field of v (any of its sha256, sha384 or sha512
// entries; the strongest is required to match) or, when there is none, its sha1 shasum.
func Verify(name string, v VersionInfo, data []byte) error {
	if v.Dist.Integrity != "" {
		best, bestRank := "", -1
		for _, entry := range strings.Fields(v.Dist.Integrity) {
			algo, _, ok := strings.Cut(entry, "-")
			rank := map[string]int{"sha256": 1, "sha384": 2, "sha512": 3}[algo]
			if ok && rank > bestRank {
				best, bestRank = entry, rank
			}
		}
		if bestRank < 0 {
			return fmt.Errorf("%s@%s: unsupported integrity %q", name, v.Version, v.Dist.Integrity)
		}
		algo, want, _ := strings.Cut(best, "-")
		var sum []byte
		switch algo {
		case "sha256":
			s := sha256.Sum256(data)
			sum = s[:]
		case "sha384":
			s := sha512.Sum384(data)
			sum = s[:]
		default:
			s := sha512.Sum512(data)
			sum = s[:]
		}
		if base64.StdEncoding.EncodeToString(sum) != want {
			return fmt.Errorf("%s@%s: the tarball does not match the registry's %s integrity", name, v.Version, algo)
		}
		return nil
	}
	s := sha1.Sum(data)
	if hex.EncodeToString(s[:]) != strings.ToLower(v.Dist.Shasum) {
		return fmt.Errorf("%s@%s: the tarball does not match the registry's shasum", name, v.Version)
	}
	return nil
}
