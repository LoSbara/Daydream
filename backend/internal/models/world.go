package models

import "time"

type WorldFlag struct {
	ID          string    `json:"id,omitempty"`
	CharacterID string    `json:"character_id"`
	Scope       string    `json:"scope"`
	Key         string    `json:"key"`
	Value       string    `json:"value"`
	Description string    `json:"description"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type GMWorldFlag struct {
	Scope       string `json:"scope"`
	Key         string `json:"key"`
	Value       string `json:"value"`
	Description string `json:"description"`
}
