package db

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
)

// Migrate crea namespace/database se non esistono, poi applica lo schema.
// Le istruzioni DEFINE sono idempotenti: si può chiamare ad ogni avvio.
func (c *Client) Migrate() error {
	// Fase 1: crea namespace e database a livello root (senza headers NS/DB)
	if err := c.execRoot(fmt.Sprintf(
		"DEFINE NAMESPACE IF NOT EXISTS %s; USE NS %s; DEFINE DATABASE IF NOT EXISTS %s;",
		c.ns, c.ns, c.db,
	)); err != nil {
		return fmt.Errorf("migrate: create ns/db: %w", err)
	}

	// Fase 2: applica lo schema nel contesto NS/DB
	schemaPath := schemaFilePath()
	data, err := os.ReadFile(schemaPath)
	if err != nil {
		return fmt.Errorf("migrate: read schema: %w", err)
	}
	if err := c.Exec(string(data), nil); err != nil {
		return fmt.Errorf("migrate: exec schema: %w", err)
	}
	return nil
}

// execRoot esegue SQL senza headers NS/DB (livello root).
func (c *Client) execRoot(sql string) error {
	req, err := http.NewRequest(http.MethodPost, c.baseURL+"/sql", bytes.NewBufferString(sql))
	if err != nil {
		return err
	}
	req.SetBasicAuth(c.user, c.pass)
	req.Header.Set("Content-Type", "text/plain")
	req.Header.Set("Accept", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return fmt.Errorf("surreal root http %d: %s", resp.StatusCode, body)
	}

	var results []QueryResult
	if err := json.Unmarshal(body, &results); err != nil {
		return err
	}
	for _, r := range results {
		if r.Status != "OK" {
			var errMsg string
			json.Unmarshal(r.Result, &errMsg) //nolint
			return fmt.Errorf("surreal root: %s", errMsg)
		}
	}
	return nil
}

func schemaFilePath() string {
	// In sviluppo: percorso relativo alla root del modulo Go.
	// In produzione: usa la variabile d'ambiente SCHEMA_PATH se definita.
	if p := os.Getenv("SCHEMA_PATH"); p != "" {
		return p
	}
	_, file, _, _ := runtime.Caller(0)
	// file = backend/internal/db/migrate.go  →  ../.. = backend/
	root := filepath.Join(filepath.Dir(file), "..", "..")
	return filepath.Join(root, "configs", "schema.surql")
}
