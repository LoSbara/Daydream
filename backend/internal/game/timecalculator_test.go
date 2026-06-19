package game

import "testing"

func TestCalculateTimeElapsed_KnownCategories(t *testing.T) {
	cases := []struct {
		category string
		want     int
	}{
		{"conversation", 25},
		{"exploration", 30},
		{"rest", 420},
		{"crafting", 60},
		{"travel_regional", 240},
		{"travel_long", 1440},
	}

	for _, c := range cases {
		got := CalculateTimeElapsed(c.category, nil)
		if got != c.want {
			t.Errorf("categoria %q: got %d, want %d", c.category, got, c.want)
		}
	}
}

func TestCalculateTimeElapsed_CombatBonus(t *testing.T) {
	// combat base = 20; ogni ENEMY_DEAD aggiunge 15
	tags := []string{"ENEMY_DEAD", "ENEMY_DEAD"}
	got := CalculateTimeElapsed("combat", tags)
	expected := 20 + 2*15 // 50
	if got != expected {
		t.Errorf("combat con 2 ENEMY_DEAD: got %d, want %d", got, expected)
	}
}

func TestCalculateTimeElapsed_ExplorationDungeonMove(t *testing.T) {
	// exploration con 3 DUNGEON_MOVE_*: base diventa 3*20=60
	tags := []string{"DUNGEON_MOVE_NORTH", "DUNGEON_MOVE_EAST", "DUNGEON_MOVE_SOUTH"}
	got := CalculateTimeElapsed("exploration", tags)
	expected := 3 * 20 // 60
	if got != expected {
		t.Errorf("exploration con 3 mosse: got %d, want %d", got, expected)
	}
}

func TestCalculateTimeElapsed_ExplorationNoMove(t *testing.T) {
	// exploration senza DUNGEON_MOVE_*: usa base 30
	got := CalculateTimeElapsed("exploration", nil)
	if got != 30 {
		t.Errorf("exploration senza mosse: got %d, want 30", got)
	}
}

func TestCalculateTimeElapsed_UnknownCategoryInferredFromTags_Combat(t *testing.T) {
	// Categoria sconosciuta con ENEMY_DEAD → infer 30
	tags := []string{"ENEMY_DEAD"}
	got := CalculateTimeElapsed("unknown_action", tags)
	if got != 30 {
		t.Errorf("categoria sconosciuta con ENEMY_DEAD: got %d, want 30", got)
	}
}

func TestCalculateTimeElapsed_UnknownCategoryInferredFromTags_Regen(t *testing.T) {
	// Solo PLAYER_HP_+N senza nemici → riposo implicito, infer 120
	tags := []string{"PLAYER_HP_+20"}
	got := CalculateTimeElapsed("unknown_action", tags)
	if got != 120 {
		t.Errorf("categoria sconosciuta con regen: got %d, want 120", got)
	}
}

func TestCalculateTimeElapsed_MinimumFloor(t *testing.T) {
	// Caso degenere: nessun tag, categoria sconosciuta → infer restituisce 20, non sotto 10
	got := CalculateTimeElapsed("unknown_action", nil)
	if got < 10 {
		t.Errorf("valore sotto il floor di 10 minuti: %d", got)
	}
}
