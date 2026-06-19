package models

type MarketListing struct {
	Item      Item   `json:"item"`
	Price     int    `json:"price"`     // prezzo di mercato (con markup)
	Sold      bool   `json:"sold"`
	NegoDone  bool   `json:"nego_done"` // già contrattato
	Analyzed  bool   `json:"analyzed"`  // analizzato dal giocatore prima dell'acquisto
}

type MarketState struct {
	Listings []MarketListing `json:"listings"`
	Location string          `json:"location"` // dove è stato generato
}

type NegotiateResult struct {
	Success     bool   `json:"success"`
	NewPrice    int    `json:"new_price"`
	OldPrice    int    `json:"old_price"`
	DiscountPct int    `json:"discount_pct"`
	Narrative   string `json:"narrative"` // testo GM breve
}
