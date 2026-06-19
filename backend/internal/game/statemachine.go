package game

import (
	"fmt"
	"daydream/internal/models"
)

// transitionTable definisce le transizioni lecite tra stati di gioco.
// Chiave: stato corrente. Valore: set di stati raggiungibili.
var transitionTable = map[models.GameStateEnum]map[models.GameStateEnum]bool{
	models.StateWorldNavigation: {
		models.StateCombat:           true,
		models.StateDungeonExplore:   true,
	},
	models.StateCombat: {
		models.StateWorldNavigation: true,
	},
	models.StateDungeonExplore: {
		models.StateWorldNavigation: true,
		models.StateDungeonCombat:   true,
	},
	models.StateDungeonCombat: {
		models.StateDungeonExplore: true,
	},
}

// ValidateTransition verifica che la transizione da→verso sia lecita.
func ValidateTransition(from, to models.GameStateEnum) error {
	allowed, ok := transitionTable[from]
	if !ok {
		return fmt.Errorf("stato di partenza sconosciuto: %s", from)
	}
	if !allowed[to] {
		return fmt.Errorf("transizione %s→%s non consentita", from, to)
	}
	return nil
}

// ApplyGameState aggiorna il session con il nuovo stato, validando la transizione.
// Se il GM propone uno stato non raggiungibile, la funzione lo ignora e logga un warning.
func ApplyGameState(session *models.GameSession, update *models.GameStateUpdate) {
	if update == nil {
		return
	}

	if update.Location != "" {
		session.Location = update.Location
	}
	if update.SubLocation != "" {
		session.SubLocation = update.SubLocation
	}
	if update.ZoneType != "" {
		session.ZoneType = update.ZoneType
	}

	// Gestione combattimento
	if update.CombatActive != nil {
		newCombat := *update.CombatActive

		if newCombat && !session.CombatActive {
			// Inizia combattimento
			if err := ValidateTransition(session.GameState, models.StateCombat); err == nil {
				session.GameState = models.StateCombat
				session.CombatActive = true
			}
			if update.CurrentEnemy != nil {
				session.CurrentEnemy = update.CurrentEnemy
			}
		} else if !newCombat && session.CombatActive {
			// Fine combattimento
			session.CombatActive = false
			session.CurrentEnemy = nil
			session.TacticalTension = 0

			switch session.GameState {
			case models.StateCombat:
				session.GameState = models.StateWorldNavigation
			case models.StateDungeonCombat:
				session.GameState = models.StateDungeonExplore
			}
		}
	}

	// Aggiornamento nemico corrente (durante il combattimento)
	if update.CurrentEnemy != nil && session.CombatActive {
		session.CurrentEnemy = update.CurrentEnemy
	}
}
