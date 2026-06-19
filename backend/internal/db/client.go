package db

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"time"
)

var ErrNotFound = errors.New("record not found")

// Client è un thin wrapper sull'HTTP API di SurrealDB.
// Non usa l'SDK Go ufficiale per evitare dipendenze da breaking changes del SDK.
type Client struct {
	baseURL string
	ns      string
	db      string
	user    string
	pass    string
	http    *http.Client
}

// QueryResult rappresenta il risultato di una singola istruzione SurrealQL.
type QueryResult struct {
	Result json.RawMessage `json:"result"`
	Status string          `json:"status"`
	Time   string          `json:"time"`
}

// First deserializza il primo record del risultato in v.
// Ritorna ErrNotFound se il risultato è vuoto.
func (qr QueryResult) First(v any) error {
	if qr.Status != "OK" {
		var errMsg string
		json.Unmarshal(qr.Result, &errMsg) //nolint
		return fmt.Errorf("surreal: %s", errMsg)
	}
	var records []json.RawMessage
	if err := json.Unmarshal(qr.Result, &records); err != nil {
		return fmt.Errorf("surreal: parse results: %w", err)
	}
	if len(records) == 0 {
		return ErrNotFound
	}
	return json.Unmarshal(records[0], v)
}

// All deserializza tutti i record del risultato in v (deve essere un puntatore a slice).
func (qr QueryResult) All(v any) error {
	if qr.Status != "OK" {
		var errMsg string
		json.Unmarshal(qr.Result, &errMsg) //nolint
		return fmt.Errorf("surreal: %s", errMsg)
	}
	return json.Unmarshal(qr.Result, v)
}

// New crea e verifica un Client SurrealDB.
func New() (*Client, error) {
	c := &Client{
		baseURL: os.Getenv("SURREAL_URL"),
		ns:      os.Getenv("SURREAL_NS"),
		db:      os.Getenv("SURREAL_DB"),
		user:    os.Getenv("SURREAL_USER"),
		pass:    os.Getenv("SURREAL_PASS"),
		http:    &http.Client{Timeout: 10 * time.Second},
	}
	if err := c.ping(); err != nil {
		return nil, fmt.Errorf("surrealdb connection failed: %w", err)
	}
	return c, nil
}

func (c *Client) ping() error {
	_, err := c.Query("RETURN 1;", nil)
	return err
}

// Query esegue una o più istruzioni SurrealQL e ritorna i risultati.
// vars è una mappa di variabili rese disponibili come $chiave nel query.
func (c *Client) Query(sql string, vars map[string]any) ([]QueryResult, error) {
	endpoint := c.baseURL + "/sql"

	if len(vars) > 0 {
		params := url.Values{}
		for k, v := range vars {
			params.Set(k, fmt.Sprintf("%v", v))
		}
		endpoint += "?" + params.Encode()
	}

	req, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewBufferString(sql))
	if err != nil {
		return nil, err
	}
	req.SetBasicAuth(c.user, c.pass)
	req.Header.Set("surreal-ns", c.ns)
	req.Header.Set("surreal-db", c.db)
	req.Header.Set("Content-Type", "text/plain")
	req.Header.Set("Accept", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("surreal http: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("surreal http %d: %s", resp.StatusCode, body)
	}

	var results []QueryResult
	if err := json.Unmarshal(body, &results); err != nil {
		return nil, fmt.Errorf("surreal: decode response: %w", err)
	}
	return results, nil
}

// Exec è una scorciatoia per query che non restituiscono dati significativi (UPDATE, DELETE, DEFINE...).
func (c *Client) Exec(sql string, vars map[string]any) error {
	results, err := c.Query(sql, vars)
	if err != nil {
		return err
	}
	for _, r := range results {
		if r.Status != "OK" {
			var errMsg string
			json.Unmarshal(r.Result, &errMsg) //nolint
			return fmt.Errorf("surreal exec: %s", errMsg)
		}
	}
	return nil
}

// UpdateRecord aggiorna un record con CONTENT, preservando il record ID.
// Serializza data in JSON, rimuove il campo "id" per evitare conflitti,
// e poi esegue: UPDATE <recordID> CONTENT <json>.
func (c *Client) UpdateRecord(recordID string, data any) error {
	jsonBytes, err := json.Marshal(data)
	if err != nil {
		return err
	}
	// Strip "id" field: SurrealDB gestisce l'ID del record separatamente
	var m map[string]any
	if err := json.Unmarshal(jsonBytes, &m); err != nil {
		return err
	}
	delete(m, "id")

	contentBytes, err := json.Marshal(m)
	if err != nil {
		return err
	}

	sql := fmt.Sprintf("UPDATE %s CONTENT %s;", recordID, string(contentBytes))
	return c.Exec(sql, nil)
}

// CreateRecord crea un nuovo record nella tabella specificata.
// Restituisce il record creato deserializzato in result.
func (c *Client) CreateRecord(table string, data any, result any) error {
	jsonBytes, err := json.Marshal(data)
	if err != nil {
		return err
	}
	var m map[string]any
	if err := json.Unmarshal(jsonBytes, &m); err != nil {
		return err
	}
	delete(m, "id") // lascia che SurrealDB generi l'ID

	contentBytes, err := json.Marshal(m)
	if err != nil {
		return err
	}

	sql := fmt.Sprintf("CREATE %s CONTENT %s;", table, string(contentBytes))
	results, err := c.Query(sql, nil)
	if err != nil {
		return err
	}
	return results[0].First(result)
}
