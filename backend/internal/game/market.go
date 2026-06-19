package game

import (
	"daydream/internal/models"
	"fmt"
	"math/rand"
)

// Tipi di mercato per location — influenzano il pool di item
var marketTypeByLocation = map[string][]string{
	"città":        {"weapon", "armor", "accessory", "head", "boots"},
	"mercato":      {"accessory", "head", "legs", "boots", "offhand"},
	"porto":        {"accessory", "offhand", "material"},
	"villaggio":    {"armor", "boots", "legs"},
	"accampamento": {"weapon", "offhand"},
}

var defaultMarketTypes = []string{"weapon", "armor", "accessory"}

func marketTypesForLocation(location string) []string {
	for key, types := range marketTypeByLocation {
		if containsCI(location, key) {
			return types
		}
	}
	return defaultMarketTypes
}

func containsCI(s, sub string) bool {
	if len(s) < len(sub) {
		return false
	}
	sl := []rune(s)
	subl := []rune(sub)
	for i := 0; i <= len(sl)-len(subl); i++ {
		match := true
		for j, c := range subl {
			if toLowerRune(sl[i+j]) != toLowerRune(c) {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}

func toLowerRune(r rune) rune {
	if r >= 'A' && r <= 'Z' {
		return r + 32
	}
	return r
}

// GenerateMarketListings genera gli item del mercato influenzati da LUC e TEC.
// luc: influenza la rarità degli item in vendita
// tec: influenza la qualità degli affix magici
func GenerateMarketListings(luc, tec, charLevel int, location string) *models.MarketState {
	// Rarità disponibili al mercato: sempre almeno uncommon
	rarityWeights := []int{40, 35, 18, 6, 1} // uncommon, rare, epic, legendary, (slot extra)
	rarities := []string{"uncommon", "rare", "epic", "legendary", "legendary"}

	// LUC sposta verso rarità più alte (meno aggressivo del loot da nemici)
	lucBonus := luc - 5
	if lucBonus < 0 {
		lucBonus = 0
	}
	// Sposta max 20% totale
	shift := lucBonus / 2
	if shift > 20 {
		shift = 20
	}
	for i := 0; i < len(rarityWeights)-1 && shift > 0; i++ {
		take := rarityWeights[i] - 5
		if take <= 0 || take > shift {
			take = shift
		}
		if take <= 0 {
			continue
		}
		rarityWeights[i] -= take
		rarityWeights[len(rarityWeights)-1] += take
		shift -= take
	}

	// Genera 4-6 item (LUC alta = più item)
	numItems := 4
	if luc >= 10 {
		numItems = 5
	}
	if luc >= 18 {
		numItems = 6
	}

	_ = marketTypesForLocation(location)

	var listings []models.MarketListing
	for i := 0; i < numItems; i++ {
		rarity := weightedRarity(rarities, rarityWeights)
		item := generateItem(rarity, charLevel, tec)

		// Markup di mercato: 1.3x - 2.0x in base alla rarità
		markups := map[string]float64{
			"uncommon": 1.4, "rare": 1.6, "epic": 1.8, "legendary": 2.0,
		}
		markup := markups[rarity]
		if markup == 0 {
			markup = 1.5
		}
		price := int(float64(item.Price) * markup)

		// TEC alta: il personaggio riconosce item di qualità → sconto base 5-10%
		if tec >= 12 {
			tecDiscount := 1.0 - float64(tec-10)*0.005
			if tecDiscount < 0.85 {
				tecDiscount = 0.85
			}
			price = int(float64(price) * tecDiscount)
		}

		// Al mercato gli item sono sempre anonimi: non sai cosa stai comprando.
		item.Appraised = false
		item.Analyzed = false

		listings = append(listings, models.MarketListing{
			Item:  item,
			Price: price,
		})
	}

	return &models.MarketState{
		Listings: listings,
		Location: location,
	}
}

// NegotiatePrice: contrattazione con un venditore. Influenzata da LUC (e TEC se item magico).
// Ritorna il risultato della contrattazione.
func NegotiatePrice(listing models.MarketListing, luc, tec int) models.NegotiateResult {
	oldPrice := listing.Price

	// Probabilità di successo: base 20% + 3% per punto LUC sopra 5
	lucBonus := luc - 5
	if lucBonus < 0 {
		lucBonus = 0
	}
	successChance := 20 + lucBonus*3
	if successChance > 70 {
		successChance = 70
	}

	// TEC aiuta con item magici/tecnici
	isMagical := false
	for stat := range listing.Item.StatBonus {
		if stat == "TEC" || stat == "LUC" {
			isMagical = true
			break
		}
	}
	if isMagical {
		tecBonus := (tec - 5) / 3
		if tecBonus < 0 {
			tecBonus = 0
		}
		successChance += tecBonus
		if successChance > 80 {
			successChance = 80
		}
	}

	roll := rand.Intn(100)

	var discountPct int
	var narrative string
	success := false

	if roll < successChance {
		// Successo pieno
		success = true
		discountPct = 15 + rand.Intn(16) // 15-30%
		if luc >= 15 {
			discountPct += 5
		}
		narratives := []string{
			fmt.Sprintf("Il venditore annuisce con rispetto. «Sei un buon osservatore. %d%% di sconto, e non ne parlo con nessuno.»", discountPct),
			fmt.Sprintf("Con un sorriso, il mercante cede. «Va bene, %d%% in meno. Ma solo perché mi sei simpatico.»", discountPct),
			fmt.Sprintf("La tua sicurezza convince il venditore. Accetta uno sconto del %d%%.", discountPct),
		}
		narrative = narratives[rand.Intn(len(narratives))]
	} else if roll < successChance+20 {
		// Successo parziale
		success = true
		discountPct = 5 + rand.Intn(6) // 5-10%
		narratives := []string{
			fmt.Sprintf("Il venditore sospira. «Posso fare %d%% di meno, ma è il mio limite.»", discountPct),
			fmt.Sprintf("Dopo qualche esitazione, ottieni uno sconto del %d%%.", discountPct),
		}
		narrative = narratives[rand.Intn(len(narratives))]
	} else {
		// Fallimento
		discountPct = 0
		narratives := []string{
			"Il venditore scuote la testa. «Questo è il prezzo finale. Prendere o lasciare.»",
			"«Ho altri clienti in attesa», dice il mercante seccamente.",
			"La contrattazione non va a buon fine. Il venditore resta fermo sul prezzo.",
		}
		narrative = narratives[rand.Intn(len(narratives))]
	}

	newPrice := oldPrice
	if discountPct > 0 {
		newPrice = int(float64(oldPrice) * (1.0 - float64(discountPct)/100.0))
	}

	return models.NegotiateResult{
		Success:     success,
		NewPrice:    newPrice,
		OldPrice:    oldPrice,
		DiscountPct: discountPct,
		Narrative:   narrative,
	}
}

// AnalysisResult è il risultato di un'analisi di item (esportato).
type AnalysisResult struct {
	Stats  map[string]int
	Rarity string
}

// ApplyTecAnalysis calcola le stat percepite in base a TEC e LUC del personaggio.
// TEC è il contributore primario, LUC aggiunge un bonus del 30% del suo valore.
// TEC bassa → errori grandi. TEC alta → analisi precisa.
func ApplyTecAnalysis(item models.Item, tec, luc int) AnalysisResult {
	rarityOrder := []string{"common", "uncommon", "rare", "epic", "legendary"}

	// LUC contribuisce al 30% come TEC effettiva aggiuntiva
	effectiveTec := float64(tec) + float64(luc)*0.3

	// Calcola accuratezza: 0.0 = totalmente sbagliata, 1.0 = perfetta
	// effTEC 0-10: 20-50%, 11-30: 55-75%, 31-60: 76-90%, 61+: 91-100%
	var accuracy float64
	switch {
	case effectiveTec <= 10:
		accuracy = 0.20 + effectiveTec*0.03
	case effectiveTec <= 30:
		accuracy = 0.50 + (effectiveTec-10)*0.0125
	case effectiveTec <= 60:
		accuracy = 0.75 + (effectiveTec-30)*0.005
	default:
		extra := effectiveTec - 60
		if extra > 40 {
			extra = 40
		}
		accuracy = 0.90 + extra*0.0025
	}

	// Perceived stats: ogni valore reale moltiplicato per un fattore casuale
	// centrato su 1.0 con deviazione dipendente dall'accuracy
	perceived := map[string]int{}
	for stat, val := range item.StatBonus {
		deviation := 1.0 - accuracy
		factor := 1.0 + (rand.Float64()*2-1)*deviation*1.5
		pVal := int(float64(val) * factor)
		if pVal < 0 {
			pVal = 0
		}
		if pVal > 0 {
			perceived[stat] = pVal
		}
	}

	// Perceived rarity: con bassa accuracy può essere sbagliata di 1 tier
	realIdx := 0
	for i, r := range rarityOrder {
		if r == item.Rarity {
			realIdx = i
			break
		}
	}
	perceivedIdx := realIdx
	if accuracy < 0.7 {
		shift := rand.Intn(3) - 1 // -1, 0, +1
		perceivedIdx = realIdx + shift
		if perceivedIdx < 0 {
			perceivedIdx = 0
		}
		if perceivedIdx >= len(rarityOrder) {
			perceivedIdx = len(rarityOrder) - 1
		}
	}

	return AnalysisResult{
		Stats:  perceived,
		Rarity: rarityOrder[perceivedIdx],
	}
}
