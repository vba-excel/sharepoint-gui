package spgui

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
	"net/http"

	"github.com/koltyakov/gosip"

	"github.com/vba-excel/spapi"
	edgeondemand "github.com/vba-excel/sp-edgeondemand"
)

type Config struct {
	ConfigPath string // ex.: "private.json"
	SiteURL    string // override opcional
}

type SPGUI struct {
	cfg   Config
	svc   *spapi.SPService
	spcli *gosip.SPClient
}

func NewSPGUI(cfg Config) *SPGUI { return &SPGUI{cfg: cfg} }

// Configurar/alterar caminho do config em runtime (opcional).
func (g *SPGUI) SetConfig(cfg Config) { g.cfg = cfg; g.svc = nil; g.spcli = nil }

// Bind-friendly DTOs (evitar any nos parâmetros da fronteira)
type ListQuery struct {
	ListName   string `json:"list"`
	Select     string `json:"select"`
	Filter     string `json:"filter"`
	OrderBy    string `json:"orderby"`
	Top        int    `json:"top"`
	All        bool   `json:"all"`
	LatestOnly bool   `json:"latestOnly"`
}

type QuerySummary struct {
	Items        int  `json:"items"`
	PagesFetched int  `json:"pages"`
	Throttled    bool `json:"throttled"`
	Partial      bool `json:"partial"`
	UsedFallback bool `json:"fallback"`
	StoppedEarly bool `json:"stoppedEarly"`
}

type ListResponse struct {
	Items   []map[string]any `json:"items"`
	Summary QuerySummary     `json:"summary"`
}

// ---------- Public methods (expostos ao frontend) ----------

// Ping só para sanity-check do binding.
func (g *SPGUI) Ping() string { return "ok" }

// ListItems GUI → usa spapi internamente
func (g *SPGUI) ListItems(q ListQuery) (*ListResponse, error) {
	if err := g.ensureService(); err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	items, stats, err := g.svc.ListItems(ctx, spapi.ListQueryOptions{
		ListName:   q.ListName,
		Select:     q.Select,
		Filter:     q.Filter,
		OrderBy:    q.OrderBy,
		Top:        q.Top,
		All:        q.All,
		LatestOnly: q.LatestOnly,
	})
	if err != nil {
		return nil, err
	}
	return &ListResponse{
		Items: items,
		Summary: QuerySummary{
			Items:        len(items),
			PagesFetched: stats.PagesFetched,
			Throttled:    stats.Throttled,
			Partial:      stats.Partial,
			UsedFallback: stats.UsedFallback,
			StoppedEarly: stats.StoppedEarly,
		},
	}, nil
}

// GetItem simples (exemplo)
func (g *SPGUI) GetItem(list string, id int, selectFields string) (map[string]any, error) {
	if err := g.ensureService(); err != nil { return nil, err }
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	return g.svc.GetItemByID(ctx, list, id, selectFields)
}

// ---------- Internals ----------

func (g *SPGUI) ensureService() error {
	if g.svc != nil {
		return nil
	}
	if g.cfg.ConfigPath == "" {
		g.cfg.ConfigPath = "private.json"
	}

	// carregar AuthCnfg (edge on demand)
	auth := &edgeondemand.AuthCnfg{}
	if err := auth.ReadConfig(g.cfg.ConfigPath); err != nil {
		return fmt.Errorf("ler %s: %w", g.cfg.ConfigPath, err)
	}
	if g.cfg.SiteURL != "" {
		auth.SiteURL = g.cfg.SiteURL
	}
	// defaults mínimos (edgeondemand já normaliza também)
	if auth.EdgeOptions == nil {
		auth.EdgeOptions = &edgeondemand.EdgeConfig{}
	}
	if auth.EdgeOptions.TimeoutSeconds == 0 {
		auth.EdgeOptions.TimeoutSeconds = 180
	}

	// http.Client semelhante ao do CLI
	effectiveTimeout := 30 * time.Second
	if auth.EdgeOptions != nil && auth.EdgeOptions.TimeoutSeconds > 0 {
		effectiveTimeout = time.Duration(auth.EdgeOptions.TimeoutSeconds) * time.Second
	}
	httpTransport := &http.Transport{
		Proxy:                 http.ProxyFromEnvironment,
		TLSHandshakeTimeout:   10 * time.Second,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}
	spHTTPClient := http.Client{
		Timeout:   effectiveTimeout,
		Transport: httpTransport,
	}
	client := &gosip.SPClient{
		Client:     spHTTPClient,
		AuthCnfg:   auth,
		ConfigPath: g.cfg.ConfigPath,
	}
	g.spcli = client
	g.svc = spapi.New(client)
	return nil
}

// Util: pretty (útil para debug via frontend)
func (g *SPGUI) JSON(v any) string {
	b, _ := json.MarshalIndent(v, "", "  ")
	return string(b)
}
