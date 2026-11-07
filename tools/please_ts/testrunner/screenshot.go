package testrunner

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// screenshotBinding is the name of the function the page calls to ask the host for a capture.
const screenshotBinding = "__cdpScreenshot"

// chromiumRenderingFlags make the headless browser render the same pixels on every machine: no
// subpixel text positioning or hinting, no LCD antialiasing, no scrollbars taking layout space,
// one fixed colour profile and device scale. They are only passed when screenshots are in use, so
// the layout of other browser tests does not change.
var chromiumRenderingFlags = []string{
	"--font-render-hinting=none",
	"--disable-font-subpixel-positioning",
	"--disable-lcd-text",
	"--hide-scrollbars",
	"--force-color-profile=srgb",
	"--force-device-scale-factor=1",
}

// screenshotHelperJS adds expectScreenshot to the page. The returned promise settles once the host
// has captured (or failed to capture) the region, so a test must await it before touching the DOM
// again. It waits for web fonts and two animation frames first, so the capture sees a settled page.
const screenshotHelperJS = `(() => {
	let seq = 0;
	const waiting = {};
	window.__screenshotDone = function(id, error) {
		const w = waiting[id];
		delete waiting[id];
		if (!w) return;
		if (error) w.reject(new Error(error)); else w.resolve();
	};
	window.expectScreenshot = async function(name, options) {
		options = options || {};
		if (typeof name !== 'string' || name === '' || name === '.' || name === '..' || !/^[A-Za-z0-9_.-]+$/.test(name)) {
			throw new Error('invalid screenshot name ' + JSON.stringify(name) + ': use letters, digits, "_", "." and "-"');
		}
		if (name.endsWith('.diff')) {
			throw new Error('screenshot name ' + name + ' is reserved for diff images');
		}
		if (document.fonts && document.fonts.ready) {
			await document.fonts.ready;
		}
		await new Promise(r => requestAnimationFrame(() => requestAnimationFrame(r)));

		let x = 0, y = 0, width, height;
		if (options.selector) {
			const el = document.querySelector(options.selector);
			if (!el) throw new Error('screenshot ' + name + ': no element matches ' + options.selector);
			const r = el.getBoundingClientRect();
			x = Math.floor(r.left + window.scrollX);
			y = Math.floor(r.top + window.scrollY);
			width = Math.ceil(r.right + window.scrollX) - x;
			height = Math.ceil(r.bottom + window.scrollY) - y;
		} else if (options.fullPage) {
			width = Math.ceil(Math.max(document.documentElement.scrollWidth, document.body ? document.body.scrollWidth : 0));
			height = Math.ceil(Math.max(document.documentElement.scrollHeight, document.body ? document.body.scrollHeight : 0));
		} else {
			width = window.innerWidth;
			height = window.innerHeight;
		}
		if (width <= 0 || height <= 0) {
			throw new Error('screenshot ' + name + ': the region is empty (' + width + 'x' + height + ')');
		}

		const id = ++seq;
		const done = new Promise((resolve, reject) => { waiting[id] = { resolve, reject }; });
		window.` + screenshotBinding + `(JSON.stringify({ id: id, name: name, x: x, y: y, width: width, height: height }));
		return done;
	};
})()`

// screenshotRequest is what the page sends through the binding.
type screenshotRequest struct {
	ID     int    `json:"id"`
	Name   string `json:"name"`
	X      int    `json:"x"`
	Y      int    `json:"y"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
}

func parseScreenshotRequest(payload string) (screenshotRequest, error) {
	var req screenshotRequest
	if err := json.Unmarshal([]byte(payload), &req); err != nil {
		return req, fmt.Errorf("malformed screenshot request: %w", err)
	}
	if !validSnapshotName(req.Name) || strings.HasSuffix(req.Name, ".diff") {
		return req, fmt.Errorf("invalid screenshot name %q", req.Name)
	}
	if req.Width <= 0 || req.Height <= 0 {
		return req, fmt.Errorf("screenshot %s: empty region %dx%d", req.Name, req.Width, req.Height)
	}
	return req, nil
}

// pixelTolerance says how far two screenshots may differ and still match.
type pixelTolerance struct {
	// Channel is the largest per-channel (0-255) difference a pixel may have and count as equal.
	Channel int
	// Ratio is the largest share (0-1) of pixels allowed to differ.
	Ratio float64
}

// compareImages counts the pixels of a and b that differ by more than tol.Channel in any channel.
// It also returns a diff image: unchanged pixels faded, differing pixels in solid red. When the
// sizes differ it returns ok=false and no diff.
func compareImages(golden, actual image.Image, tol pixelTolerance) (differing, total int, diff *image.NRGBA, ok bool) {
	gb, ab := golden.Bounds(), actual.Bounds()
	if gb.Dx() != ab.Dx() || gb.Dy() != ab.Dy() {
		return 0, 0, nil, false
	}
	diff = image.NewNRGBA(image.Rect(0, 0, gb.Dx(), gb.Dy()))
	for y := 0; y < gb.Dy(); y++ {
		for x := 0; x < gb.Dx(); x++ {
			g := color.NRGBAModel.Convert(golden.At(gb.Min.X+x, gb.Min.Y+y)).(color.NRGBA)
			a := color.NRGBAModel.Convert(actual.At(ab.Min.X+x, ab.Min.Y+y)).(color.NRGBA)
			if channelDelta(g, a) > tol.Channel {
				differing++
				diff.SetNRGBA(x, y, color.NRGBA{R: 255, A: 255})
			} else {
				diff.SetNRGBA(x, y, color.NRGBA{R: a.R, G: a.G, B: a.B, A: 60})
			}
		}
	}
	return differing, gb.Dx() * gb.Dy(), diff, true
}

func channelDelta(a, b color.NRGBA) int {
	d := absDiff(a.R, b.R)
	for _, p := range [][2]uint8{{a.G, b.G}, {a.B, b.B}, {a.A, b.A}} {
		if v := absDiff(p[0], p[1]); v > d {
			d = v
		}
	}
	return d
}

func absDiff(a, b uint8) int {
	if a > b {
		return int(a - b)
	}
	return int(b - a)
}

// checkScreenshots compares the captured PNGs with the golden PNGs and returns one test result per
// screenshot. Every capture is written to outDir as <name>.png, and a failing comparison also
// writes <name>.diff.png, so accepting a change means copying the capture over the golden. A
// golden that nothing captured is reported as stale.
func checkScreenshots(goldenFiles []string, outDir string, captured map[string][]byte, tol pixelTolerance) ([]browserTestResult, error) {
	goldens := map[string]string{}
	for _, f := range goldenFiles {
		name := strings.TrimSuffix(filepath.Base(f), ".png")
		if !validSnapshotName(name) || strings.HasSuffix(name, ".diff") {
			return nil, fmt.Errorf("invalid golden screenshot name %q", f)
		}
		if prev, dup := goldens[name]; dup {
			return nil, fmt.Errorf("goldens %q and %q share the screenshot name %q", prev, f, name)
		}
		goldens[name] = f
	}

	var results []browserTestResult
	for _, name := range sortedKeys(captured) {
		if !validSnapshotName(name) {
			return nil, fmt.Errorf("invalid screenshot name %q", name)
		}
		if outDir != "" {
			if err := os.MkdirAll(outDir, 0755); err != nil {
				return nil, err
			}
			if err := os.WriteFile(filepath.Join(outDir, name+".png"), captured[name], 0644); err != nil {
				return nil, err
			}
		}

		res := browserTestResult{Name: "screenshot: " + name}
		goldenPath, ok := goldens[name]
		if !ok {
			res.Error = fmt.Sprintf("no golden for screenshot %q; add %s.png to the target's screenshots (the capture is in the test outputs)", name, name)
			results = append(results, res)
			continue
		}
		delete(goldens, name)

		res.Error = compareScreenshot(name, goldenPath, captured[name], outDir, tol)
		results = append(results, res)
	}

	for _, name := range sortedKeys(goldens) {
		results = append(results, browserTestResult{
			Name:  "screenshot: " + name,
			Error: fmt.Sprintf("golden %s was not captured by any test; delete it or restore the test", goldens[name]),
		})
	}
	return results, nil
}

// compareScreenshot returns the failure message for one screenshot, or "" when it matches.
func compareScreenshot(name, goldenPath string, actualPNG []byte, outDir string, tol pixelTolerance) string {
	raw, err := os.ReadFile(goldenPath)
	if err != nil {
		return fmt.Sprintf("cannot read golden %s: %v", goldenPath, err)
	}
	golden, err := png.Decode(bytes.NewReader(raw))
	if err != nil {
		return fmt.Sprintf("golden %s is not a valid PNG: %v", goldenPath, err)
	}
	actual, err := png.Decode(bytes.NewReader(actualPNG))
	if err != nil {
		return fmt.Sprintf("the capture of %q is not a valid PNG: %v", name, err)
	}

	differing, total, diff, ok := compareImages(golden, actual, tol)
	if !ok {
		gb, ab := golden.Bounds(), actual.Bounds()
		return fmt.Sprintf("screenshot %q is %dx%d, golden %s is %dx%d", name, ab.Dx(), ab.Dy(), goldenPath, gb.Dx(), gb.Dy())
	}
	if float64(differing) <= tol.Ratio*float64(total) {
		return ""
	}

	msg := fmt.Sprintf("screenshot %q differs from %s: %d of %d pixels (%.3f%%) differ by more than %d, allowed %.3f%%",
		name, goldenPath, differing, total, 100*float64(differing)/float64(total), tol.Channel, 100*tol.Ratio)
	if outDir != "" {
		var buf bytes.Buffer
		if err := png.Encode(&buf, diff); err == nil {
			if err := os.WriteFile(filepath.Join(outDir, name+".diff.png"), buf.Bytes(), 0644); err == nil {
				msg += fmt.Sprintf("\n  diff image: %s.diff.png in the test outputs", name)
			}
		}
	}
	return msg
}

// cdpSender is the part of CDPClient the screenshot collector needs.
type cdpSender interface {
	Send(method string, params map[string]interface{}) (json.RawMessage, error)
	Evaluate(expression string) (json.RawMessage, error)
}

// screenshotCollector serves the page's expectScreenshot calls and keeps what it captured.
type screenshotCollector struct {
	client cdpSender

	mu       sync.Mutex
	captured map[string][]byte
}

func newScreenshotCollector(client cdpSender) *screenshotCollector {
	return &screenshotCollector{client: client, captured: map[string][]byte{}}
}

// onBindingCalled handles a Runtime.bindingCalled event. It runs on the CDP read loop, which must
// not block on a command's response, so the capture happens on its own goroutine.
func (c *screenshotCollector) onBindingCalled(params json.RawMessage) {
	var ev struct {
		Name    string `json:"name"`
		Payload string `json:"payload"`
	}
	if err := json.Unmarshal(params, &ev); err != nil || ev.Name != screenshotBinding {
		return
	}
	go c.handle(ev.Payload)
}

// handle captures the requested region and settles the page's promise with the outcome.
func (c *screenshotCollector) handle(payload string) {
	req, err := parseScreenshotRequest(payload)
	if err == nil {
		err = c.capture(req)
	}
	msg := "null"
	if err != nil {
		quoted, _ := json.Marshal(err.Error())
		msg = string(quoted)
	}
	// A request that could not even be parsed has no id to answer; the page then times out its
	// test, which is the best that can be done for a payload the helper never produces.
	_, _ = c.client.Evaluate(fmt.Sprintf("window.__screenshotDone(%d, %s)", req.ID, msg))
}

func (c *screenshotCollector) capture(req screenshotRequest) error {
	c.mu.Lock()
	_, dup := c.captured[req.Name]
	c.mu.Unlock()
	if dup {
		return fmt.Errorf("screenshot %s was captured twice", req.Name)
	}

	res, err := c.client.Send("Page.captureScreenshot", map[string]interface{}{
		"format":                "png",
		"fromSurface":           true,
		"captureBeyondViewport": true,
		"clip": map[string]interface{}{
			"x": req.X, "y": req.Y, "width": req.Width, "height": req.Height, "scale": 1,
		},
	})
	if err != nil {
		return fmt.Errorf("capturing screenshot %s: %w", req.Name, err)
	}
	var out struct {
		Data string `json:"data"`
	}
	if err := json.Unmarshal(res, &out); err != nil {
		return fmt.Errorf("reading the capture of %s: %w", req.Name, err)
	}
	data, err := base64.StdEncoding.DecodeString(out.Data)
	if err != nil || len(data) == 0 {
		return fmt.Errorf("the capture of %s is empty or not base64", req.Name)
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	c.captured[req.Name] = data
	return nil
}

// results returns a copy of everything captured so far.
func (c *screenshotCollector) results() map[string][]byte {
	c.mu.Lock()
	defer c.mu.Unlock()
	out := make(map[string][]byte, len(c.captured))
	for k, v := range c.captured {
		out[k] = v
	}
	return out
}
