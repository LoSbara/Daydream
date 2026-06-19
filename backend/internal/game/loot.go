package game

import (
	"daydream/internal/models"
	"fmt"
	"math/rand"
)

// ─── Rarity weights ──────────────────────────────────────────────────────────

type lootTable struct {
	rarities []string
	weights  []int
	goldMin  int
	goldMax  int
	itemProb int // 0-100
}

// Loot tables base (senza LUC). LUC modificherà i pesi runtime.
var baseLootTables = map[string]lootTable{
	"normal": {
		rarities: []string{"common", "uncommon", "rare", "epic"},
		weights:  []int{60, 30, 9, 1},
		goldMin:  10, goldMax: 40, itemProb: 45,
	},
	"elite": {
		rarities: []string{"uncommon", "rare", "epic", "legendary"},
		weights:  []int{35, 45, 18, 2},
		goldMin:  50, goldMax: 130, itemProb: 75,
	},
	"boss": {
		rarities: []string{"rare", "epic", "legendary"},
		weights:  []int{30, 50, 20},
		goldMin:  150, goldMax: 400, itemProb: 100,
	},
}

// applyLUCToWeights: LUC sposta peso da rarità basse a rarità alte.
// Per ogni punto di LUC oltre 5, +1% ai pesi alti, -1% ai pesi bassi.
func applyLUCToWeights(weights []int, luc int) []int {
	result := make([]int, len(weights))
	copy(result, weights)
	bonus := luc - 5
	if bonus <= 0 || len(result) < 2 {
		return result
	}
	// Sposta dal peso più basso al più alto (proporzionalmente)
	shift := bonus // ogni punto LUC oltre 5 vale 1% di shift
	for i := 0; i < len(result)-1 && shift > 0; i++ {
		take := minInt(shift, result[i]-1)
		if take <= 0 {
			continue
		}
		result[i] -= take
		result[len(result)-1] += take
		shift -= take
	}
	return result
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func weightedRarity(rarities []string, weights []int) string {
	total := 0
	for _, w := range weights {
		total += w
	}
	r := rand.Intn(total)
	for i, w := range weights {
		r -= w
		if r < 0 {
			return rarities[i]
		}
	}
	return rarities[len(rarities)-1]
}

// ─── Affix pools ─────────────────────────────────────────────────────────────

// stat range per rarity
var rarityStatRange = map[string][2]int{
	"common":    {1, 3},
	"uncommon":  {2, 5},
	"rare":      {4, 8},
	"epic":      {7, 14},
	"legendary": {12, 22},
}

// numero di affixes per rarity
var rarityAffixCount = map[string][2]int{
	"common":    {1, 1},
	"uncommon":  {2, 2},
	"rare":      {2, 3},
	"epic":      {3, 4},
	"legendary": {4, 5},
}

// affix pools per tipo di item
var affixPools = map[string][]string{
	"weapon":    {"STR", "DEX", "AGI", "TEC", "LUC"},
	"armor":     {"VIT", "AGI", "STR", "DEX"},
	"offhand":   {"VIT", "DEX", "TEC", "LUC"},
	"accessory": {"STR", "DEX", "AGI", "TEC", "VIT", "LUC"},
	"head":      {"VIT", "TEC", "LUC", "AGI"},
	"legs":      {"AGI", "VIT", "STR", "DEX"},
	"boots":     {"AGI", "DEX", "VIT", "STR"},
}

// stat "magiche" — il loro valore è influenzato da TEC
var magicalStats = map[string]bool{"TEC": true, "LUC": true}

// ─── Item naming ─────────────────────────────────────────────────────────────

// aggettivi per rarity
var rarityAdjectives = map[string][]string{
	"common":    {"Consumato", "Arrugginito", "Logoro", "Antico", "Grezzo"},
	"uncommon":  {"Robusto", "Affilato", "Temprato", "Potenziato", "Solido"},
	"rare":      {"Forgiato", "Inciso", "Brillante", "Autentico", "Raffinato"},
	"epic":      {"Arcano", "Maledetto", "Sacro", "Infuocato", "Leggendario"},
	"legendary": {"Primordiale", "Divino", "Eterno", "Cosmico", "Trascendente"},
}

// nome base per tipo + sub-tipo
var weaponNames = []string{"Spada", "Lama", "Ascia", "Mazza", "Daga", "Bastone", "Alabarda", "Sciabola"}
var armorNames = []string{"Corazza", "Tunica", "Maglia", "Veste", "Piastre"}
var offhandNames = []string{"Scudo", "Orbe", "Talismano", "Lanterna", "Parata"}
var accessoryNames = []string{"Anello", "Amuleto", "Ciondolo", "Bracciale", "Fibbia"}
var headNames = []string{"Elmo", "Cappuccio", "Casco", "Diadema", "Benda"}
var legsNames = []string{"Gambali", "Schinieri", "Pantaloni Rinforzati"}
var bootsNames = []string{"Stivali", "Scarpe Corazzate", "Sandali Magici"}

// suffissi per stat dominante
var statSuffixes = map[string][]string{
	"STR": {"della Potenza", "del Guerriero", "della Forza Bruta", "del Colosso"},
	"DEX": {"della Precisione", "del Cacciatore", "dell'Arciere", "del Tiratore"},
	"AGI": {"del Vento", "della Folgore", "dello Scattante", "dell'Ombra"},
	"TEC": {"dell'Arcanista", "del Mago", "del Saggio", "dell'Incantatore"},
	"VIT": {"del Guardiano", "della Resistenza", "del Bastione", "del Difensore"},
	"LUC": {"della Fortuna", "del Vagabondo", "dell'Avventuriero", "del Prescelto"},
}

func itemNamesFor(t string) []string {
	switch t {
	case "weapon":
		return weaponNames
	case "armor":
		return armorNames
	case "offhand":
		return offhandNames
	case "accessory":
		return accessoryNames
	case "head":
		return headNames
	case "legs":
		return legsNames
	case "boots":
		return bootsNames
	default:
		return []string{"Oggetto"}
	}
}

func generateName(itemType, rarity string, stats map[string]int) string {
	adj := rarityAdjectives[rarity][rand.Intn(len(rarityAdjectives[rarity]))]
	base := itemNamesFor(itemType)
	baseName := base[rand.Intn(len(base))]

	// trova la stat dominante per il suffisso
	topStat := ""
	topVal := 0
	for k, v := range stats {
		if v > topVal {
			topVal = v
			topStat = k
		}
	}

	suffix := ""
	if topStat != "" {
		sufPool := statSuffixes[topStat]
		if len(sufPool) > 0 {
			suffix = " " + sufPool[rand.Intn(len(sufPool))]
		}
	}

	return fmt.Sprintf("%s %s%s", adj, baseName, suffix)
}

// ─── Item types + slots ───────────────────────────────────────────────────────

type itemTypeDef struct {
	Type   string
	Slot   string
	Weight int // probabilità relativa
}

var itemTypeDefs = []itemTypeDef{
	{Type: "weapon", Slot: "weapon", Weight: 30},
	{Type: "armor", Slot: "chest", Weight: 25},
	{Type: "offhand", Slot: "offhand", Weight: 10},
	{Type: "accessory", Slot: "accessory_1", Weight: 15},
	{Type: "head", Slot: "head", Weight: 10},
	{Type: "legs", Slot: "legs", Weight: 5},
	{Type: "boots", Slot: "boots", Weight: 5},
}

func randomItemType() itemTypeDef {
	total := 0
	for _, d := range itemTypeDefs {
		total += d.Weight
	}
	r := rand.Intn(total)
	for _, d := range itemTypeDefs {
		r -= d.Weight
		if r < 0 {
			return d
		}
	}
	return itemTypeDefs[0]
}

// ─── Price per rarity ─────────────────────────────────────────────────────────

var rarityBasePrice = map[string]int{
	"common": 50, "uncommon": 200, "rare": 600,
	"epic": 2000, "legendary": 8000,
}

// ─── Core generation ─────────────────────────────────────────────────────────

// generateItem crea un item procedurale influenzato da LUC e TEC.
// tec: influenza la qualità degli affix magici
func generateItem(rarity string, level, tec int) models.Item {
	typeDef := randomItemType()

	// Quanti affixes?
	affixRange := rarityAffixCount[rarity]
	numAffixes := affixRange[0]
	if affixRange[1] > affixRange[0] {
		numAffixes += rand.Intn(affixRange[1]-affixRange[0]+1)
	}
	// TEC ogni 8 punti sopra 5 dà +1 affix, max +2
	tecBonus := (tec - 5) / 8
	if tecBonus < 0 {
		tecBonus = 0
	}
	if tecBonus > 2 {
		tecBonus = 2
	}
	numAffixes += tecBonus
	// Cap per rarity
	maxAffixes := affixRange[1] + 2
	if numAffixes > maxAffixes {
		numAffixes = maxAffixes
	}

	// Pool affix per questo tipo
	pool := affixPools[typeDef.Type]
	if pool == nil {
		pool = affixPools["accessory"]
	}

	// Genera affix unici
	statBonus := map[string]int{}
	usedStats := map[string]bool{}
	statRange := rarityStatRange[rarity]

	for i := 0; i < numAffixes; i++ {
		// Scegli stat non già usata
		available := []string{}
		for _, s := range pool {
			if !usedStats[s] {
				available = append(available, s)
			}
		}
		if len(available) == 0 {
			break
		}
		stat := available[rand.Intn(len(available))]
		usedStats[stat] = true

		// Valore base
		baseVal := statRange[0] + rand.Intn(statRange[1]-statRange[0]+1)

		// TEC moltiplica le stat magiche
		if magicalStats[stat] {
			tecMult := 1.0 + float64(tec)*0.03
			baseVal = int(float64(baseVal) * tecMult)
		}

		// Level scaling (lieve)
		levelBonus := level / 5
		baseVal += levelBonus

		statBonus[stat] = baseVal
	}

	name := generateName(typeDef.Type, rarity, statBonus)

	return models.Item{
		ID:        fmt.Sprintf("%s_%s_%d", typeDef.Type, rarity, rand.Intn(100000)),
		Name:      name,
		Type:      typeDef.Type,
		Slot:      typeDef.Slot,
		StatBonus: statBonus,
		Rarity:    rarity,
		Price:     rarityBasePrice[rarity] * (1 + level/5),
		Quantity:  1,
		Appraised: rarity == "common",
	}
}

// ─── GenerateLoot (FIRMA AGGIORNATA) ─────────────────────────────────────────

// GenerateLoot genera loot per un nemico dato il livello e le stat del personaggio.
// luc: influenza la rarity del drop (ogni punto sopra 5 migliora le chance)
// tec: influenza la qualità degli affix magici
func GenerateLoot(enemy *models.Enemy, charLevel, luc, tec int) models.LootResult {
	if enemy == nil {
		return models.LootResult{}
	}

	tier := enemy.Tier
	if tier == "" {
		tier = "normal"
	}

	table, ok := baseLootTables[tier]
	if !ok {
		table = baseLootTables["normal"]
	}

	// Gold con scaling livello
	goldRange := table.goldMax - table.goldMin
	gold := table.goldMin + rand.Intn(goldRange+1)
	levelMult := 1.0 + float64(charLevel-1)*0.12
	gold = int(float64(gold) * levelMult)

	// LUC bonus oro: ogni punto LUC sopra 5 +1% oro
	lucGoldBonus := 1.0 + float64(luc-5)*0.01
	if lucGoldBonus > 1.25 {
		lucGoldBonus = 1.25
	}
	if lucGoldBonus > 0 {
		gold = int(float64(gold) * lucGoldBonus)
	}

	var items []models.Item

	// Drop item?
	dropRoll := rand.Intn(100)
	// LUC aumenta la probabilità di drop: +1% per punto LUC sopra 5
	lucItemBonus := luc - 5
	if lucItemBonus < 0 {
		lucItemBonus = 0
	}
	effectiveProb := table.itemProb + lucItemBonus
	if effectiveProb > 95 {
		effectiveProb = 95
	}

	if dropRoll < effectiveProb {
		weights := applyLUCToWeights(table.weights, luc)
		rarity := weightedRarity(table.rarities, weights)
		items = append(items, generateItem(rarity, charLevel, tec))
	}

	// Boss: item extra garantito
	if tier == "boss" {
		weights := applyLUCToWeights(table.weights, luc)
		rarity := weightedRarity(table.rarities, weights)
		// L'extra item è sempre almeno raro
		if rarity == "uncommon" || rarity == "common" {
			rarity = "rare"
		}
		items = append(items, generateItem(rarity, charLevel, tec))
	}

	return models.LootResult{Gold: gold, Items: items}
}
