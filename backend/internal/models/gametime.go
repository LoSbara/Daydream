package models

import "fmt"

// GameTime rappresenta l'ora in-game (giorno, ora, minuto).
type GameTime struct {
	Day    int `json:"day"`
	Hour   int `json:"hour"`
	Minute int `json:"minute"`
}

// AddMinutes avanza il clock di n minuti, gestendo rollover ora/giorno.
func (gt GameTime) AddMinutes(n int) GameTime {
	total := gt.Day*1440 + gt.Hour*60 + gt.Minute + n
	if total < 0 {
		total = 0
	}
	day := total / 1440
	rem := total % 1440
	return GameTime{Day: day, Hour: rem / 60, Minute: rem % 60}
}

// TotalMinutes restituisce i minuti totali dall'inizio del gioco.
func (gt GameTime) TotalMinutes() int {
	return gt.Day*1440 + gt.Hour*60 + gt.Minute
}

// TimeOfDay restituisce la fascia oraria corrente.
func (gt GameTime) TimeOfDay() string {
	switch {
	case gt.Hour >= 6 && gt.Hour < 12:
		return "morning"
	case gt.Hour >= 12 && gt.Hour < 17:
		return "afternoon"
	case gt.Hour >= 17 && gt.Hour < 21:
		return "evening"
	default:
		return "night"
	}
}

// IsNight restituisce true se è notte (21:00–05:59).
func (gt GameTime) IsNight() bool {
	return gt.Hour >= 21 || gt.Hour < 6
}

// FormatDisplay restituisce una stringa leggibile tipo "Giorno 3, 14:30".
func (gt GameTime) FormatDisplay() string {
	return fmt.Sprintf("Giorno %d, %02d:%02d", gt.Day, gt.Hour, gt.Minute)
}
