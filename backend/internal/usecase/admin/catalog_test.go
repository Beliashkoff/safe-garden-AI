package admin

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSlugify(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"latin", "Super Grow 5000", "super-grow-5000"},
		{"cyrillic", "Удобрение Зелёный Сад", "udobrenie-zelenyy-sad"},
		{"mixed punctuation", "NPK 10-10-10 (универсал)", "npk-10-10-10-universal"},
		{"collapses dashes", "a   b---c", "a-b-c"},
		{"empty", "!!!", ""},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, Slugify(tc.in))
		})
	}
}

func TestValidPassword(t *testing.T) {
	tests := []struct {
		name string
		pw   string
		want bool
	}{
		{"ok long with digit", "correct-horse-7-battery", true},
		{"too short", "abc12345", false},
		{"no digit", "onlylettershere!", false},
		{"no letter", "123456789012", false},
		{"cyrillic letters count", "пароль-длинный-123", true},
		{"over bcrypt cap", string(make([]byte, 80)), false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, validPassword(tc.pw))
		})
	}
}

func TestNormalizeProduct(t *testing.T) {
	valid := ProductInput{
		Name:      "Зелёный рост",
		ShortDesc: "Азотное удобрение для рассады",
		Category:  "минеральные",
		Problems:  []string{"nitrogen_deficiency"},
		Active:    true,
	}

	t.Run("derives slug from russian name", func(t *testing.T) {
		got, err := normalizeProduct(valid)
		require.NoError(t, err)
		assert.Equal(t, "zelenyy-rost", got.Slug)
	})

	t.Run("empty plants becomes nil (universal)", func(t *testing.T) {
		in := valid
		in.Plants = []string{"  ", ""}
		got, err := normalizeProduct(in)
		require.NoError(t, err)
		assert.Nil(t, got.Plants)
	})

	t.Run("plants are lowercased and trimmed", func(t *testing.T) {
		in := valid
		in.Plants = []string{" Tomato ", "огурец"}
		got, err := normalizeProduct(in)
		require.NoError(t, err)
		assert.Equal(t, []string{"tomato", "огурец"}, got.Plants)
	})

	tests := []struct {
		name   string
		mutate func(*ProductInput)
	}{
		{"missing name", func(p *ProductInput) { p.Name = "" }},
		{"missing short_desc", func(p *ProductInput) { p.ShortDesc = "" }},
		{"missing category", func(p *ProductInput) { p.Category = "" }},
		{"no problems", func(p *ProductInput) { p.Problems = nil }},
		{"unknown problem key", func(p *ProductInput) { p.Problems = []string{"bad_key"} }},
		{"duplicate problem key", func(p *ProductInput) {
			p.Problems = []string{"wilting", "wilting"}
		}},
		{"bad slug chars", func(p *ProductInput) { p.Slug = "no spaces" }},
		{"negative price", func(p *ProductInput) { price := int32(-1); p.PriceRub = &price }},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			in := valid
			in.Problems = append([]string(nil), valid.Problems...)
			tc.mutate(&in)
			_, err := normalizeProduct(in)
			assert.True(t, errors.Is(err, ErrValidation), "want ErrValidation, got %v", err)
		})
	}
}
