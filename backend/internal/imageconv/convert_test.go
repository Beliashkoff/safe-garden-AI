package imageconv

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsHEIC(t *testing.T) {
	c := New()

	assert.True(t, c.IsHEIC("image/heic", nil))
	assert.True(t, c.IsHEIC("image/heif", nil))

	// ISO-BMFF ftyp box with a HEIC brand.
	magic := append([]byte{0, 0, 0, 0x18}, []byte("ftypheic")...)
	magic = append(magic, make([]byte, 8)...)
	assert.True(t, c.IsHEIC("", magic))

	assert.False(t, c.IsHEIC("image/jpeg", []byte{0xff, 0xd8, 0xff}))
	assert.False(t, c.IsHEIC("", []byte("short")))
	assert.False(t, c.IsHEIC("", append([]byte{0, 0, 0, 0x18}, []byte("ftypjpeg")...)))
}

func TestToJPEG_InvalidData(t *testing.T) {
	_, err := New().ToJPEG([]byte("definitely not a heic file"))
	require.Error(t, err)
}
