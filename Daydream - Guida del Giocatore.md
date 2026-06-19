# Daydream — Guida del Giocatore

> Hai appena creato il tuo personaggio. Questa guida ti spiega come funziona il gioco — dal primo messaggio al livello 100.

---

## Indice

1. [Il Game Master](#1-il-game-master)
2. [Il tuo personaggio](#2-il-tuo-personaggio)
3. [L'inventario e l'equipaggiamento](#3-inventario)
4. [Il tempo](#4-il-tempo)
5. [Il mercato](#5-il-mercato)
6. [Le quest](#6-le-quest)
7. [Skill e albero](#7-skill-e-albero)
8. [Le specializzazioni](#8-le-specializzazioni)
9. [Il mondo che ricorda](#9-il-mondo-che-ricorda)
10. [I pannelli dell'interfaccia](#10-i-pannelli)

---

## 1. Il Game Master

Daydream è guidato da un Game Master artificiale. Non ci sono comandi, menu di scelta o bottoni narrativi: **scrivi cosa fa il tuo personaggio**, e il GM risponde in italiano, in modo naturale, interpretando l'intenzione e narrando le conseguenze.

**Esempi di input validi:**

```
Entro nella taverna e cerco qualcuno che abbia bisogno di un lavoro.
Attacco il goblin con la mia spada, mirando alla coscia.
Provo a convincere il mercante che l'oggetto vale la metà di quello che chiede.
Mi fermo ad ascoltare — sento qualcosa di strano in questo corridoio.
Cerco un posto tranquillo dove riposare per la notte.
```

Il GM è coerente con la storia. Se hai stretto un patto con un NPC, lo ricorda. Se hai distrutto un accampamento, la zona è cambiata. Le tue scelte hanno peso perché il sistema le registra e le reinietta nel contesto del GM ad ogni turno.

---

## 2. Il tuo personaggio

### Statistiche

Il tuo personaggio è definito da **6 statistiche**, una scala di livello da 1 a 100 e tre risorse vitali.

| Stat | Nome completo | A cosa serve |
|------|---------------|-------------|
| STR  | Forza         | Danno fisico in mischia, capacità di trasporto |
| DEX  | Destrezza     | Precisione attacchi, danno ranged, critico |
| AGI  | Agilità       | Velocità, schivata, iniziativa in combattimento |
| TEC  | Tecnica       | Potere magico, identificazione oggetti, negoziazione specializzata |
| VIT  | Vitalità      | HP massimi, resistenza ai danni |
| LUC  | Fortuna       | Rarità del loot, drop rate, successo in contrattazioni e analisi |

**Valori iniziali:** ogni stat parte da **5**. Alla creazione hai distribuito **15 punti extra** — più il bonus di classe (+3 a 2 stat specifiche). A ogni level-up ricevi **5 punti stat** aggiuntivi, assegnabili liberamente quando vuoi.

### Risorse

| Risorsa | Nome | Descrizione |
|---------|------|-------------|
| HP  | Punti vita   | Se scende a 0 muori. Respawn a Crysta con penalità in oro. Si rigenera completamente dormendo. |
| MP  | Energia magica | Consumata dalle skill magiche e attive. TEC alta la espande; VIT bassa la tiene bassa. |
| STM | Stamina      | Consumata dalle skill fisiche e da azioni impegnative. Si esaurisce più in fretta senza riposo. |

### Livello

Guadagni esperienza da combattimenti, quest e eventi narrativi. Ogni level-up porta:
- **+5 punti stat**
- **+3 punti skill**
- **HP/MP/STM ripristinati al massimo**

Il livello massimo è **100**.

---

## 3. Inventario

### La borsa e gli slot di equipaggiamento

Tutto quello che trovi, compri o ricevi finisce nella borsa. Puoi equipaggiare oggetti in questi slot:

`weapon` · `offhand` · `head` · `chest` · `legs` · `boots` · `accessory_1` · `accessory_2`

Gli item equipaggiati danno bonus alle stat che si sommano al tuo totale. Un'arma +4 STR aggiunge quattro punti di forza finché la porti.

### Oggetti non identificati

La maggior parte degli oggetti ottenuti da nemici o acquistati al mercato sono **non identificati** (`appraised: false`). Puoi vedere che esiste qualcosa, ma le sue statistiche reali sono oscurate.

Per identificare un oggetto in borsa, usa il pulsante **Identifica** nel pannello inventario. Il costo in oro dipende da livello e rarità dell'oggetto. La qualità dell'analisi dipende da:

- **TEC** — contributo primario
- **LUC** — contribuisce al 30% come bonus effettivo (`effective_tec = TEC + LUC × 0.3`)

**Analisi imperfetta:** con TEC bassa, le statistiche mostrate potrebbero essere sbagliate di una percentuale significativa. Potresti valutare un'arma 200 oro e scoprire, equipaggiandola, che ne vale 80. O il contrario.

| effective_tec | Accuratezza delle stat percepite |
|---------------|----------------------------------|
| 0–10          | 20–50% (errori grandi, rarità può shiftare ±1 tier) |
| 11–30         | 50–75% |
| 31–60         | 75–90% |
| 61+           | 90–100% |

---

## 4. Il tempo

Daydream ha un **orologio interno** visibile nella barra in alto (`☀ Giorno 3  14:30` / `🌙 Giorno 3  23:15`). Il tempo avanza automaticamente ad ogni azione — non devi tenerne traccia tu.

### Tempo per azione

| Azione | Tempo |
|--------|-------|
| Conversazione con NPC | ~25 minuti |
| Combattimento | 20 min + 15 per nemico sconfitto |
| Esplorazione dungeon | ~30 min + 20 per stanza |
| Viaggio locale (stessa area) | ~20 minuti |
| Viaggio verso altra città | ~4 ore |
| Viaggio lungo | ~1 giorno |
| Riposo completo | ~7 ore |

Il GM classifica ogni azione (`action_category`); il backend calcola i minuti esatti senza lasciare la scelta al modello.

### Giorno e notte

Mercanti e NPC sono tipicamente disponibili dalle **08:00 alle 20:00**. Di notte le botteghe chiudono. Certi eventi, incontri e nemici esistono solo nelle ore notturne.

### Sleep deprivation

Se non dormi, il tuo personaggio accumula fatica. I debuff si applicano automaticamente e persistono finché non riposi.

| Ore sveglio | Stato | Effetto |
|-------------|-------|---------|
| < 20h       | Riposato | Nessuno |
| 20–28h      | Stanco | Tutte le stat −2 |
| 28–36h      | Esausto | Stat −5, HP max −15%, MP max −30% |
| 36–48h      | Al limite | Stat −10, HP max −30% |
| 48h+        | Collasso imminente | Stat −10, HP max −30%, flag critico al GM |

Per riposare, dì al GM di fermarti a dormire o di cercare una locanda. Il clock avanza di ~7 ore e tutti i debuff da fatica svaniscono.

---

## 5. Il mercato

In ogni location trovi un mercato con **4–6 oggetti in vendita** (di più con LUC alta). Gli item del mercato sono **sempre anonimi** (`analyzed: false`) — vedi il prezzo, ma non cosa stai comprando.

### Analizzare prima di comprare

Premi **🔍 Analizza** per esaminare un oggetto prima dell'acquisto. L'analisi usa TEC+LUC e può dare stat percepite imprecise (vedi tabella accuratezza sopra). Le stat percepite appaiono con il prefisso `~` in giallo.

### Contrattare

Puoi tentare un'offerta su qualsiasi oggetto. Il successo dipende principalmente da **LUC**, con un bonus se stai trattando item magici e hai TEC alta.

| Esito | Effetto |
|-------|---------|
| Successo pieno | Sconto 15–30% |
| Successo parziale | Sconto 5–10% |
| Fallimento | Prezzo invariato — o NPC indispettito |

La contrattazione è possibile **una sola volta per item** per listino.

---

## 6. Le quest

Le quest arrivano dal GM — da NPC incontrati, eventi narrativi o conseguenze delle tue azioni. Ogni quest ha una **difficoltà** (da ★ a ★★★★★) e una **urgenza** che definisce quanto tempo hai.

### Urgenza e scadenza

| Urgency | Tempo disponibile | Contesto tipico |
|---------|-------------------|-----------------|
| low | ~50+ ore | Missioni politiche, eventi lenti |
| medium | ~28 ore | Situazioni che peggiorano ma danno tempo |
| high | ~14 ore | Agisci presto |
| critical | ~6 ore | Oggi. Ogni ora conta. |

La scadenza è **assoluta in tempo di gioco** — dormire, viaggiare e combattere consumano le ore disponibili.

### Le quest non aspettano

Se ignori una quest, il mondo va avanti. Ogni quest ha degli **stage di escalation** che si attivano automaticamente al consumo di specifiche percentuali del tempo. Esempio per una quest `high` "Elimina il Villaggio Goblin":

- **33%**: i goblin si espandono — il GM lo narra, un world flag viene emesso
- **66%**: iniziano a razziare i villaggi vicini
- **90%**: ultima chance — le conseguenze sono imminenti
- **100%**: la quest scade, lo stato diventa `expired`, le conseguenze sono permanenti

### Ricompense automatiche

Al completamento di una quest (il GM emette il completamento), le ricompense vengono applicate **automaticamente**:
- Oro accreditato immediatamente
- Esperienza processata (può scatenare un level-up)
- Item inseriti in borsa

### Il pannello Quest

Mostra per ogni missione attiva:
- Stelle difficoltà e colore urgency
- Countdown `⏱ 1g 6h` in tempo di gioco reale
- Warning con descrizione dello stage corrente di escalation
- Ricompense attese come badge

---

## 7. Skill e albero

Ogni classe ha un **albero di 16 skill** organizzato in 4 rami da 4 nodi ciascuno. Le skill si sbloccano spendendo punti skill — guadagni **3 punti ogni livello**.

### Costi per tier

| Tier | Costo | Note |
|------|-------|------|
| T1 — Base     | 3 pt | Sempre disponibili |
| T2 — Avanzato | 7 pt | Richiede T1 del ramo |
| T3 — Esperto  | 15 pt | Richiede T2 del ramo |
| T4 — Capstone | 28 pt | Poteri definitivi della classe |

A livello 100 avrai **~297 punti totali** — abbastanza per padroneggiare 5–6 rami su 20. Non puoi avere tutto. Scegliere dove investire è parte del gioco.

### Skill uniche (GMCustomSkill)

Il GM può concederti abilità che non esistono nell'albero standard — legate alla tua storia, agli NPC incontrati o agli eventi vissuti. Appaiono nella sezione **Abilità Uniche** del pannello albero.

Le skill uniche hanno **5 livelli** di potenziamento, upgrade spendendo punti skill:

| Passaggio | Costo |
|-----------|-------|
| Lv 1 → 2 | 15 pt |
| Lv 2 → 3 | 30 pt |
| Lv 3 → 4 | 50 pt |
| Lv 4 → 5 | 80 pt (massimo) |

Portare una skill unica al massimo costa **175 punti** — più della metà del budget totale a Lv 100. È una scelta di build valida, ma esclusiva.

---

## 8. Le specializzazioni

A tre momenti chiave del percorso, il gioco ti chiede chi vuoi diventare. Una finestra modale blocca la sessione e presenta due opzioni per il tuo tier. La scelta è **definitiva**.

| Livello | Tier | Cosa succede |
|---------|------|-------------|
| 25 | Primo | Hai imparato le basi. Scegli la tua direzione. |
| 50 | Secondo | Punto di svolta — la scelta cambia come funzioni in combattimento e narrazione. |
| 80 | Terzo | Ascensione. Rarissima — pochi arrivano qui. |

Ogni scelta dà un **bonus passivo permanente** e modifica come il GM interpreta le tue azioni.

---

## 9. Il mondo che ricorda

Daydream tiene traccia degli eventi in modo persistente attraverso i **world flags** — pezzi di stato narrativo scritti dal GM in tempo reale.

I flag sono organizzati per scope:

| Scope | Cosa registra |
|-------|---------------|
| world | Eventi globali — guerre, cataclismi, ere |
| kingdom | Stato dei regni e delle fazioni principali |
| city | Stato di una singola città o zona |
| npc | Relazioni e stato di NPC specifici |
| faction | Posizione e potere delle fazioni |
| dungeon | Stato di esplorazione e boss già sconfitti |
| player | Flag specifici del tuo personaggio |

Il GM legge i flag rilevanti ad ogni turno e li usa per costruire la risposta. Se hai salvato un villaggio, quella città ti riconosce. Se hai tradito un alleato, le sue connessioni lo sanno. Se una quest è scaduta, le conseguenze sono state emesse come flag permanenti nel mondo.

Il pannello **Mondo** mostra tutti i flag attivi, aggiornati in tempo reale.

---

## 10. I pannelli dell'interfaccia

L'interfaccia è composta da **pannelli mobili e ridimensionabili**. Trascina le intestazioni per riposizionarli, usa le maniglie per ridimensionarli.

| Pannello | Funzione |
|----------|----------|
| 💬 Chat | La conversazione con il GM |
| 📊 Stats | Statistiche, risorse, status effect attivi |
| 🎒 Inventario | Borsa ed equipaggiamento con slot |
| ⚡ Skill | Le skill attive nel loadout |
| 📜 Quest | Missioni attive, countdown, cronologia |
| 🌍 Mondo | World flags per scope |
| 🗺 Dungeon | Mappa generata durante l'esplorazione |
| 🏪 Mercato | Item in vendita nella location corrente |
| 🌳 Albero Skill | Albero completo e abilità uniche |

I preset di layout — **Default**, **Combat**, **Exploration** — riorganizzano i pannelli per adattarsi alla situazione. Puoi salvare la tua configurazione personalizzata.

---

*Daydream — Guida del Giocatore · Versione 2026-06-19*
