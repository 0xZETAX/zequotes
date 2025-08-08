package handler

import (
	"bytes"
	"crypto/sha1"
	_ "embed"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"time"
)

//go:embed data/quotes.json
var quotesJSON []byte

type Quote struct {
	ID       string `json:"id"`
	Author   string `json:"author"`
	Text     string `json:"text"`
	Lang     string `json:"lang"`
	Category string `json:"category"`
}

var (
	allQuotes []Quote
	etag      string
)

func init() {
	if err := json.Unmarshal(quotesJSON, &allQuotes); err != nil {
		panic("failed to parse quotes.json: " + err.Error())
	}
	sum := sha1.Sum(quotesJSON)
	etag = fmt.Sprintf(`W/"%x"`, sum)
	rand.Seed(time.Now().UnixNano())
}

func Handler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Vary", "Origin,Accept-Encoding,If-None-Match")
	w.Header().Set("Cache-Control", "public, s-maxage=600, stale-while-revalidate=86400")
	w.Header().Set("ETag", etag)
	w.Header().Set("Content-Type", "application/json; charset=utf-8")

	if inm := r.Header.Get("If-None-Match"); inm != "" && inm == etag {
		w.WriteHeader(http.StatusNotModified)
		return
	}

	switch r.Method {
	case http.MethodGet:
		handleGET(w, r)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
		_ = json.NewEncoder(w).Encode(map[string]any{"error": "method_not_allowed"})
	}
}

func handleGET(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	id := q.Get("id")
	lang := strings.ToLower(q.Get("lang"))
	cat := strings.ToLower(q.Get("category"))
	limit := 1
	if ls := q.Get("limit"); ls != "" {
		if n, err := strconv.Atoi(ls); err == nil && n > 0 && n <= 100 {
			limit = n
		}
	}

	filtered := allQuotes[:0]
	for _, it := range allQuotes {
		if id != "" && it.ID != id {
			continue
		}
		if lang != "" && strings.ToLower(it.Lang) != lang {
			continue
		}
		if cat != "" && strings.ToLower(it.Category) != cat {
			continue
		}
		filtered = append(filtered, it)
	}
	if len(filtered) == 0 {
		filtered = allQuotes
	}

	out := sample(filtered, limit)

	resp, _ := json.Marshal(map[string]any{
		"count":  len(out),
		"result": out,
	})

	if q.Get("pretty") == "1" {
		var buf bytes.Buffer
		if err := json.Indent(&buf, resp, "", "  "); err == nil {
			resp = buf.Bytes()
		}
	}

	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(resp)
}

func sample(src []Quote, n int) []Quote {
	if n >= len(src) {
		cp := append([]Quote(nil), src...)
		rand.Shuffle(len(cp), func(i, j int) { cp[i], cp[j] = cp[j], cp[i] })
		return cp
	}
	seen := make(map[int]struct{}, n)
	out := make([]Quote, 0, n)
	for len(out) < n {
		i := rand.Intn(len(src))
		if _, ok := seen[i]; ok {
			continue
		}
		seen[i] = struct{}{}
		out = append(out, src[i])
	}
	return out
}
