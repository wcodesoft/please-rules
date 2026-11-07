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
	"testing"
)

func solid(w, h int, c color.NRGBA) *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.SetNRGBA(x, y, c)
		}
	}
	return img
}

func encodePNG(t *testing.T, img image.Image) []byte {
	t.Helper()
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

var (
	white = color.NRGBA{255, 255, 255, 255}
	black = color.NRGBA{0, 0, 0, 255}
)

func TestCompareImages(t *testing.T) {
	base := solid(10, 10, white)
	oneBlack := solid(10, 10, white)
	oneBlack.SetNRGBA(3, 4, black)
	slightlyOff := solid(10, 10, color.NRGBA{250, 255, 255, 255})

	tests := []struct {
		name          string
		actual        image.Image
		tol           pixelTolerance
		wantDiffering int
		wantOK        bool
	}{
		{name: "identical", actual: base, wantOK: true},
		{name: "one pixel", actual: oneBlack, wantDiffering: 1, wantOK: true},
		{name: "every pixel off by 5, zero tolerance", actual: slightlyOff, wantDiffering: 100, wantOK: true},
		{name: "every pixel off by 5, tolerance 5", actual: slightlyOff, tol: pixelTolerance{Channel: 5}, wantOK: true},
		{name: "different size", actual: solid(10, 11, white), wantOK: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			differing, total, diff, ok := compareImages(base, tt.actual, tt.tol)
			if ok != tt.wantOK || differing != tt.wantDiffering {
				t.Fatalf("compareImages() = differing %d ok %v, want %d %v", differing, ok, tt.wantDiffering, tt.wantOK)
			}
			if ok && (total != 100 || diff == nil) {
				t.Errorf("total %d diff %v", total, diff)
			}
		})
	}
}

func TestCheckScreenshots(t *testing.T) {
	goldenPNG := encodePNG(t, solid(8, 8, white))
	oneOff := solid(8, 8, white)
	oneOff.SetNRGBA(0, 0, black)

	tests := []struct {
		name     string
		goldens  map[string][]byte
		captured map[string][]byte
		tol      pixelTolerance
		want     map[string]string // result name -> substring of its error, "" for a pass
		wantDiff bool
	}{
		{
			name:     "match",
			goldens:  map[string][]byte{"a": goldenPNG},
			captured: map[string][]byte{"a": goldenPNG},
			want:     map[string]string{"screenshot: a": ""},
		},
		{
			name:     "one pixel differs, no tolerance",
			goldens:  map[string][]byte{"a": goldenPNG},
			captured: map[string][]byte{"a": encodePNG(t, oneOff)},
			want:     map[string]string{"screenshot: a": "1 of 64 pixels"},
			wantDiff: true,
		},
		{
			name:     "one pixel differs, ratio allows it",
			goldens:  map[string][]byte{"a": goldenPNG},
			captured: map[string][]byte{"a": encodePNG(t, oneOff)},
			tol:      pixelTolerance{Ratio: 0.02},
			want:     map[string]string{"screenshot: a": ""},
		},
		{
			name:     "size mismatch",
			goldens:  map[string][]byte{"a": goldenPNG},
			captured: map[string][]byte{"a": encodePNG(t, solid(8, 9, white))},
			want:     map[string]string{"screenshot: a": "is 8x9"},
		},
		{
			name:     "missing golden",
			captured: map[string][]byte{"a": goldenPNG},
			want:     map[string]string{"screenshot: a": "no golden"},
		},
		{
			name:     "stale golden",
			goldens:  map[string][]byte{"a": goldenPNG, "old": goldenPNG},
			captured: map[string][]byte{"a": goldenPNG},
			want:     map[string]string{"screenshot: a": "", "screenshot: old": "not captured"},
		},
		{
			name:     "corrupt golden",
			goldens:  map[string][]byte{"a": []byte("nope")},
			captured: map[string][]byte{"a": goldenPNG},
			want:     map[string]string{"screenshot: a": "not a valid PNG"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			goldenDir, outDir := t.TempDir(), filepath.Join(t.TempDir(), "out")
			var files []string
			for name, data := range tt.goldens {
				f := filepath.Join(goldenDir, name+".png")
				if err := os.WriteFile(f, data, 0644); err != nil {
					t.Fatal(err)
				}
				files = append(files, f)
			}

			results, err := checkScreenshots(files, outDir, tt.captured, tt.tol)
			if err != nil {
				t.Fatal(err)
			}
			if len(results) != len(tt.want) {
				t.Fatalf("got %d results %v, want %d", len(results), results, len(tt.want))
			}
			for _, r := range results {
				want, ok := tt.want[r.Name]
				if !ok {
					t.Errorf("unexpected result %q", r.Name)
				} else if want == "" && r.Error != "" || !strings.Contains(r.Error, want) {
					t.Errorf("%s: error %q, want it to contain %q", r.Name, r.Error, want)
				}
			}
			for name := range tt.captured {
				if _, err := os.Stat(filepath.Join(outDir, name+".png")); err != nil {
					t.Errorf("capture %s missing from the outputs: %v", name, err)
				}
			}
			_, diffErr := os.Stat(filepath.Join(outDir, "a.diff.png"))
			if (diffErr == nil) != tt.wantDiff {
				t.Errorf("diff image written = %v, want %v", diffErr == nil, tt.wantDiff)
			}
		})
	}
}

func TestCheckScreenshotsRejectsUnsafeNames(t *testing.T) {
	if _, err := checkScreenshots(nil, t.TempDir(), map[string][]byte{"../evil": nil}, pixelTolerance{}); err == nil {
		t.Error("a screenshot named ../evil was accepted")
	}
	if _, err := checkScreenshots([]string{"x/a.diff.png"}, "", nil, pixelTolerance{}); err == nil {
		t.Error("a golden named a.diff was accepted")
	}
}

func TestParseScreenshotRequest(t *testing.T) {
	tests := []struct {
		payload string
		wantErr bool
	}{
		{payload: `{"id":1,"name":"home","x":0,"y":0,"width":10,"height":20}`},
		{payload: `{"id":1,"name":"../x","x":0,"y":0,"width":10,"height":20}`, wantErr: true},
		{payload: `{"id":1,"name":"home.diff","x":0,"y":0,"width":10,"height":20}`, wantErr: true},
		{payload: `{"id":1,"name":"home","x":0,"y":0,"width":0,"height":20}`, wantErr: true},
		{payload: `nope`, wantErr: true},
	}
	for _, tt := range tests {
		if _, err := parseScreenshotRequest(tt.payload); (err != nil) != tt.wantErr {
			t.Errorf("parseScreenshotRequest(%s) error = %v, wantErr %v", tt.payload, err, tt.wantErr)
		}
	}
}

// fakeSender stands in for the browser: it answers captures with a fixed PNG and records the
// expressions evaluated in the page.
type fakeSender struct {
	mu        sync.Mutex
	png       []byte
	sendErr   error
	evaluated chan string
	clips     []map[string]interface{}
}

func (f *fakeSender) Send(method string, params map[string]interface{}) (json.RawMessage, error) {
	if method != "Page.captureScreenshot" {
		return nil, fmt.Errorf("unexpected method %s", method)
	}
	f.mu.Lock()
	f.clips = append(f.clips, params["clip"].(map[string]interface{}))
	f.mu.Unlock()
	if f.sendErr != nil {
		return nil, f.sendErr
	}
	return json.Marshal(map[string]string{"data": base64.StdEncoding.EncodeToString(f.png)})
}

func (f *fakeSender) Evaluate(expression string) (json.RawMessage, error) {
	f.evaluated <- expression
	return nil, nil
}

func bindingEvent(payload string) json.RawMessage {
	raw, _ := json.Marshal(map[string]string{"name": screenshotBinding, "payload": payload})
	return raw
}

func TestScreenshotCollector(t *testing.T) {
	img := encodePNG(t, solid(4, 4, white))
	req := `{"id":3,"name":"home","x":1,"y":2,"width":4,"height":4}`

	t.Run("captures the requested clip and answers the page", func(t *testing.T) {
		f := &fakeSender{png: img, evaluated: make(chan string, 1)}
		c := newScreenshotCollector(f)
		c.onBindingCalled(bindingEvent(req))
		if got := <-f.evaluated; got != "window.__screenshotDone(3, null)" {
			t.Errorf("page was told %q", got)
		}
		if !bytes.Equal(c.results()["home"], img) {
			t.Error("the capture was not stored")
		}
		if clip := f.clips[0]; clip["x"] != 1 || clip["y"] != 2 || clip["width"] != 4 || clip["height"] != 4 {
			t.Errorf("clip = %v", clip)
		}
	})

	t.Run("a capture failure rejects the page's promise", func(t *testing.T) {
		f := &fakeSender{sendErr: fmt.Errorf("boom"), evaluated: make(chan string, 1)}
		c := newScreenshotCollector(f)
		c.onBindingCalled(bindingEvent(req))
		if got := <-f.evaluated; !strings.Contains(got, `window.__screenshotDone(3, "capturing screenshot home: boom")`) {
			t.Errorf("page was told %q", got)
		}
		if len(c.results()) != 0 {
			t.Error("a failed capture was stored")
		}
	})

	t.Run("a second capture of the same name is rejected", func(t *testing.T) {
		f := &fakeSender{png: img, evaluated: make(chan string, 2)}
		c := newScreenshotCollector(f)
		c.onBindingCalled(bindingEvent(req))
		<-f.evaluated
		c.onBindingCalled(bindingEvent(req))
		if got := <-f.evaluated; !strings.Contains(got, "captured twice") {
			t.Errorf("page was told %q", got)
		}
	})

	t.Run("other bindings are ignored", func(t *testing.T) {
		f := &fakeSender{png: img, evaluated: make(chan string, 1)}
		c := newScreenshotCollector(f)
		raw, _ := json.Marshal(map[string]string{"name": "somethingElse", "payload": req})
		c.onBindingCalled(raw)
		select {
		case got := <-f.evaluated:
			t.Errorf("an unrelated binding triggered %q", got)
		default:
		}
	})
}
