package game

import (
	"fmt"
	"strings"

	"daydream/internal/models"
)

// ─── helper ──────────────────────────────────────────────────────────────────

func node(id, name, desc, typ, branch string, order int, prereqs []string, spec string, mp, stm, cd int, stat map[string]int, effect string) models.SkillTreeNode {
	return models.SkillTreeNode{
		ID: id, Name: name, Description: desc, Type: typ, Branch: branch,
		BranchOrder: order, Prerequisites: prereqs, SpecRequired: spec,
		Cost: 1, MPCost: mp, STMCost: stm, Cooldown: cd,
		StatBonus: stat, EffectDesc: effect,
	}
}

func p(ids ...string) []string { return ids }
func s(kv ...interface{}) map[string]int {
	m := map[string]int{}
	for i := 0; i+1 < len(kv); i += 2 {
		m[kv[i].(string)] = kv[i+1].(int)
	}
	return m
}

// ─── ALBERI ──────────────────────────────────────────────────────────────────

var classSkillTrees = map[string]models.ClassSkillTree{

// ══════════════════════════════════════════════════════════
// MERCENARIO
// ══════════════════════════════════════════════════════════
"Mercenario": {Class: "Mercenario", Branches: []models.SkillBranch{
	{Name: "Offesa", Icon: "⚔", Skills: []models.SkillTreeNode{
		node("merc_off_1", "Colpo Potenziato", "Un attacco concentrato che infligge 150% del danno normale.", "active", "Offesa", 1, nil, "", 8, 0, 1, nil,
			"Attacco singolo a 150% danno fisico. Garantisce danno sopra la media."),
		node("merc_off_2", "Ferita Aperta", "Lascia una ferita che sanguina per 3 turni, infliggendo danno progressivo.", "active", "Offesa", 2, p("merc_off_1"), "", 12, 0, 3, nil,
			"Applica emorragia: danno progressivo per 3 turni (10/15/20% HP per turno)."),
		node("merc_off_3", "Colpo Devastante", "Attacco brutale che può stordire il bersaglio. Richiede la furia del Berserker.", "active", "Offesa", 3, p("merc_off_2"), "mercenario_t1_berserker", 20, 0, 5, nil,
			"Danno x2.5, 60% probabilità di stordire il bersaglio per 1 turno."),
		node("merc_off_4", "Esecuzione", "Colpo finale letale. Triplo danno su nemici sotto il 25% HP.", "active", "Offesa", 4, p("merc_off_3"), "", 35, 0, 8, nil,
			"Danno x3 se il bersaglio è sotto 25% HP. Danno x1.5 altrimenti."),
	}},
	{Name: "Difesa", Icon: "🛡", Skills: []models.SkillTreeNode{
		node("merc_dif_1", "Postura Difensiva", "Abbassa la guardia offensiva per ridurre i danni subiti del 30% per 2 turni.", "active", "Difesa", 1, nil, "", 0, 15, 3, nil,
			"Riduce danni in entrata del 30% per 2 turni. Incompatibile con attacchi nello stesso turno."),
		node("merc_dif_2", "Contrattacco", "Riflesso di combattimento: 20% di probabilità di rispondere automaticamente agli attacchi.", "passive", "Difesa", 2, p("merc_dif_1"), "", 0, 0, 0, nil,
			"Passivo: ogni volta che vieni colpito, 20% chance di attaccare immediatamente."),
		node("merc_dif_3", "Fortezza", "Anni di allenamento indurito il tuo corpo. +3 VIT permanente.", "passive", "Difesa", 3, p("merc_dif_2"), "", 0, 0, 0, s("VIT", 3),
			"Passivo: +3 VIT permanente. Il tuo corpo è diventato più resistente."),
		node("merc_dif_4", "Indomabile", "Una volta per scontro, quando scendi sotto il 20% HP recuperi il 30% degli HP. Solo per chi ha scelto di proteggere gli altri.", "passive", "Difesa", 4, p("merc_dif_3"), "mercenario_t1_guardian", 0, 0, 0, nil,
			"Passivo: trigger automatico a 20% HP — recupero istantaneo 30% HP max. Una volta per scontro."),
	}},
	{Name: "Mobilità", Icon: "💨", Skills: []models.SkillTreeNode{
		node("merc_mob_1", "Scatto", "Ti lanci verso il bersaglio: il primo attacco dopo lo scatto è garantito critico.", "active", "Mobilità", 1, nil, "", 0, 10, 2, nil,
			"Movimento rapido verso il bersaglio. Il prossimo attacco nello stesso turno è critico garantito."),
		node("merc_mob_2", "Disimpegno", "Esci dal corpo a corpo senza esporre la schiena. Nessun attacco di opportunità.", "active", "Mobilità", 2, p("merc_mob_1"), "", 0, 8, 2, nil,
			"Riposizionamento sicuro: esci dalla mischia senza subire attacchi di opportunità."),
		node("merc_mob_3", "Vento di Lama", "Un arco di lama che colpisce tutti i nemici in mischia.", "active", "Mobilità", 3, p("merc_mob_2"), "", 15, 15, 4, nil,
			"AoE in mischia: attacca tutti i nemici adiacenti con 80% del danno normale ciascuno."),
		node("merc_mob_4", "Riflessi da Combattimento", "I tuoi riflessi affinati riducono le probabilità che tu venga colpito. +2 AGI.", "passive", "Mobilità", 4, p("merc_mob_3"), "", 0, 0, 0, s("AGI", 2),
			"Passivo: +2 AGI, +10% schivata. Il corpo reagisce prima che la mente elabori."),
	}},
	{Name: "Maestria", Icon: "⭐", Skills: []models.SkillTreeNode{
		node("merc_mae_1", "Analisi del Nemico", "Osservi il nemico con occhio esperto e ne identifichi i punti deboli.", "active", "Maestria", 1, nil, "", 5, 0, 4, nil,
			"Il GM rivela una debolezza specifica del nemico (elemento, postura, tipo di danno efficace)."),
		node("merc_mae_2", "Arte della Guerra", "Per 4 turni i tuoi attacchi ignorano il 20% dell'armatura nemica.", "active", "Maestria", 2, p("merc_mae_1"), "", 25, 0, 7, nil,
			"Penetrazione armatura 20% per 4 turni. Si cumula con altri modificatori."),
		node("merc_mae_3", "Gran Maestro", "Decenni di battaglia condensati in muscoli e istinti. +2 STR, +1 DEX.", "passive", "Maestria", 3, p("merc_mae_2"), "", 0, 0, 0, s("STR", 2, "DEX", 1),
			"Passivo: +2 STR, +1 DEX. Sei un guerriero completo."),
		node("merc_mae_4", "Colpo del Campione", "Il tuo attacco definitivo. Una volta per dungeon, liberi tutta la tua potenza.", "active", "Maestria", 4, p("merc_mae_3"), "mercenario_t2_warlord", 50, 0, 0, nil,
			"Attacco definitivo: danno x4, ignora tutta l'armatura. Una volta per dungeon."),
	}},
}},

// ══════════════════════════════════════════════════════════
// SCOUT
// ══════════════════════════════════════════════════════════
"Scout": {Class: "Scout", Branches: []models.SkillBranch{
	{Name: "Furtività", Icon: "🌑", Skills: []models.SkillTreeNode{
		node("scout_fur_1", "Passo Silenzioso", "I tuoi movimenti non producono suono. +10% probabilità di sorpresa.", "passive", "Furtività", 1, nil, "", 0, 0, 0, nil,
			"Passivo: +10% probabilità di iniziare il combattimento in stealth. Movimenti silenziosi."),
		node("scout_fur_2", "Colpo alle Spalle", "Attacco da posizione furtiva: danno x2.5 e possibilità di stordire.", "active", "Furtività", 2, p("scout_fur_1"), "", 10, 0, 3, nil,
			"Richiede stealth o vantaggio posizionale. Danno x2.5, 40% stordimento."),
		node("scout_fur_3", "Dissolversi", "Ti fai da parte: torni in stealth in pieno combattimento per 1 turno.", "active", "Furtività", 3, p("scout_fur_2"), "scout_t1_assassin", 15, 0, 5, nil,
			"Entra in stealth per 1 turno anche durante il combattimento. Il prossimo attacco è garantito colpo alle spalle."),
		node("scout_fur_4", "Ombra Vivente", "Sei diventato parte dell'oscurità. Fuori dal combattimento sei quasi invisibile.", "passive", "Furtività", 4, p("scout_fur_3"), "", 0, 0, 0, nil,
			"Passivo: fuori combattimento sei praticamente invisibile. In combat, stealth dura 2 turni."),
	}},
	{Name: "Precisione", Icon: "🎯", Skills: []models.SkillTreeNode{
		node("scout_pre_1", "Tiro Preciso", "Un colpo mirato che ignora una parte dell'armatura del bersaglio.", "active", "Precisione", 1, nil, "", 8, 0, 1, nil,
			"Attacco ranged con 15% penetrazione armatura. Danno garantito al minimo sopra la soglia base."),
		node("scout_pre_2", "Punto Debole", "Hai occhio per le vulnerabilità. +15% danno critico.", "passive", "Precisione", 2, p("scout_pre_1"), "", 0, 0, 0, nil,
			"Passivo: +15% danno critico su tutti gli attacchi. Identifichi istintivamente dove colpire."),
		node("scout_pre_3", "Mira Letale", "Spendi un turno a mirare per scatenare un colpo devastante al prossimo attacco.", "active", "Precisione", 3, p("scout_pre_2"), "", 12, 0, 4, nil,
			"Miri per 1 turno (vulnerabile): il prossimo attacco infligge danno x3 garantito critico."),
		node("scout_pre_4", "Tiratore d'Élite", "Il tuo controllo è assoluto. Ogni attacco a distanza ha 25% probabilità di colpo critico.", "passive", "Precisione", 4, p("scout_pre_3"), "scout_t1_ranger", 0, 0, 0, s("DEX", 2),
			"Passivo: +25% critico su attacchi ranged, +2 DEX. Sei il tiratore più letale."),
	}},
	{Name: "Veleno", Icon: "☠", Skills: []models.SkillTreeNode{
		node("scout_vel_1", "Dardo Avvelenato", "Colpisci con una sostanza tossica che indebolisce il bersaglio.", "active", "Veleno", 1, nil, "", 10, 0, 2, nil,
			"Applica veleno: danno ogni turno per 4 turni, -10% alle resistenze del bersaglio."),
		node("scout_vel_2", "Veleno Potenziato", "Il tuo veleno è più concentrato. Durata +2 turni, danno +50%.", "passive", "Veleno", 2, p("scout_vel_1"), "", 0, 0, 0, nil,
			"Passivo: tutti i veleni durano 2 turni in più e infliggono 50% danno aggiuntivo."),
		node("scout_vel_3", "Gas Tossico", "Lanci un flacone che avvelena tutti i nemici in un'area.", "active", "Veleno", 3, p("scout_vel_2"), "", 20, 0, 5, nil,
			"AoE: applica veleno a tutti i nemici nell'area. Ogni nemico tossicato subisce stack separati."),
		node("scout_vel_4", "Tocco Mortale", "Il tuo veleno evolve in una neurotossina: paralisi e danno estremo.", "active", "Veleno", 4, p("scout_vel_3"), "scout_t2_shadow", 30, 0, 8, nil,
			"Veleno neurotossico: danno x2, 50% probabilità paralisi per 2 turni, non curabile con antidoti comuni."),
	}},
	{Name: "Esplorazione", Icon: "🗺", Skills: []models.SkillTreeNode{
		node("scout_esp_1", "Sensi Acuti", "Percepisci minacce e opportunità prima che siano visibili. +1 LUC.", "passive", "Esplorazione", 1, nil, "", 0, 0, 0, s("LUC", 1),
			"Passivo: +1 LUC, percepisci trappole e imboscate automaticamente."),
		node("scout_esp_2", "Trappola", "Posiziona una trappola che si attiva sul prossimo nemico che passa.", "active", "Esplorazione", 2, p("scout_esp_1"), "", 0, 12, 3, nil,
			"Piazza una trappola nell'ambiente. Il nemico che ci cade subisce danno e rallentamento (GM gestisce il posizionamento)."),
		node("scout_esp_3", "Orientamento Tattico", "La tua conoscenza del terreno ti dà vantaggio. Trovi sempre strade alternative.", "passive", "Esplorazione", 3, p("scout_esp_2"), "", 0, 0, 0, nil,
			"Passivo: il GM rivela uscite alternative, percorsi segreti e aree nascoste. +20% chance loot segreto nei dungeon."),
		node("scout_esp_4", "Istinto del Cacciatore", "Sei nel tuo elemento ovunque. Nessun ambiente ti sorprende.", "passive", "Esplorazione", 4, p("scout_esp_3"), "scout_t2_tracker", 0, 0, 0, s("AGI", 1, "LUC", 1),
			"Passivo: +1 AGI, +1 LUC. Vantaggio tattico in qualsiasi ambiente naturale o urbano."),
	}},
}},

// ══════════════════════════════════════════════════════════
// MAGO
// ══════════════════════════════════════════════════════════
"Mago": {Class: "Mago", Branches: []models.SkillBranch{
	{Name: "Distruzione", Icon: "🔥", Skills: []models.SkillTreeNode{
		node("mago_dis_1", "Proiettile Arcano", "Un dardo di pura energia magica. Veloce, preciso, efficiente.", "active", "Distruzione", 1, nil, "", 10, 0, 0, nil,
			"Attacco magico base. Colpisce sempre, ignora armatura fisica, 120% danno TEC-scalato."),
		node("mago_dis_2", "Esplosione Elementale", "Un'esplosione focalizzata che colpisce un'area ristretta.", "active", "Distruzione", 2, p("mago_dis_1"), "", 20, 0, 3, nil,
			"Danno AoE in area piccola. 150% danno magico, applica elemento (fuoco/ghiaccio/fulmine) scelto al cast."),
		node("mago_dis_3", "Tempesta Arcana", "Scateni una tempesta di energia che dura 3 turni.", "active", "Distruzione", 3, p("mago_dis_2"), "mago_t1_elementale", 40, 0, 7, nil,
			"Canale per 3 turni: ogni turno colpisce tutti i nemici. Danno crescente (100/150/200%). Interrompibile."),
		node("mago_dis_4", "Apocalisse Arcana", "La tua magia raggiunge il picco assoluto. Un'esplosione che non lascia scampo.", "active", "Distruzione", 4, p("mago_dis_3"), "", 80, 0, 12, nil,
			"Danno x5 su area larga, ignora resistenze magiche. Il caster subisce 20% del danno inflitto per il recoil."),
	}},
	{Name: "Controllo", Icon: "❄", Skills: []models.SkillTreeNode{
		node("mago_ctr_1", "Rallentamento", "Rallenti i movimenti del bersaglio, riducendo la sua velocità d'attacco.", "active", "Controllo", 1, nil, "", 8, 0, 2, nil,
			"Applica rallentamento: -30% velocità attacco, -20% schivata per 2 turni."),
		node("mago_ctr_2", "Intralcio", "Lacci di energia arcana bloccano il bersaglio per 1 turno.", "active", "Controllo", 2, p("mago_ctr_1"), "", 15, 0, 4, nil,
			"Immobilizza il bersaglio per 1 turno completo. Può essere resistito da nemici di alto livello."),
		node("mago_ctr_3", "Prigione Arcana", "Una gabbia di energia sigilla il bersaglio per 2 turni.", "active", "Controllo", 3, p("mago_ctr_2"), "", 30, 0, 6, nil,
			"Imprigiona il bersaglio per 2 turni: non può muoversi né attaccare, ma anche gli alleati non possono colpirlo."),
		node("mago_ctr_4", "Mente Dominata", "La tua magia penetra la mente nemica. Il bersaglio combatte per te per 2 turni.", "active", "Controllo", 4, p("mago_ctr_3"), "mago_t2_incantatore", 50, 0, 10, nil,
			"Dominazione mentale per 2 turni. Il nemico attacca i propri alleati. Non funziona su boss."),
	}},
	{Name: "Protezione", Icon: "🔵", Skills: []models.SkillTreeNode{
		node("mago_pro_1", "Scudo Magico", "Un campo di forza assorbe i danni. +15 MP max.", "passive", "Protezione", 1, nil, "", 0, 0, 0, s("TEC", 1),
			"Passivo: +1 TEC. In combattimento, 15% dei danni subiti viene assorbito dal pool MP invece che HP."),
		node("mago_pro_2", "Assorbimento Arcano", "Assorbi un attacco magico e converti parte dell'energia in MP.", "active", "Protezione", 2, p("mago_pro_1"), "", 0, 0, 3, nil,
			"Reazione: blocca il prossimo attacco magico e recupera MP pari al 30% del danno bloccato."),
		node("mago_pro_3", "Bolla di Forza", "Crei una sfera di forza intorno a te o un alleato che assorbe 3 colpi.", "active", "Protezione", 3, p("mago_pro_2"), "", 25, 0, 6, nil,
			"Scudo che assorbe i prossimi 3 attacchi (fisici o magici). Dura fino alla fine dello scontro o fino a esaurimento."),
		node("mago_pro_4", "Annullamento Magico", "Cancelli completamente un incantesimo nemico. Una volta per scontro.", "active", "Protezione", 4, p("mago_pro_3"), "", 20, 0, 8, nil,
			"Reazione: annulla completamente un incantesimo nemico. Una volta per scontro."),
	}},
	{Name: "Canalizzazione", Icon: "✨", Skills: []models.SkillTreeNode{
		node("mago_can_1", "Visione Arcana", "Percepisci le energie magiche nell'ambiente. Rivela incantesimi attivi e debolezze.", "active", "Canalizzazione", 1, nil, "", 5, 0, 4, nil,
			"Il GM rivela: resistenze/debolezze magiche del nemico, incantesimi attivi, oggetti magici nascosti nell'area."),
		node("mago_can_2", "Canalizzazione Pura", "Concentri il flusso di mana. +2 TEC permanente.", "passive", "Canalizzazione", 2, p("mago_can_1"), "", 0, 0, 0, s("TEC", 2),
			"Passivo: +2 TEC. La tua connessione al mana è più profonda."),
		node("mago_can_3", "Sovraccarico", "Sacrifichi HP per potenziare il prossimo incantesimo: danno x2.", "active", "Canalizzazione", 3, p("mago_can_2"), "", 0, 0, 5, nil,
			"Spendi 15% HP max: il prossimo incantesimo lancia a doppia potenza. Stacks con altri modificatori."),
		node("mago_can_4", "Trasformazione Arcana", "Lasci che il mana fluisca liberamente: per 3 turni tutti i tuoi incantesimi non costano MP.", "active", "Canalizzazione", 4, p("mago_can_3"), "mago_t2_evocatore", 0, 0, 10, nil,
			"Per 3 turni: costo MP = 0 su tutti gli incantesimi. Alla fine, esaurisci 40% HP per il recoil energetico."),
	}},
}},

// ══════════════════════════════════════════════════════════
// SACERDOTE
// ══════════════════════════════════════════════════════════
"Sacerdote": {Class: "Sacerdote", Branches: []models.SkillBranch{
	{Name: "Guarigione", Icon: "💚", Skills: []models.SkillTreeNode{
		node("sac_gua_1", "Cura", "Un incantesimo di guarigione base che ripristina HP.", "active", "Guarigione", 1, nil, "", 15, 0, 0, nil,
			"Ripristina 25% HP max a sé stesso o un alleato."),
		node("sac_gua_2", "Cura Avanzata", "Una guarigione più potente che ripristina HP e rimuove un debuff.", "active", "Guarigione", 2, p("sac_gua_1"), "", 25, 0, 2, nil,
			"Ripristina 45% HP max. Rimuove automaticamente 1 debuff dal bersaglio."),
		node("sac_gua_3", "Rigenerazione", "Applica un effetto rigenerativo che cura nel tempo per 4 turni.", "active", "Guarigione", 3, p("sac_gua_2"), "", 20, 0, 4, nil,
			"Bersaglio recupera 10% HP max per 4 turni consecutivi."),
		node("sac_gua_4", "Resurrezione", "Il potere divino sfida la morte. Riporti in vita un personaggio sconfitto.", "active", "Guarigione", 4, p("sac_gua_3"), "sacerdote_t1_guaritore", 60, 0, 0, nil,
			"Riporta in vita un alleato sconfitto con 30% HP. Una volta per dungeon."),
	}},
	{Name: "Protezione", Icon: "✝", Skills: []models.SkillTreeNode{
		node("sac_pro_1", "Benedizione", "Proteggi un alleato con la grazia divina: +15% a tutte le resistenze per 3 turni.", "active", "Protezione", 1, nil, "", 12, 0, 3, nil,
			"Benedizione su alleato: +15% resistenze fisiche e magiche per 3 turni."),
		node("sac_pro_2", "Aura Sacra", "Emani un'aura costante che protegge passivamente te e gli alleati vicini.", "passive", "Protezione", 2, p("sac_pro_1"), "", 0, 0, 0, nil,
			"Passivo: aura che riduce i danni subiti da te del 10% permanentemente."),
		node("sac_pro_3", "Barriera Divina", "Scudi un alleato con una barriera che assorbe i danni per 2 turni.", "active", "Protezione", 3, p("sac_pro_2"), "", 30, 0, 5, nil,
			"Barriera su alleato: assorbe fino al 50% HP max in danni per 2 turni."),
		node("sac_pro_4", "Consacrazione", "Consacri il terreno: nemici non-morti/demoni subiscono penalità, alleati recuperano HP.", "active", "Protezione", 4, p("sac_pro_3"), "sacerdote_t2_paladino", 40, 0, 8, nil,
			"Consacra l'area per 3 turni: alleati recuperano 8% HP per turno, non-morti/demoni subiscono -20% a tutte le stat."),
	}},
	{Name: "Offesa Sacra", Icon: "⚡", Skills: []models.SkillTreeNode{
		node("sac_off_1", "Colpo Sacro", "Incanalate energia divina nel colpo fisico. Danno sacro aggiuntivo.", "active", "Offesa Sacra", 1, nil, "", 10, 0, 1, nil,
			"Attacco fisico + danno sacro bonus (30% TEC). Efficace contro non-morti e demoni (+50%)."),
		node("sac_off_2", "Giudizio", "Giudicate il nemico: lo marchiate e tutti i danni successivi aumentano.", "active", "Offesa Sacra", 2, p("sac_off_1"), "", 18, 0, 4, nil,
			"Segna il bersaglio: per 3 turni subisce +20% danno da tutte le fonti."),
		node("sac_off_3", "Purificazione", "Un raggio di luce purificante che brucia il male e rimuove buff nemici.", "active", "Offesa Sacra", 3, p("sac_off_2"), "sacerdote_t1_esorcista", 25, 0, 5, nil,
			"Danno sacro massiccio (x2 TEC), rimuove tutti i buff attivi del bersaglio."),
		node("sac_off_4", "Vendetta Divina", "L'ira del divino si abbatte sul nemico. Più HP hai perso, più è potente.", "active", "Offesa Sacra", 4, p("sac_off_3"), "", 40, 0, 9, nil,
			"Danno sacro scalato: base x2, +0.1x per ogni 5% HP mancanti (max x4 a 10% HP)."),
	}},
	{Name: "Grazia Divina", Icon: "🌟", Skills: []models.SkillTreeNode{
		node("sac_gra_1", "Ispirazione", "Le tue parole infondono coraggio: l'alleato target ottiene +20% a danni e velocità per 2 turni.", "active", "Grazia Divina", 1, nil, "", 15, 0, 4, nil,
			"Alleato: +20% danno e velocità per 2 turni."),
		node("sac_gra_2", "Rimozione Maledizione", "Rimuovi maledizioni, veleni e stati negativi con la luce divina.", "active", "Grazia Divina", 2, p("sac_gra_1"), "", 10, 0, 2, nil,
			"Rimuove tutti gli stati negativi (veleno, maledizione, paura, paralisi) da sé stesso o alleato."),
		node("sac_gra_3", "Potenziamento Divino", "La grazia divina si radica in te. +2 VIT, +1 TEC permanente.", "passive", "Grazia Divina", 3, p("sac_gra_2"), "", 0, 0, 0, s("VIT", 2, "TEC", 1),
			"Passivo: +2 VIT, +1 TEC. La tua connessione col divino si approfondisce."),
		node("sac_gra_4", "Grazia Celestiale", "Per 3 turni, ogni incantesimo che lanci ha effetto doppio e costo dimezzato.", "active", "Grazia Divina", 4, p("sac_gra_3"), "sacerdote_t2_oracolo", 0, 0, 12, nil,
			"Stato di grazia per 3 turni: tutti gli incantesimi a costo MP/2 e doppia potenza. Straordinariamente raro da attivare."),
	}},
}},

// ══════════════════════════════════════════════════════════
// INGEGNERE
// ══════════════════════════════════════════════════════════
"Ingegnere": {Class: "Ingegnere", Branches: []models.SkillBranch{
	{Name: "Costrutti", Icon: "🔧", Skills: []models.SkillTreeNode{
		node("ing_cos_1", "Torretta Basica", "Costruisci una torretta automatica che attacca ogni turno per 4 turni.", "active", "Costrutti", 1, nil, "", 0, 15, 3, nil,
			"Deploya una torretta: attacca ogni turno con danno = 60% del tuo danno fisico. Dura 4 turni o finché distrutta."),
		node("ing_cos_2", "Torretta Avanzata", "Versione potenziata: dual-shot e più duratura.", "active", "Costrutti", 2, p("ing_cos_1"), "", 0, 20, 5, nil,
			"Torretta avanzata: doppio sparo per turno, dura 5 turni, 80% del tuo danno fisico per colpo."),
		node("ing_cos_3", "Golem da Campo", "Costruisci un golem che combatte al tuo fianco per l'intera battaglia.", "active", "Costrutti", 3, p("ing_cos_2"), "ingegnere_t1_meccanico", 0, 30, 0, nil,
			"Evoca un golem per tutta la battaglia: 50% HP max, 80% ATK. Una volta per dungeon."),
		node("ing_cos_4", "Grande Costrutto", "Il tuo capolavoro meccanico: un costrutto gigante che domina il campo.", "active", "Costrutti", 4, p("ing_cos_3"), "", 0, 50, 0, nil,
			"Evoca il Grande Costrutto: 150% HP max, 130% ATK, azione speciale ogni 3 turni. Una volta per sessione."),
	}},
	{Name: "Esplosivi", Icon: "💥", Skills: []models.SkillTreeNode{
		node("ing_esp_1", "Granada Fumogena", "Oscura il campo di battaglia, riducendo la precisione nemica.", "active", "Esplosivi", 1, nil, "", 0, 8, 2, nil,
			"Area fumogena per 2 turni: -25% precisione di tutti i nemici nell'area."),
		node("ing_esp_2", "Granada a Frammentazione", "Esplosivo che infligge danni a tutti i nemici in un'area.", "active", "Esplosivi", 2, p("ing_esp_1"), "", 0, 15, 3, nil,
			"AoE danno fisico: 120% danno a tutti i nemici nell'area. Può distruggere oggetti dell'ambiente."),
		node("ing_esp_3", "Bomba Arcana", "Combini chimica e magia per un esplosivo devastante.", "active", "Esplosivi", 3, p("ing_esp_2"), "", 15, 15, 5, nil,
			"AoE danno misto (fisico + magico): 150% danno, applica debolezza magica per 2 turni."),
		node("ing_esp_4", "Detonazione di Massa", "Piazzi cariche in tutto il campo e le fai esplodere simultaneamente.", "active", "Esplosivi", 4, p("ing_esp_3"), "ingegnere_t2_cybernetics", 20, 40, 10, nil,
			"Piazzi 5 cariche (posizioni scelte con GM) e le fai esplodere tutte: danno devastante x2 per ogni carica su stesso bersaglio."),
	}},
	{Name: "Potenziamenti", Icon: "⚙", Skills: []models.SkillTreeNode{
		node("ing_pot_1", "Analisi Tecnica", "Analizzi il nemico con occhio tecnico. +1 TEC permanente.", "passive", "Potenziamenti", 1, nil, "", 0, 0, 0, s("TEC", 1),
			"Passivo: +1 TEC. Ogni nemico che analizzi (primo turno di scontro) rivela la sua vulnerabilità tecnica."),
		node("ing_pot_2", "Modifica Armatura", "Modifica la tua armatura sul campo. +2 VIT permanente.", "passive", "Potenziamenti", 2, p("ing_pot_1"), "", 0, 0, 0, s("VIT", 2),
			"Passivo: +2 VIT. Puoi modificare armatura alleata durante il riposo per +1 VIT temporaneo."),
		node("ing_pot_3", "Overclock", "Sovraccarichi i tuoi sistemi: per 3 turni velocità e danno aumentano del 30%.", "active", "Potenziamenti", 3, p("ing_pot_2"), "", 0, 20, 6, nil,
			"Per 3 turni: +30% velocità attacco, +30% danno. Al termine: stun di 1 turno per surriscaldamento."),
		node("ing_pot_4", "Sistema Integrato", "Integri tutti i tuoi strumenti in un sistema coerente. Ogni costruzione potenzia le successive.", "passive", "Potenziamenti", 4, p("ing_pot_3"), "ingegnere_t2_stratega", 0, 0, 0, s("TEC", 1, "VIT", 1),
			"Passivo: ogni torretta/golem attivo aumenta il tuo danno del 10% (stackabile). +1 TEC, +1 VIT."),
	}},
	{Name: "Alchimia", Icon: "🧪", Skills: []models.SkillTreeNode{
		node("ing_alc_1", "Pozione d'Emergenza", "Sintetizzi al volo una pozione curativa quando ne hai più bisogno.", "active", "Alchimia", 1, nil, "", 0, 10, 3, nil,
			"Recupera 30% HP istantaneamente. Puoi usarla anche in combattimento senza perdere il turno."),
		node("ing_alc_2", "Catalizzatore", "Un reagente che amplifica il prossimo effetto chimico o magico del 50%.", "active", "Alchimia", 2, p("ing_alc_1"), "", 10, 10, 4, nil,
			"Il prossimo oggetto consumabile o incantesimo usato entro 2 turni ha effetto +50%."),
		node("ing_alc_3", "Acido Corrosivo", "Lanci acido che scioglie l'armatura e lascia il nemico vulnerabile.", "active", "Alchimia", 3, p("ing_alc_2"), "ingegnere_t1_alchimista", 15, 0, 4, nil,
			"Danno acido nel tempo + riduzione armatura del bersaglio del 30% per 3 turni."),
		node("ing_alc_4", "Grande Opera", "Il tuo capolavoro alchemico: un elisir che trasforma temporaneamente le tue capacità.", "active", "Alchimia", 4, p("ing_alc_3"), "", 0, 0, 0, nil,
			"Bevi l'elisir: per 5 turni tutte le tue stat raddoppiano. Preparazione richiede riposo (una volta per sessione)."),
	}},
}},

} // fine classSkillTrees

// ─── Funzioni pubbliche ───────────────────────────────────────────────────────

// GetClassSkillTree restituisce l'albero di una classe con i campi Unlocked e Available calcolati.
func GetClassSkillTree(class string, unlockedIDs []string, chosenSpecs []string) *models.ClassSkillTree {
	// Normalizza la classe per gestire variazioni di maiuscole/minuscole dal DB
	if class != "" {
		normalized := strings.ToUpper(class[:1]) + strings.ToLower(class[1:])
		if _, found := classSkillTrees[normalized]; found {
			class = normalized
		}
	}
	base, ok := classSkillTrees[class]
	if !ok {
		return nil
	}

	unlockedSet := map[string]bool{}
	for _, id := range unlockedIDs {
		unlockedSet[id] = true
	}

	// Costi per tier: posizione 0→3 costo 3/7/15/28
	tierCosts := []int{3, 7, 15, 28}

	result := models.ClassSkillTree{Class: base.Class}
	for _, branch := range base.Branches {
		var skills []models.SkillTreeNode
		for ni, sk := range branch.Skills {
			sk.Cost = tierCosts[minInt(ni, 3)]
			sk.Unlocked = unlockedSet[sk.ID]
			sk.Available = canUnlockNode(sk, unlockedSet, chosenSpecs)
			skills = append(skills, sk)
		}
		result.Branches = append(result.Branches, models.SkillBranch{
			Name: branch.Name, Icon: branch.Icon, Skills: skills,
		})
	}
	return &result
}

// canUnlockNode controlla se tutti i prerequisiti e la spec richiesta sono soddisfatti.
func canUnlockNode(sk models.SkillTreeNode, unlocked map[string]bool, chosenSpecs []string) bool {
	if unlocked[sk.ID] {
		return false // già sbloccata
	}
	for _, prereq := range sk.Prerequisites {
		if !unlocked[prereq] {
			return false
		}
	}
	if sk.SpecRequired != "" {
		found := false
		for _, cs := range chosenSpecs {
			if cs == sk.SpecRequired {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

// GetUnlockedSkillsForPrompt formatta le skill sbloccate per l'iniezione nel prompt GM.
func GetUnlockedSkillsForPrompt(class string, unlockedIDs []string) string {
	if len(unlockedIDs) == 0 {
		return ""
	}
	// Normalizza la classe
	if class != "" {
		normalized := strings.ToUpper(class[:1]) + strings.ToLower(class[1:])
		if _, found := classSkillTrees[normalized]; found {
			class = normalized
		}
	}
	tree, ok := classSkillTrees[class]
	if !ok {
		return ""
	}
	unlockedSet := map[string]bool{}
	for _, id := range unlockedIDs {
		unlockedSet[id] = true
	}

	var sb strings.Builder
	sb.WriteString("## SKILL SBLOCCATE (Albero Abilità)\n")
	for _, branch := range tree.Branches {
		first := true
		for _, sk := range branch.Skills {
			if !unlockedSet[sk.ID] {
				continue
			}
			if first {
				sb.WriteString(fmt.Sprintf("**%s %s:**\n", branch.Icon, branch.Name))
				first = false
			}
			if sk.Type == "active" {
				cost := ""
				if sk.MPCost > 0 {
					cost += fmt.Sprintf("MP:%d ", sk.MPCost)
				}
				if sk.STMCost > 0 {
					cost += fmt.Sprintf("STM:%d ", sk.STMCost)
				}
				if sk.Cooldown > 0 {
					cost += fmt.Sprintf("CD:%dt", sk.Cooldown)
				}
				sb.WriteString(fmt.Sprintf("- **%s** [Attivo | %s]: %s\n", sk.Name, strings.TrimSpace(cost), sk.EffectDesc))
			} else {
				sb.WriteString(fmt.Sprintf("- **%s** [Passivo]: %s\n", sk.Name, sk.EffectDesc))
			}
		}
	}
	sb.WriteString("\nQuando il giocatore usa una skill per nome, risolvila secondo l'EffectDesc sopra.\n")
	return sb.String()
}
