package game

import (
	"strings"
	"testing"

	"daydream/internal/models"
)

// ── parseGMResponse ───────────────────────────────────────────────────────────

func TestParseGMResponse_ValidJSON(t *testing.T) {
	raw := `{"narrative":"Il viaggio comincia.","action_category":"exploration"}`
	resp, err := parseGMResponse(raw)
	if err != nil {
		t.Fatalf("errore inatteso: %v", err)
	}
	if resp.Narrative != "Il viaggio comincia." {
		t.Errorf("narrative = %q, atteso %q", resp.Narrative, "Il viaggio comincia.")
	}
}

func TestParseGMResponse_MarkdownBlock(t *testing.T) {
	raw := "```json\n{\"narrative\":\"Testo narrativo.\",\"action_category\":\"combat\"}\n```"
	resp, err := parseGMResponse(raw)
	if err != nil {
		t.Fatalf("errore inatteso: %v", err)
	}
	if resp.Narrative != "Testo narrativo." {
		t.Errorf("narrative = %q, atteso %q", resp.Narrative, "Testo narrativo.")
	}
}

func TestParseGMResponse_MarkdownBlockNoClosingFence(t *testing.T) {
	raw := "```json\n{\"narrative\":\"Senza fence finale.\",\"action_category\":\"conversation\"}"
	resp, err := parseGMResponse(raw)
	if err != nil {
		t.Fatalf("dovrebbe recuperare anche senza ```: %v", err)
	}
	if resp.Narrative == "" {
		t.Error("narrative non deve essere vuota")
	}
}

func TestParseGMResponse_EmptyMarkdownBlock(t *testing.T) {
	// Solo ``` senza contenuto — precedentemente causava crash su lines[len(lines)-1]
	raw := "```\n```"
	_, err := parseGMResponse(raw)
	if err == nil {
		t.Fatal("atteso errore per JSON vuoto, ma non ne è stato restituito uno")
	}
}

func TestParseGMResponse_NoJSON(t *testing.T) {
	_, err := parseGMResponse("risposta in testo libero senza JSON")
	if err == nil {
		t.Fatal("atteso errore per mancanza di JSON")
	}
}

func TestParseGMResponse_MissingNarrative(t *testing.T) {
	raw := `{"action_category":"exploration","battle_tags":[]}`
	_, err := parseGMResponse(raw)
	if err == nil {
		t.Fatal("atteso errore per narrative mancante")
	}
	if !strings.Contains(err.Error(), "narrative") {
		t.Errorf("errore non menziona 'narrative': %v", err)
	}
}

func TestParseGMResponse_JSONWithLeadingText(t *testing.T) {
	raw := `Ecco la risposta: {"narrative":"Arriva il nemico.","action_category":"combat"} fine.`
	resp, err := parseGMResponse(raw)
	if err != nil {
		t.Fatalf("errore inatteso: %v", err)
	}
	if resp.Narrative != "Arriva il nemico." {
		t.Errorf("narrative = %q", resp.Narrative)
	}
}

// ── tickStatusEffect ──────────────────────────────────────────────────────────

func makeStats(hpMax, hpCurrent, mpMax, mpCurrent, stmMax, stmCurrent int) models.Stats {
	return models.Stats{
		HP:  models.Resource{Max: hpMax, Current: hpCurrent},
		MP:  models.Resource{Max: mpMax, Current: mpCurrent},
		STM: models.Resource{Max: stmMax, Current: stmCurrent},
	}
}

func se(name string) models.StatusEffect {
	return models.StatusEffect{Name: name, TurnsRemaining: 3}
}

func TestTickStatusEffect_Poison(t *testing.T) {
	stats := makeStats(100, 80, 100, 100, 100, 100)
	tickStatusEffect(&stats, se("POISON"))
	// POISON: -5% di MaxHP = max(1, 100*5/100) = max(1,5) = 5
	expected := 75
	if stats.HP.Current != expected {
		t.Errorf("HP dopo POISON = %d, atteso %d", stats.HP.Current, expected)
	}
}

func TestTickStatusEffect_Bleed(t *testing.T) {
	stats := makeStats(100, 80, 100, 100, 100, 100)
	tickStatusEffect(&stats, se("BLEED"))
	// BLEED: -8% = 8
	expected := 72
	if stats.HP.Current != expected {
		t.Errorf("HP dopo BLEED = %d, atteso %d", stats.HP.Current, expected)
	}
}

func TestTickStatusEffect_Burn(t *testing.T) {
	stats := makeStats(100, 80, 100, 100, 100, 100)
	tickStatusEffect(&stats, se("BURN"))
	// BURN: HP -6% = 6, MP -4% = 4
	if stats.HP.Current != 74 {
		t.Errorf("HP dopo BURN = %d, atteso 74", stats.HP.Current)
	}
	if stats.MP.Current != 96 {
		t.Errorf("MP dopo BURN = %d, atteso 96", stats.MP.Current)
	}
}

func TestTickStatusEffect_Regen(t *testing.T) {
	stats := makeStats(100, 50, 100, 100, 100, 100)
	tickStatusEffect(&stats, se("REGEN"))
	// REGEN: +8% = 8
	expected := 58
	if stats.HP.Current != expected {
		t.Errorf("HP dopo REGEN = %d, atteso %d", stats.HP.Current, expected)
	}
}

func TestTickStatusEffect_RegenCap(t *testing.T) {
	stats := makeStats(100, 98, 100, 100, 100, 100)
	tickStatusEffect(&stats, se("REGEN"))
	// 98 + 8 = 106, ma clampa a 100
	if stats.HP.Current != 100 {
		t.Errorf("HP dopo REGEN da quasi-full = %d, atteso 100", stats.HP.Current)
	}
}

func TestTickStatusEffect_RegenMP(t *testing.T) {
	stats := makeStats(100, 100, 100, 60, 100, 100)
	tickStatusEffect(&stats, se("REGEN_MP"))
	// REGEN_MP: +10% di MaxMP = 10
	if stats.MP.Current != 70 {
		t.Errorf("MP dopo REGEN_MP = %d, atteso 70", stats.MP.Current)
	}
}

func TestTickStatusEffect_RegenSTM(t *testing.T) {
	stats := makeStats(100, 100, 100, 100, 100, 40)
	tickStatusEffect(&stats, se("REGEN_STM"))
	// REGEN_STM: +10% di MaxSTM = 10
	if stats.STM.Current != 50 {
		t.Errorf("STM dopo REGEN_STM = %d, atteso 50", stats.STM.Current)
	}
}

func TestTickStatusEffect_MinimumOneTick(t *testing.T) {
	// Con MaxHP = 5, 5% = 0 → min clampa a 1
	stats := makeStats(5, 5, 10, 10, 10, 10)
	tickStatusEffect(&stats, se("POISON"))
	if stats.HP.Current != 4 {
		t.Errorf("HP dopo POISON su MaxHP=5 = %d, atteso 4", stats.HP.Current)
	}
}

func TestTickStatusEffect_PoisonFloorZero(t *testing.T) {
	// HP già a 1: POISON porta a 0, non negativo
	stats := makeStats(100, 1, 100, 100, 100, 100)
	tickStatusEffect(&stats, se("POISON"))
	if stats.HP.Current != 0 {
		t.Errorf("HP dopo POISON su HP=1 = %d, atteso 0", stats.HP.Current)
	}
}

func TestTickStatusEffect_UnknownEffect(t *testing.T) {
	// Effetti sconosciuti non devono modificare le stats
	stats := makeStats(100, 80, 100, 100, 100, 100)
	before := stats
	tickStatusEffect(&stats, se("UNKNOWN_BUFF"))
	if stats != before {
		t.Errorf("stats modificate da effetto sconosciuto: prima=%+v, dopo=%+v", before, stats)
	}
}
