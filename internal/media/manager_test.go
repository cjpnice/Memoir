package media

import (
	"bytes"
	"image"
	"image/color"
	"image/jpeg"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSaveOriginalRejectsUnsupportedImageWithHelpfulError(t *testing.T) {
	manager, err := NewManager(t.TempDir())
	if err != nil {
		t.Fatalf("new manager: %v", err)
	}

	_, err = manager.SaveOriginal("project", "image", "notes.txt", strings.NewReader("not an image"))
	if err == nil {
		t.Fatalf("expected unsupported image error")
	}
	if !strings.Contains(err.Error(), "不是可识别的图片格式") {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(err.Error(), "JPG、PNG、HEIC 或 HEIF") {
		t.Fatalf("expected supported format hint, got: %v", err)
	}
}

func TestConvertHEICToJPEGMissingToolReturnsHelpfulError(t *testing.T) {
	t.Setenv("PATH", t.TempDir())

	_, err := convertHEICToJPEG(bytes.Repeat([]byte{0}, 32), "sample.heic")
	if err == nil {
		t.Fatalf("expected missing HEIC converter error")
	}
	if !strings.Contains(err.Error(), "HEIC/HEIF 图片需要系统转换工具支持") {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(err.Error(), "ImageMagick") || !strings.Contains(err.Error(), "heif-convert") {
		t.Fatalf("expected converter install hint, got: %v", err)
	}
}

func TestSaveOriginalUsesHEICConverterFallback(t *testing.T) {
	root := t.TempDir()
	manager, err := NewManager(root)
	if err != nil {
		t.Fatalf("new manager: %v", err)
	}

	fixturePath := filepath.Join(root, "converted.jpg")
	if err := os.WriteFile(fixturePath, testJPEG(t), 0o644); err != nil {
		t.Fatalf("write jpeg fixture: %v", err)
	}
	binDir := filepath.Join(root, "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatalf("create fake bin dir: %v", err)
	}
	fakeSips := filepath.Join(binDir, "sips")
	script := `#!/bin/sh
out=""
want_out="false"
for arg in "$@"; do
  if [ "$want_out" = "true" ]; then
    out="$arg"
    want_out="false"
    continue
  fi
  if [ "$arg" = "--out" ]; then
    want_out="true"
  fi
done
/bin/cp "$MEMOIR_FAKE_JPEG" "$out"
`
	if err := os.WriteFile(fakeSips, []byte(script), 0o755); err != nil {
		t.Fatalf("write fake sips: %v", err)
	}
	t.Setenv("PATH", binDir)
	t.Setenv("MEMOIR_FAKE_JPEG", fixturePath)

	saved, err := manager.SaveOriginal("project", "image", "sample.heic", bytes.NewReader([]byte("not a real heic file")))
	if err != nil {
		t.Fatalf("save converted heic original: %v", err)
	}
	if saved.MimeType != "image/jpeg" || saved.Format != "jpeg" {
		t.Fatalf("expected converted jpeg metadata, got mime=%q format=%q", saved.MimeType, saved.Format)
	}
	if saved.Width != 6 || saved.Height != 4 {
		t.Fatalf("expected converted image dimensions 6x4, got %dx%d", saved.Width, saved.Height)
	}
	if !strings.HasSuffix(saved.OriginalURL, ".jpg") {
		t.Fatalf("expected converted original to be stored as jpg, got %s", saved.OriginalURL)
	}
	if _, err := os.Stat(filepath.Join(root, strings.TrimPrefix(saved.OriginalURL, "/media/"))); err != nil {
		t.Fatalf("expected converted original to exist: %v", err)
	}
}

func TestMultipleJPEGsCanBeImported(t *testing.T) {
	manager, err := NewManager(t.TempDir())
	if err != nil {
		t.Fatalf("new manager: %v", err)
	}

	fixtures := map[string][]byte{
		"rainy-walk.jpg":   testJPEG(t),
		"album-cover.jpg":  testJPEGWithColor(t, color.RGBA{R: 26, G: 92, B: 70, A: 255}),
		"detail-frame.jpg": testJPEGWithColor(t, color.RGBA{R: 156, G: 78, B: 55, A: 255}),
	}
	for name, raw := range fixtures {
		saved, err := manager.SaveOriginal("demo", strings.TrimSuffix(name, ".jpg"), name, bytes.NewReader(raw))
		if err != nil {
			t.Fatalf("import fixture %s: %v", name, err)
		}
		if saved.Width == 0 || saved.Height == 0 {
			t.Fatalf("fixture %s is missing dimensions: %#v", name, saved)
		}
		if saved.ThumbnailURL == "" || saved.OriginalURL == "" {
			t.Fatalf("fixture %s did not produce expected media metadata: %#v", name, saved)
		}
	}
}

func testJPEG(t *testing.T) []byte {
	return testJPEGWithColor(t, color.RGBA{R: 80, G: 120, B: 160, A: 255})
}

func testJPEGWithColor(t *testing.T, fill color.RGBA) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 6, 4))
	for y := 0; y < 4; y++ {
		for x := 0; x < 6; x++ {
			img.SetRGBA(x, y, color.RGBA{
				R: fill.R + uint8(x*12),
				G: fill.G + uint8(y*18),
				B: fill.B + uint8(x*y),
				A: 255,
			})
		}
	}
	var out bytes.Buffer
	if err := jpeg.Encode(&out, img, &jpeg.Options{Quality: 90}); err != nil {
		t.Fatalf("encode jpeg: %v", err)
	}
	return out.Bytes()
}
