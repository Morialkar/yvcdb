package i18n

import "testing"

func TestEnglishIsDefault(t *testing.T) {
	if got := New("").T("pipeline"); got != "Pipeline:" {
		t.Fatalf("got %q", got)
	}
	if got := New("de").Language; got != "en" {
		t.Fatalf("got language %q", got)
	}
}

func TestFrenchTranslationAndFormatting(t *testing.T) {
	l := New("fr")
	if got := l.T("git.missing", "/tmp/project"); got != "⚠  Pas de dépôt git détecté dans /tmp/project" {
		t.Fatalf("got %q", got)
	}
}
