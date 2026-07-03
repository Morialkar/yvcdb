package i18n

import "fmt"

type Localizer struct{ Language string }

func New(language string) Localizer {
	if language != "fr" {
		language = "en"
	}
	return Localizer{Language: language}
}

var messages = map[string][2]string{
	"app.title":            {"YVCDB — Your Vibe Code Deserves Better", "YVCDB — Ton Vibe Code Mérite Mieux"},
	"app.subtitle":         {"Claude Code / Codex CLI · Automated refactoring loop", "Claude Code / Codex CLI · Boucle de refactoring"},
	"pipeline":             {"Pipeline:", "Pipeline :"},
	"model.title":          {"%s model to use", "Modèle %s à utiliser"},
	"model.help":           {"Enter a model alias or full model ID for the configured provider.", "Entrez un alias ou l'identifiant complet d'un modèle pour le fournisseur configuré."},
	"model.warning":        {"Cost and plan usage depend on the selected model.", "Le coût et la consommation du plan dépendent du modèle choisi."},
	"model.prompt":         {"Model > ", "Modèle > "},
	"confirm.quit":         {"[enter] Confirm   [esc] Quit", "[entrée] Confirmer   [esc] Quitter"},
	"git.missing":          {"⚠  No git repository found in %s", "⚠  Pas de dépôt git détecté dans %s"},
	"git.init_question":    {"Initialize git now? (recommended)", "Initialiser git maintenant ? (recommandé)"},
	"git.yes":              {"Yes — init + snapshot", "Oui — init + snapshot"},
	"git.no":               {"No — continue without git", "Non — continuer sans git"},
	"feedback.title":       {"Reply to the agent with refinement instructions", "Répondre à l'agent avec une consigne de raffinement"},
	"feedback.help":        {"This response will be injected into a new iteration of the current phase.", "Cette réponse sera injectée dans une nouvelle itération de la phase courante."},
	"feedback.placeholder": {"Describe precisely what must be corrected…", "Décris précisément ce qui doit être corrigé…"},
	"feedback.send":        {"[enter] Send   [esc] Cancel", "[entrée] Envoyer   [esc] Annuler"},
	"iteration":            {"Iteration %d | %s", "Itération %d | %s"},
	"fix.round":            {"▶ Interactive fix — round %d", "▶ Correction interactive — round %d"},
	"decision.question":    {"Is the %s result satisfactory?", "Résultat de %s satisfaisant ?"},
	"decision.approve":     {"Approved", "Approuvé"},
	"decision.retry":       {"Retry", "Réitérer"},
	"decision.refine":      {"Reply/refine", "Répondre/raffiner"},
	"decision.skip":        {"Skip", "Skip"},
	"decision.quit":        {"Quit", "Quitter"},
	"fix.name":             {"fix", "la correction"},
	"tabs.help":            {"(tab/1-3 to switch)", "(tab/1-3 pour changer)"},
	"checklist.title":      {"📋 Final checklist — failed criteria can be fixed interactively", "📋 Checklist finale — les critères échoués pourront être corrigés interactivement"},
	"check.yesno":          {"[y] yes  [n] no", "[o] oui  [n] non"},
	"done.ready":           {"🎉 %d/%d — Code is production-ready!", "🎉 %d/%d — Code prêt pour production !"},
	"done.score":           {"⚠  %d/%d criteria satisfied.", "⚠  %d/%d critères satisfaits."},
	"done.failed":          {"Failed criteria:", "Critères échoués :"},
	"done.fix":             {"Start an interactive fix for failed criteria", "Lancer une correction interactive des critères échoués"},
	"done.logs":            {"Logs: %s | Session: %s", "Logs : %s | Session : %s"},
	"done.quit":            {"[q] Quit", "[q] Quitter"},
	"merge.failed":         {"⚠ Merge failures:\n%s", "⚠ Merges échoués :\n%s"},
}

func (l Localizer) T(key string, args ...any) string {
	pair, ok := messages[key]
	if !ok {
		return key
	}
	value := pair[0]
	if l.Language == "fr" {
		value = pair[1]
	}
	if len(args) > 0 {
		return fmt.Sprintf(value, args...)
	}
	return value
}

func (l Localizer) Pick(en, fr string) string {
	if l.Language == "fr" {
		return fr
	}
	return en
}
