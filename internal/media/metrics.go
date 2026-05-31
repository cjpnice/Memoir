package media

import (
	"image"
	"math"

	xdraw "golang.org/x/image/draw"

	"memoir/internal/domain"
)

// SavedImage contains the file metadata emitted by the media manager.
type SavedImage struct {
	FileName     string
	MimeType     string
	FileSize     int64
	Width        int
	Height       int
	OriginalURL  string
	ThumbnailURL string
	Metrics      domain.ImageMetrics
	StoragePath  string
	Format       string
}

func computeMetrics(src image.Image, fileSize int) domain.ImageMetrics {
	bounds := src.Bounds()
	gray := image.NewGray(image.Rect(0, 0, 64, 64))
	xdraw.NearestNeighbor.Scale(gray, gray.Bounds(), src, bounds, xdraw.Src, nil)

	var sum, sumSq, sharpness float64
	total := float64(len(gray.Pix))
	for y := 0; y < 64; y++ {
		for x := 0; x < 64; x++ {
			idx := y*gray.Stride + x
			v := float64(gray.Pix[idx]) / 255.0
			sum += v
			sumSq += v * v
			if x > 0 {
				prev := float64(gray.Pix[y*gray.Stride+x-1]) / 255.0
				sharpness += math.Abs(v - prev)
			}
			if y > 0 {
				prev := float64(gray.Pix[(y-1)*gray.Stride+x]) / 255.0
				sharpness += math.Abs(v - prev)
			}
		}
	}
	mean := sum / total
	variance := (sumSq / total) - mean*mean
	if variance < 0 {
		variance = 0
	}
	return domain.ImageMetrics{
		AspectRatio: float64(bounds.Dx()) / math.Max(float64(bounds.Dy()), 1),
		Brightness:  mean,
		Contrast:    math.Sqrt(variance),
		Sharpness:   sharpness / total,
		FileSize:    int64(fileSize),
		Width:       bounds.Dx(),
		Height:      bounds.Dy(),
	}
}
