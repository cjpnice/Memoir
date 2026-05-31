package media

import (
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	"io"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	_ "image/png"

	xdraw "golang.org/x/image/draw"

	"memoir/internal/domain"
)

// Manager stores originals, thumbnails, and derived images.
type Manager struct {
	root string
}

// NewManager creates a media manager rooted at dir.
func NewManager(dir string) (*Manager, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	return &Manager{root: dir}, nil
}

// Root returns the media storage root.
func (m *Manager) Root() string {
	return m.root
}

// PublicURL converts a storage-relative path into a public /media URL.
func (m *Manager) PublicURL(rel string) string {
	return "/media/" + filepath.ToSlash(rel)
}

// DeleteRelativeURL removes a stored media file referenced by a public /media URL.
func (m *Manager) DeleteRelativeURL(publicURL string) error {
	if strings.TrimSpace(publicURL) == "" {
		return nil
	}
	rel := strings.TrimPrefix(publicURL, "/media/")
	if rel == publicURL || rel == "" {
		return nil
	}
	return os.Remove(filepath.Join(m.root, rel))
}

// SaveOriginal stores an uploaded image and generates a thumbnail plus metrics.
func (m *Manager) SaveOriginal(projectID, imageID, fileName string, reader io.Reader) (saved SavedImage, err error) {
	raw, err := io.ReadAll(reader)
	if err != nil {
		return SavedImage{}, err
	}

	decodeBytes := raw
	img, format, err := image.Decode(bytes.NewReader(decodeBytes))
	if err != nil && isHEICFile(fileName, raw) {
		decodeBytes, err = convertHEICToJPEG(raw, fileName)
		if err != nil {
			return SavedImage{}, err
		}
		img, format, err = image.Decode(bytes.NewReader(decodeBytes))
	}
	if err != nil {
		return SavedImage{}, fmt.Errorf("%s 不是可识别的图片格式，请使用 JPG、PNG、HEIC 或 HEIF", fileName)
	}

	ext := extForFormat(format)
	originalRel := filepath.Join("projects", projectID, "originals", imageID+ext)
	thumbRel := filepath.Join("projects", projectID, "thumbs", imageID+".jpg")

	originalAbs := filepath.Join(m.root, originalRel)
	thumbAbs := filepath.Join(m.root, thumbRel)

	if err := os.MkdirAll(filepath.Dir(originalAbs), 0o755); err != nil {
		return SavedImage{}, err
	}
	if err := os.MkdirAll(filepath.Dir(thumbAbs), 0o755); err != nil {
		return SavedImage{}, err
	}

	if err := os.WriteFile(originalAbs, decodeBytes, 0o644); err != nil {
		return SavedImage{}, err
	}
	if err := m.writeThumbnail(thumbAbs, img); err != nil {
		return SavedImage{}, err
	}

	metrics := computeMetrics(img, len(decodeBytes))

	bounds := img.Bounds()
	return SavedImage{
		FileName:     fileName,
		MimeType:     mimeForFormat(format),
		Width:        bounds.Dx(),
		Height:       bounds.Dy(),
		FileSize:     int64(len(decodeBytes)),
		OriginalURL:  m.PublicURL(originalRel),
		ThumbnailURL: m.PublicURL(thumbRel),
		Metrics:      metrics,
		StoragePath:  originalAbs,
		Format:       format,
	}, nil
}

// SaveDerived writes a cropped derivative based on a normalized crop box.
func (m *Manager) SaveDerived(projectID, imageID string, sourceImage image.Image, box domain.CropBox) (string, error) {
	crop := normalizedCropBox(sourceImage.Bounds(), box)
	if crop.Empty() {
		return "", errors.New("crop box is empty")
	}

	dst := image.NewRGBA(image.Rect(0, 0, crop.Dx(), crop.Dy()))
	draw.Draw(dst, dst.Bounds(), sourceImage, crop.Min, draw.Src)

	rel := filepath.Join("projects", projectID, "derived", imageID+"_crop.jpg")
	abs := filepath.Join(m.root, rel)
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		return "", err
	}
	file, err := os.Create(abs)
	if err != nil {
		return "", err
	}
	defer file.Close()
	if err := jpeg.Encode(file, dst, &jpeg.Options{Quality: 92}); err != nil {
		return "", err
	}
	return m.PublicURL(rel), nil
}

// SaveVariant writes a processed JPEG derivative and returns its public URL and dimensions.
func (m *Manager) SaveVariant(projectID, imageID, suffix string, sourceImage image.Image) (string, int, int, error) {
	suffix = strings.Trim(strings.TrimSpace(suffix), "_-/")
	if suffix == "" {
		suffix = "edit"
	}
	bounds := sourceImage.Bounds()
	dst := image.NewRGBA(image.Rect(0, 0, bounds.Dx(), bounds.Dy()))
	draw.Draw(dst, dst.Bounds(), sourceImage, bounds.Min, draw.Src)

	rel := filepath.Join("projects", projectID, "derived", imageID+"_"+suffix+".jpg")
	abs := filepath.Join(m.root, rel)
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		return "", 0, 0, err
	}
	file, err := os.Create(abs)
	if err != nil {
		return "", 0, 0, err
	}
	defer file.Close()
	if err := jpeg.Encode(file, dst, &jpeg.Options{Quality: 92}); err != nil {
		return "", 0, 0, err
	}
	return m.PublicURL(rel), bounds.Dx(), bounds.Dy(), nil
}

// SaveJPEGBytes writes model-generated image bytes as a derivative.
func (m *Manager) SaveJPEGBytes(projectID, imageID, suffix string, raw []byte) (string, int, int, error) {
	img, _, err := image.Decode(bytes.NewReader(raw))
	if err != nil {
		return "", 0, 0, err
	}
	return m.SaveVariant(projectID, imageID, suffix, img)
}

// StoragePathFromURL converts a public /media URL back to its local path.
func (m *Manager) StoragePathFromURL(publicURL string) (string, error) {
	rel := strings.TrimPrefix(publicURL, "/media/")
	if rel == publicURL || rel == "" {
		return "", errors.New("invalid media URL")
	}
	return filepath.Join(m.root, rel), nil
}

// ImageAdjustments stores simple non-destructive color controls.
type ImageAdjustments struct {
	Brightness int
	Contrast   int
	Saturation int
	Warmth     int
}

// Rotate returns a copy rotated to the nearest right angle.
func Rotate(src image.Image, degrees int) image.Image {
	normalized := ((degrees % 360) + 360) % 360
	bounds := src.Bounds()
	w, h := bounds.Dx(), bounds.Dy()
	switch normalized {
	case 90:
		dst := image.NewRGBA(image.Rect(0, 0, h, w))
		for y := 0; y < h; y++ {
			for x := 0; x < w; x++ {
				dst.Set(h-1-y, x, src.At(bounds.Min.X+x, bounds.Min.Y+y))
			}
		}
		return dst
	case 180:
		dst := image.NewRGBA(image.Rect(0, 0, w, h))
		for y := 0; y < h; y++ {
			for x := 0; x < w; x++ {
				dst.Set(w-1-x, h-1-y, src.At(bounds.Min.X+x, bounds.Min.Y+y))
			}
		}
		return dst
	case 270:
		dst := image.NewRGBA(image.Rect(0, 0, h, w))
		for y := 0; y < h; y++ {
			for x := 0; x < w; x++ {
				dst.Set(y, w-1-x, src.At(bounds.Min.X+x, bounds.Min.Y+y))
			}
		}
		return dst
	default:
		dst := image.NewRGBA(image.Rect(0, 0, w, h))
		draw.Draw(dst, dst.Bounds(), src, bounds.Min, draw.Src)
		return dst
	}
}

// Adjust applies lightweight brightness, contrast, saturation, and warmth changes.
func Adjust(src image.Image, adjustments ImageAdjustments) image.Image {
	bounds := src.Bounds()
	dst := image.NewRGBA(image.Rect(0, 0, bounds.Dx(), bounds.Dy()))
	brightness := float64(adjustments.Brightness) / 100
	contrast := 1 + float64(adjustments.Contrast)/100
	saturation := 1 + float64(adjustments.Saturation)/100
	warmth := float64(adjustments.Warmth) / 180

	for y := 0; y < bounds.Dy(); y++ {
		for x := 0; x < bounds.Dx(); x++ {
			r16, g16, b16, a16 := src.At(bounds.Min.X+x, bounds.Min.Y+y).RGBA()
			r := float64(r16) / 65535
			g := float64(g16) / 65535
			b := float64(b16) / 65535
			a := uint8(a16 / 257)

			r = ((r-0.5)*contrast + 0.5) + brightness + warmth
			g = ((g-0.5)*contrast + 0.5) + brightness
			b = ((b-0.5)*contrast + 0.5) + brightness - warmth

			luma := 0.299*r + 0.587*g + 0.114*b
			r = luma + (r-luma)*saturation
			g = luma + (g-luma)*saturation
			b = luma + (b-luma)*saturation

			dst.SetRGBA(x, y, color.RGBA{
				R: clampUnitToByte(r),
				G: clampUnitToByte(g),
				B: clampUnitToByte(b),
				A: a,
			})
		}
	}
	return dst
}

// LoadImage opens a stored image from a public URL path.
func (m *Manager) LoadImage(publicURL string) (image.Image, string, error) {
	abs := filepath.Join(m.root, strings.TrimPrefix(publicURL, "/media/"))
	file, err := os.Open(abs)
	if err != nil {
		return nil, "", err
	}
	defer file.Close()
	return image.Decode(file)
}

// LoadImageFromURL opens a stored image from a public URL path.
func (m *Manager) LoadImageFromURL(publicURL string) (image.Image, string, error) {
	return m.LoadImage(publicURL)
}

// LoadSourceImage opens the original image and returns the decoded image.
func (m *Manager) LoadSourceImage(publicURL string) (image.Image, string, error) {
	return m.LoadImage(publicURL)
}

// EncodeForModel loads an image, resizes it if needed, and returns base64 JPEG data.
func (m *Manager) EncodeForModel(publicURL string, maxEdge int, quality int) (base64Data string, mimeType string, err error) {
	src, _, err := m.LoadImage(publicURL)
	if err != nil {
		return "", "", err
	}
	if maxEdge <= 0 {
		maxEdge = 1280
	}
	if quality <= 0 || quality > 100 {
		quality = 86
	}

	bounds := src.Bounds()
	w := bounds.Dx()
	h := bounds.Dy()
	targetW := w
	targetH := h
	longest := max(w, h)
	if longest > maxEdge {
		scale := float64(maxEdge) / float64(longest)
		targetW = int(math.Round(float64(w) * scale))
		targetH = int(math.Round(float64(h) * scale))
	}

	dst := image.NewRGBA(image.Rect(0, 0, max(targetW, 1), max(targetH, 1)))
	xdraw.CatmullRom.Scale(dst, dst.Bounds(), src, bounds, xdraw.Over, nil)

	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, dst, &jpeg.Options{Quality: quality}); err != nil {
		return "", "", err
	}
	return base64.StdEncoding.EncodeToString(buf.Bytes()), "image/jpeg", nil
}

func (m *Manager) writeThumbnail(path string, src image.Image) error {
	dst := image.NewRGBA(image.Rect(0, 0, 480, 480))
	fillBackground(dst, color.RGBA{R: 14, G: 18, B: 31, A: 255})

	srcBounds := src.Bounds()
	scale := math.Min(480/float64(srcBounds.Dx()), 480/float64(srcBounds.Dy()))
	if scale > 1 {
		scale = 1
	}
	w := int(float64(srcBounds.Dx()) * scale)
	h := int(float64(srcBounds.Dy()) * scale)
	if w < 1 {
		w = 1
	}
	if h < 1 {
		h = 1
	}
	x := (480 - w) / 2
	y := (480 - h) / 2
	xdraw.CatmullRom.Scale(dst, image.Rect(x, y, x+w, y+h), src, srcBounds, xdraw.Over, nil)

	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()
	return jpeg.Encode(file, dst, &jpeg.Options{Quality: 84})
}

func normalizedCropBox(bounds image.Rectangle, box domain.CropBox) image.Rectangle {
	w := float64(bounds.Dx())
	h := float64(bounds.Dy())
	x := int(math.Round(box.X * w))
	y := int(math.Round(box.Y * h))
	cw := int(math.Round(box.W * w))
	ch := int(math.Round(box.H * h))
	rect := image.Rect(x, y, x+cw, y+ch).Intersect(bounds)
	return rect
}

func fillBackground(dst *image.RGBA, c color.Color) {
	draw.Draw(dst, dst.Bounds(), &image.Uniform{C: c}, image.Point{}, draw.Src)
}

func clampUnitToByte(value float64) uint8 {
	if value <= 0 {
		return 0
	}
	if value >= 1 {
		return 255
	}
	return uint8(math.Round(value * 255))
}

func extForFormat(format string) string {
	switch format {
	case "png":
		return ".png"
	default:
		return ".jpg"
	}
}

func mimeForFormat(format string) string {
	switch format {
	case "png":
		return "image/png"
	default:
		return "image/jpeg"
	}
}

func isHEICFile(fileName string, raw []byte) bool {
	name := strings.ToLower(fileName)
	if strings.HasSuffix(name, ".heic") || strings.HasSuffix(name, ".heif") {
		return true
	}
	if len(raw) < 12 {
		return false
	}
	brand := strings.ToLower(string(raw[4:12]))
	return strings.Contains(brand, "ftypheic") ||
		strings.Contains(brand, "ftypheif") ||
		strings.Contains(brand, "ftypheix") ||
		strings.Contains(brand, "ftyphevc") ||
		strings.Contains(brand, "ftypmif1") ||
		strings.Contains(brand, "ftypmsf1")
}

func convertHEICToJPEG(raw []byte, fileName string) ([]byte, error) {
	tempDir, err := os.MkdirTemp("", "memoir-heic-*")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(tempDir)

	inputExt := ".heic"
	if strings.HasSuffix(strings.ToLower(fileName), ".heif") {
		inputExt = ".heif"
	}
	inputPath := filepath.Join(tempDir, "input"+inputExt)
	outputPath := filepath.Join(tempDir, "output.jpg")
	if err := os.WriteFile(inputPath, raw, 0o600); err != nil {
		return nil, err
	}

	if converter, err := exec.LookPath("sips"); err == nil {
		if err := exec.Command(converter, "-s", "format", "jpeg", inputPath, "--out", outputPath).Run(); err == nil {
			return os.ReadFile(outputPath)
		}
	}
	if converter, err := exec.LookPath("magick"); err == nil {
		if err := exec.Command(converter, inputPath, outputPath).Run(); err == nil {
			return os.ReadFile(outputPath)
		}
	}
	if converter, err := exec.LookPath("heif-convert"); err == nil {
		if err := exec.Command(converter, inputPath, outputPath).Run(); err == nil {
			return os.ReadFile(outputPath)
		}
	}

	return nil, errors.New("HEIC/HEIF 图片需要系统转换工具支持：macOS 可用 sips，Linux/Docker 请安装 ImageMagick 或 heif-convert")
}
