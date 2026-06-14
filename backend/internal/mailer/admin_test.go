package mailer

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRenderAdminCode(t *testing.T) {
	tests := []struct {
		name      string
		purpose   string
		wantIntro string
	}{
		{"setup", AdminCodePurposeSetup, "первичной настройки"},
		{"reset", AdminCodePurposeReset, "восстановления пароля"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			text, html, err := renderAdminCode(tc.purpose, "123456")
			require.NoError(t, err)
			assert.Contains(t, text, "123456")
			assert.Contains(t, text, tc.wantIntro)
			assert.Contains(t, html, "123456")
			assert.Contains(t, html, tc.wantIntro)
		})
	}
}

func TestAdminSubject(t *testing.T) {
	assert.Contains(t, adminSubject(AdminCodePurposeReset), "восстановления")
	assert.Contains(t, adminSubject(AdminCodePurposeSetup), "настройки")
}
