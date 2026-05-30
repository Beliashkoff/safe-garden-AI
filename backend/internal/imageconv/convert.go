// Package imageconv normalizes uploaded images into a Claude-acceptable form.
// Claude accepts image/jpeg, image/png, image/webp and image/gif but NOT HEIC,
// so HEIC/HEIF is decoded and re-encoded to JPEG (ARCH §5). Decoding uses a
// pure-Go (WASM/libheif) decoder — no CGO — so the api binary stays statically
// buildable and needs no C toolchain in its image.
package imageconv

import (
	"bytes"
	"fmt"
	"image/jpeg"

	"github.com/gen2brain/heic"
)

const jpegQuality = 90

// Converter detects HEIC and converts it to JPEG.
type Converter struct{}

// New returns a Converter.
func New() Converter { return Converter{} }

// IsHEIC reports whether the content type or the bytes' ISO-BMFF ftyp brand
// indicate HEIC/HEIF.
func (Converter) IsHEIC(contentType string, data []byte) bool {
	if contentType == "image/heic" || contentType == "image/heif" {
		return true
	}
	return hasHEICMagic(data)
}

// ToJPEG decodes HEIC bytes and re-encodes them as JPEG.
func (Converter) ToJPEG(data []byte) ([]byte, error) {
	img, err := heic.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("imageconv: decode heic: %w", err)
	}
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: jpegQuality}); err != nil {
		return nil, fmt.Errorf("imageconv: encode jpeg: %w", err)
	}
	return buf.Bytes(), nil
}

// hasHEICMagic checks the ISO base media file format ftyp brand (bytes 4..12)
// for known HEIF/HEIC brands.
func hasHEICMagic(data []byte) bool {
	if len(data) < 12 || string(data[4:8]) != "ftyp" {
		return false
	}
	switch string(data[8:12]) {
	case "heic", "heix", "heif", "heim", "heis", "hevc", "hevx", "mif1", "msf1":
		return true
	default:
		return false
	}
}
