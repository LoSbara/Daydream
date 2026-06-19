package game

import "daydream/internal/models"

// Thresholds di livello per ogni tier
var specTierLevels = map[int]int{1: 25, 2: 50, 3: 80}

// Alberi specializzazione per classe
var classSpecTrees = map[string][]models.SpecializationTier{
	"Mercenario": {
		{Tier: 1, Level: 25, Options: []models.SpecializationOption{
			{ID: "mercenario_t1_berserker", Name: "Berserker", Description: "Attacchi devastanti a scapito della difesa.", Flavor: "La furia è la tua armatura.", StatBonus: map[string]int{"STR": 2}, PassiveDesc: "Quando sotto il 30% HP, danni aumentati del 20%."},
			{ID: "mercenario_t1_guardian", Name: "Guardian", Description: "Difensore inossidabile, assorbe i colpi per i compagni.", Flavor: "Sei il muro tra il pericolo e gli alleati.", StatBonus: map[string]int{"VIT": 2}, PassiveDesc: "Riduzione danni passiva del 10%. Taunt disponibile in combattimento."},
		}},
		{Tier: 2, Level: 50, Options: []models.SpecializationOption{
			{ID: "mercenario_t2_warlord", Name: "Warlord", Description: "Comandante del campo di battaglia, potenzia le tattiche di gruppo.", Flavor: "La tua presenza ispira terrore nei nemici.", StatBonus: map[string]int{"STR": 2, "AGI": 1}, PassiveDesc: "Attacchi critici generano una carica di adrenalina. Tre cariche attivano un attacco bonus."},
			{ID: "mercenario_t2_duelist", Name: "Duelist", Description: "Maestro del combattimento uno contro uno.", Flavor: "Ogni scontro è un duello personale.", StatBonus: map[string]int{"DEX": 2, "AGI": 1}, PassiveDesc: "In combattimento contro un singolo nemico, +15% danni e +10% schivata."},
		}},
		{Tier: 3, Level: 80, Options: []models.SpecializationOption{
			{ID: "mercenario_t3_legend", Name: "Leggenda di Guerra", Description: "Il tuo nome è temuto in tutto il continente.", Flavor: "Sei diventato qualcosa di più di un guerriero.", StatBonus: map[string]int{"STR": 3, "VIT": 2}, PassiveDesc: "Passivo unico: 'Presenza schiacciante' — i nemici di basso livello fuggono o si arrendono senza combattere."},
			{ID: "mercenario_t3_champion", Name: "Campione", Description: "Combattente supremo, padrone di ogni stile.", Flavor: "Hai trasceso le scuole di combattimento.", StatBonus: map[string]int{"STR": 2, "DEX": 2, "AGI": 1}, PassiveDesc: "Ogni cinque attacchi, il prossimo è garantito critico e ignora l'armatura."},
		}},
	},
	"Scout": {
		{Tier: 1, Level: 25, Options: []models.SpecializationOption{
			{ID: "scout_t1_assassin", Name: "Assassino", Description: "Colpi precisi, veloci e letali. Elimina prima di essere visto.", Flavor: "L'ombra è la tua casa.", StatBonus: map[string]int{"DEX": 2}, PassiveDesc: "Attacchi da nascosto infliggono x2.5 danni. Puoi dissolverti nell'ombra una volta per scontro."},
			{ID: "scout_t1_ranger", Name: "Ranger", Description: "Maestro del territorio, esplorazione e attacchi a distanza.", Flavor: "Conosci ogni sentiero, ogni nascondiglio.", StatBonus: map[string]int{"AGI": 2}, PassiveDesc: "Attacchi ranged ignorano il 15% dell'armatura. Bonus esplorazione: scopri trappole e percorsi segreti automaticamente."},
		}},
		{Tier: 2, Level: 50, Options: []models.SpecializationOption{
			{ID: "scout_t2_shadow", Name: "Ombra", Description: "Infiltrazione totale. Puoi operare in qualsiasi ambiente.", Flavor: "Sei diventato invisibile anche in piena luce.", StatBonus: map[string]int{"DEX": 2, "LUC": 1}, PassiveDesc: "Stealth permanente fuori combattimento. In città puoi camuffarti come chiunque."},
			{ID: "scout_t2_tracker", Name: "Cacciatore", Description: "Segue qualsiasi preda attraverso qualsiasi terreno.", Flavor: "Nessuno sfugge alla tua caccia.", StatBonus: map[string]int{"AGI": 2, "VIT": 1}, PassiveDesc: "Nemici marcati subiscono +20% danni da tutte le fonti. Marca dura 3 turni."},
		}},
		{Tier: 3, Level: 80, Options: []models.SpecializationOption{
			{ID: "scout_t3_phantom", Name: "Spettro", Description: "Sei diventato una leggenda degli assassini.", Flavor: "Il tuo nome è sussurrato con timore.", StatBonus: map[string]int{"DEX": 3, "AGI": 2}, PassiveDesc: "Un colpo fatale per scontro che bypassa completamente la difesa. Ricarica se elimini il bersaglio."},
			{ID: "scout_t3_pathfinder", Name: "Pathfinder", Description: "Esplora l'inesplorato, apre strade per gli altri.", Flavor: "Il mondo non ha più segreti per te.", StatBonus: map[string]int{"AGI": 2, "LUC": 2, "TEC": 1}, PassiveDesc: "Accesso a dungeon e aree segrete inaccessibili agli altri. Bonus drop rari +30%."},
		}},
	},
	"Mago": {
		{Tier: 1, Level: 25, Options: []models.SpecializationOption{
			{ID: "mago_t1_elementale", Name: "Elementalista", Description: "Maestro dei quattro elementi. Distruzione su larga scala.", Flavor: "Gli elementi rispondono alla tua volontà.", StatBonus: map[string]int{"TEC": 2}, PassiveDesc: "Incantesimi elementali hanno 15% di chance di applicare un effetto secondario (brucia/gela/paralizza/acceca)."},
			{ID: "mago_t1_arcanista", Name: "Arcanista", Description: "Studia la magia pura nella sua forma più complessa.", Flavor: "Vedi il mondo come una trama di energia arcana.", StatBonus: map[string]int{"TEC": 1, "LUC": 1}, PassiveDesc: "Ogni incantesimo lanciato riduce il costo MP del prossimo del 10% (stacks fino a 30%)."},
		}},
		{Tier: 2, Level: 50, Options: []models.SpecializationOption{
			{ID: "mago_t2_evocatore", Name: "Evocatore", Description: "Chiama entità e costrutti magici a combattere al tuo fianco.", Flavor: "Non sei mai solo in battaglia.", StatBonus: map[string]int{"TEC": 2, "VIT": 1}, PassiveDesc: "Puoi mantenere un famiglio evocato che combatte autonomamente. Cresce con il personaggio."},
			{ID: "mago_t2_incantatore", Name: "Incantatore", Description: "Piega la mente e la realtà attraverso incantesimi di controllo.", Flavor: "La battaglia è già vinta prima di iniziare.", StatBonus: map[string]int{"TEC": 2, "AGI": 1}, PassiveDesc: "Incantesimi di controllo (confusione, paura, sonno) durano 50% in più."},
		}},
		{Tier: 3, Level: 80, Options: []models.SpecializationOption{
			{ID: "mago_t3_arcimago", Name: "Arcimago", Description: "Sei tra i maghi più potenti mai esistiti.", Flavor: "La magia stessa si inchina alla tua presenza.", StatBonus: map[string]int{"TEC": 4, "VIT": 1}, PassiveDesc: "Puoi lanciare un incantesimo al doppio della potenza senza costo MP, una volta per scontro."},
			{ID: "mago_t3_tessitore", Name: "Tessitore della Realtà", Description: "Manipola le leggi fisiche del mondo.", Flavor: "Non lanci incantesimi. Riscrivi la realtà.", StatBonus: map[string]int{"TEC": 3, "LUC": 2}, PassiveDesc: "Passivo unico: ogni tre turni puoi 'riscrivere' un evento — annullare un danno subito o garantire un critico."},
		}},
	},
	"Sacerdote": {
		{Tier: 1, Level: 25, Options: []models.SpecializationOption{
			{ID: "sacerdote_t1_guaritore", Name: "Guaritore", Description: "Specializzato nel ripristino vitale, guarigione avanzata.", Flavor: "La vita scorre attraverso le tue mani.", StatBonus: map[string]int{"VIT": 2}, PassiveDesc: "Guarigioni curano il 20% in più. Puoi riportare in vita un alleato una volta per dungeon."},
			{ID: "sacerdote_t1_esorcista", Name: "Esorcista", Description: "Combatte il male con la luce divina, potenziato contro non-morti.", Flavor: "Sei il giudice delle anime oscure.", StatBonus: map[string]int{"TEC": 1, "STR": 1}, PassiveDesc: "+40% danni contro non-morti e demoni. Immunità a maledizioni."},
		}},
		{Tier: 2, Level: 50, Options: []models.SpecializationOption{
			{ID: "sacerdote_t2_oracolo", Name: "Oracolo", Description: "Connesso alle forze divine, prevede e manipola gli eventi.", Flavor: "Il futuro ti sussurra i suoi segreti.", StatBonus: map[string]int{"TEC": 2, "LUC": 1}, PassiveDesc: "Una volta per scontro puoi 'prevedere' l'attacco nemico e agire per primo."},
			{ID: "sacerdote_t2_paladino", Name: "Paladino", Description: "Combattente sacro, unisce forza fisica e potere divino.", Flavor: "La fede è la tua spada.", StatBonus: map[string]int{"STR": 2, "VIT": 1}, PassiveDesc: "Attacchi fisici infliggono danno sacro bonus. Aura che riduce danni a tutti gli alleati vicini del 10%."},
		}},
		{Tier: 3, Level: 80, Options: []models.SpecializationOption{
			{ID: "sacerdote_t3_santo", Name: "Santo", Description: "La tua fede è diventata qualcosa di tangibile nel mondo.", Flavor: "Gli dei camminano con te.", StatBonus: map[string]int{"VIT": 3, "TEC": 2}, PassiveDesc: "Passivo divino: una volta al giorno puoi chiedere un 'miracolo' al GM — un intervento narrativo straordinario."},
			{ID: "sacerdote_t3_avatar", Name: "Avatar Divino", Description: "Sei il braccio mortale di una divinità.", Flavor: "Non sei più solo umano.", StatBonus: map[string]int{"TEC": 3, "STR": 2}, PassiveDesc: "Puoi trasformarti temporaneamente in forma divina per 3 turni: tutti i valori x1.5, immunità ai debuff."},
		}},
	},
	"Ingegnere": {
		{Tier: 1, Level: 25, Options: []models.SpecializationOption{
			{ID: "ingegnere_t1_meccanico", Name: "Meccanico da Campo", Description: "Costruisce e ripara meccanismi in tempo reale.", Flavor: "Ogni problema ha una soluzione meccanica.", StatBonus: map[string]int{"TEC": 1, "VIT": 1}, PassiveDesc: "Puoi costruire trappole e torrette automatiche durante l'esplorazione. Durano fino al prossimo dungeon."},
			{ID: "ingegnere_t1_alchimista", Name: "Alchimista", Description: "Crea pozioni, bombe e reagenti avanzati.", Flavor: "La chimica è la tua magia.", StatBonus: map[string]int{"TEC": 1, "LUC": 1}, PassiveDesc: "Pozioni crafted hanno doppia efficacia. Puoi creare granate elementali con materiali trovati nel dungeon."},
		}},
		{Tier: 2, Level: 50, Options: []models.SpecializationOption{
			{ID: "ingegnere_t2_cybernetics", Name: "Cybernetico", Description: "Potenzia il proprio corpo con innesti meccanici.", Flavor: "Il corpo è il primo strumento da migliorare.", StatBonus: map[string]int{"STR": 1, "AGI": 1, "VIT": 1}, PassiveDesc: "Tre moduli innesto permanenti attivi contemporaneamente (bonus scelti all'inizio di ogni dungeon)."},
			{ID: "ingegnere_t2_stratega", Name: "Stratega", Description: "Pianifica il combattimento con precisione matematica.", Flavor: "Ogni variabile è calcolata prima del primo colpo.", StatBonus: map[string]int{"TEC": 2, "AGI": 1}, PassiveDesc: "Prima di ogni scontro, puoi 'pianificare' un bonus tattico che si applica per i primi 3 turni."},
		}},
		{Tier: 3, Level: 80, Options: []models.SpecializationOption{
			{ID: "ingegnere_t3_golem", Name: "Maestro dei Costrutti", Description: "Comanda un esercito di meccanismi.", Flavor: "Non combatti. Fai combattere le tue creazioni.", StatBonus: map[string]int{"TEC": 3, "VIT": 2}, PassiveDesc: "Golem da guerra permanente al tuo fianco, che scala con il livello. Puoi sacrificarlo per un attacco devastante e ri-costruirlo dopo il dungeon."},
			{ID: "ingegnere_t3_inventore", Name: "Grande Inventore", Description: "Crea tecnologie mai viste prima.", Flavor: "Cambi il mondo con ogni invenzione.", StatBonus: map[string]int{"TEC": 4, "LUC": 1}, PassiveDesc: "Passivo unico: una volta per sessione puoi improvvisare un'invenzione contestuale — il GM descrive un gadget appropriato alla situazione."},
		}},
	},
}

func GetSpecTiersForClass(class string) []models.SpecializationTier {
	return classSpecTrees[class]
}

func GetAvailableSpecChoice(class string, level int, chosenSpecs []string) *models.SpecChoiceAvailable {
	tiers := classSpecTrees[class]
	for _, tier := range tiers {
		if level < tier.Level {
			continue
		}
		tierChosen := false
		for _, chosen := range chosenSpecs {
			for _, opt := range tier.Options {
				if opt.ID == chosen {
					tierChosen = true
					break
				}
			}
			if tierChosen {
				break
			}
		}
		if !tierChosen {
			return &models.SpecChoiceAvailable{
				Tier:    tier.Tier,
				Level:   tier.Level,
				Options: tier.Options,
			}
		}
	}
	return nil
}

func GetChosenSpecOptions(chosenSpecs []string) []models.SpecializationOption {
	var result []models.SpecializationOption
	for _, tiers := range classSpecTrees {
		for _, tier := range tiers {
			for _, opt := range tier.Options {
				for _, chosen := range chosenSpecs {
					if opt.ID == chosen {
						result = append(result, opt)
					}
				}
			}
		}
	}
	return result
}
