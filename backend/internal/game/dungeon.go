package game

import (
	"fmt"
	"math/rand"

	"daydream/internal/models"
)

// dungeonTemplates definisce i dungeon disponibili per nome e tema.
var dungeonTemplates = map[string]struct {
	Name        string
	RoomPrefixes []string
}{
	"caverna_oscura": {
		Name:        "Caverna Oscura",
		RoomPrefixes: []string{"Corridoio", "Grotta", "Anfratto", "Sala", "Passaggio"},
	},
	"torre_abbandonata": {
		Name:        "Torre Abbandonata",
		RoomPrefixes: []string{"Piano", "Camera", "Laboratorio", "Archivio", "Terrazza"},
	},
	"rovine_antiche": {
		Name:        "Rovine Antiche",
		RoomPrefixes: []string{"Tempio", "Cripta", "Salone", "Altare", "Corridoio"},
	},
}

var roomSuffixes = []string{
	"delle Ombre", "del Silenzio", "dei Caduti", "Maledetta", "Dimenticata",
	"Segreta", "Profonda", "del Caos", "dell'Oscurità", "Abbandonata",
}

var roomDescriptions = []string{
	"L'aria è pesante e maleodorante. Le pareti sembrano pulsare nell'oscurità.",
	"Tracce di sangue secco disegnano strani simboli sul pavimento.",
	"Un silenzio innaturale avvolge questo luogo. Anche i tuoi passi sembrano attutiti.",
	"Resti di equipaggiamento rotto giacciono sparpagliati. Qualcuno è già stato qui.",
	"La luce fatica a penetrare qui dentro. Qualcosa si muove nelle ombre.",
	"Le pareti sono coperte di incisioni incomprensibili che sembrano seguirti con lo sguardo.",
	"Un odore ferroso di sangue vecchio permea l'aria. Tieni alta la guardia.",
	"Questo luogo emana un'energia oscura. I tuoi sensi sono in allerta massima.",
}

// GenerateDungeon crea un dungeon procedurale con N stanze collegate.
// difficulty 1-5 controlla numero stanze (5-15) e densità nemici.
func GenerateDungeon(dungeonID string, characterLevel, difficulty int) (*models.ActiveDungeon, error) {
	tmpl, ok := dungeonTemplates[dungeonID]
	if !ok {
		return nil, fmt.Errorf("dungeon %q non trovato", dungeonID)
	}
	if difficulty < 1 {
		difficulty = 1
	}
	if difficulty > 5 {
		difficulty = 5
	}

	numRooms := 4 + difficulty*2 // 6-14 stanze
	enemyChance := 0.4 + float64(difficulty)*0.08 // 48%-80%

	rooms := make(map[string]models.DungeonRoom, numRooms)
	roomOrder := make([]string, 0, numRooms)

	// Genera stanze in sequenza lineare (backpacking style)
	for i := 0; i < numRooms; i++ {
		id := fmt.Sprintf("room_%02d", i)
		roomOrder = append(roomOrder, id)

		prefix := tmpl.RoomPrefixes[rand.Intn(len(tmpl.RoomPrefixes))]
		suffix := roomSuffixes[rand.Intn(len(roomSuffixes))]
		desc := roomDescriptions[rand.Intn(len(roomDescriptions))]

		room := models.DungeonRoom{
			ID:          id,
			Name:        fmt.Sprintf("%s %s", prefix, suffix),
			Description: desc,
			Exits:       map[string]string{},
			IsEntrance:  i == 0,
			IsBoss:      i == numRooms-1,
			Cleared:     i == 0, // l'ingresso è sempre sicuro
		}

		// Nemici: no nella stanza d'ingresso, boss nell'ultima, random nelle altre
		if i == 0 {
			room.HasEnemy = false
		} else if i == numRooms-1 {
			room.HasEnemy = true
			room.EnemyTier = "boss"
		} else if rand.Float64() < enemyChance {
			room.HasEnemy = true
			if rand.Float64() < 0.15 { // 15% chance elite
				room.EnemyTier = "elite"
			} else {
				room.EnemyTier = "normal"
			}
		}

		rooms[id] = room
	}

	// Collega le stanze: catena lineare (sempre avanti/indietro)
	// Le direzioni sono generate randomicamente ma coerenti tra stanze collegate
	dirPairs := [][2]string{
		{"nord", "sud"},
		{"est", "ovest"},
	}
	for i := 0; i < numRooms-1; i++ {
		pair := dirPairs[i%len(dirPairs)]
		forward, back := pair[0], pair[1]

		curr := rooms[roomOrder[i]]
		next := rooms[roomOrder[i+1]]

		curr.Exits[forward] = roomOrder[i+1]
		next.Exits[back] = roomOrder[i]

		rooms[roomOrder[i]] = curr
		rooms[roomOrder[i+1]] = next
	}

	// La prima stanza è già visited (il personaggio è entrato)
	entrance := rooms[roomOrder[0]]
	entrance.Visited = true
	rooms[roomOrder[0]] = entrance

	return &models.ActiveDungeon{
		ID:          dungeonID,
		Name:        tmpl.Name,
		Difficulty:  difficulty,
		CurrentRoom: roomOrder[0],
		Rooms:       rooms,
		EnteredAt:   0, // verrà impostato dal caller con il TurnID corrente
	}, nil
}

// MoveInDungeon sposta il personaggio nella direzione indicata.
// Restituisce la nuova stanza e un eventuale errore se la direzione non esiste.
func MoveInDungeon(dungeon *models.ActiveDungeon, direction string) (*models.DungeonRoom, error) {
	curr, ok := dungeon.Rooms[dungeon.CurrentRoom]
	if !ok {
		return nil, fmt.Errorf("stanza corrente %q non trovata", dungeon.CurrentRoom)
	}

	nextID, ok := curr.Exits[direction]
	if !ok {
		return nil, fmt.Errorf("nessuna uscita verso %q da questa stanza", direction)
	}

	next := dungeon.Rooms[nextID]
	next.Visited = true
	dungeon.Rooms[nextID] = next
	dungeon.CurrentRoom = nextID

	return &next, nil
}

// CurrentDungeonRoom restituisce la stanza corrente del dungeon.
func CurrentDungeonRoom(dungeon *models.ActiveDungeon) *models.DungeonRoom {
	if dungeon == nil {
		return nil
	}
	room, ok := dungeon.Rooms[dungeon.CurrentRoom]
	if !ok {
		return nil
	}
	return &room
}

// DiscoveredRooms restituisce solo le stanze già visitate dal giocatore.
func DiscoveredRooms(dungeon *models.ActiveDungeon) []models.DungeonRoom {
	var result []models.DungeonRoom
	for _, r := range dungeon.Rooms {
		if r.Visited {
			result = append(result, r)
		}
	}
	return result
}

// AvailableDungeons restituisce la lista dei dungeon disponibili.
func AvailableDungeons() []map[string]any {
	var list []map[string]any
	for id, tmpl := range dungeonTemplates {
		list = append(list, map[string]any{
			"id":   id,
			"name": tmpl.Name,
		})
	}
	return list
}
