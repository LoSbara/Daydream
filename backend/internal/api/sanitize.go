package api

import (
	"regexp"
	"strings"
	"unicode"
)

var htmlTagRe = regexp.MustCompile(`<[^>]*>`)

// sanitizeMessage pulisce il messaggio in ingresso dal giocatore prima di
// passarlo al GM. Non è rendering HTML, ma un input al prompt LLM: vogliamo
// rimuovere vettori di prompt injection ovvi e caratteri spazzatura.
func sanitizeMessage(s string) string {
	// Rimuovi tag HTML/XML (<script>, <b>, ecc.)
	s = htmlTagRe.ReplaceAllString(s, "")

	// Rimuovi caratteri di controllo (null byte, BEL, ESC, ecc.)
	// Mantieni: tabulazione (\t), newline (\n), carriage return (\r)
	s = strings.Map(func(r rune) rune {
		if unicode.IsControl(r) && r != '\t' && r != '\n' && r != '\r' {
			return -1
		}
		return r
	}, s)

	return strings.TrimSpace(s)
}
