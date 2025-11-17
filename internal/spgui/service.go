// sharepoint-gui/internal/spgui/service.go
package spgui

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"golang.org/x/net/http2"

	"github.com/koltyakov/gosip"
	"github.com/wailsapp/wails/v2/pkg/runtime"

	edgeondemand "github.com/vba-excel/sp-edgeondemand"
	"github.com/vba-excel/spapi"
)

type Config struct {
	ConfigPath       string // ex.: "private.json"
	SiteURL          string // override opcional
	GlobalTimeoutSec int    // 0 = sem limite global; >0 em segundos
	CleanOutput      bool   // equivalente ao --clean-output

	// ---- Tuning opcional do HTTP Transport (não exposto ao frontend/Wails) ----
	HTTPMaxIdleConns        int  `json:"-"`
	HTTPMaxIdleConnsPerHost int  `json:"-"`
	HTTPIdleConnTimeoutSec  int  `json:"-"`
	HTTPTLSHandshakeTOsec   int  `json:"-"` // default 10
	HTTPDisableKeepAlives   bool `json:"-"`
}

type SPGUI struct {
	cfg   Config
	svc   *spapi.SPService
	spcli *gosip.SPClient
	ctx   context.Context // contexto do Wails para diálogos, etc.

	// cancelamento cooperativo
	mu        sync.Mutex
	curCancel context.CancelFunc
	curOpID   uint64
}

func NewSPGUI(cfg Config) *SPGUI { return &SPGUI{cfg: cfg} }

// Guardar o contexto do Wails (chamado em OnStartup)
func (g *SPGUI) SetContext(ctx context.Context) { g.ctx = ctx }

// Permite trocar config em runtime a partir do GUI
func (g *SPGUI) SetConfig(cfg Config) {
	g.cfg = cfg
	g.svc = nil
	g.spcli = nil
}

// ===== DTOs =====

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

// SPAttachmentInfo é o espelho do spapi.AttachmentInfo para o Wails.
type SPAttachmentInfo struct {
	FileName          string `json:"fileName"`
	ServerRelativeURL string `json:"serverRelativeUrl"`
}

// ===== Métodos expostos (core) =====

func (g *SPGUI) Ping() string { return "ok" }

// Abre um diálogo nativo para escolher o private.json (feito no backend)
func (g *SPGUI) OpenConfigDialog() (string, error) {
	if g.ctx == nil {
		return "", fmt.Errorf("runtime context indisponível")
	}
	path, err := runtime.OpenFileDialog(g.ctx, runtime.OpenDialogOptions{
		Title: "Escolher private.json",
		Filters: []runtime.FileFilter{
			{DisplayName: "JSON", Pattern: "*.json"},
		},
	})
	return path, err
}

// CancelCurrent cancela a operação em curso (se existir).
// Devolve true se havia algo para cancelar.
func (g *SPGUI) CancelCurrent() bool {
	g.mu.Lock()
	cancel := g.curCancel
	// limpamos já a referência; cancel() é idempotente
	g.curCancel = nil
	g.mu.Unlock()

	if cancel != nil {
		cancel()
		return true
	}
	return false
}

func (g *SPGUI) ListItems(q ListQuery) (*ListResponse, error) {
	if err := g.ensureService(); err != nil {
		return nil, err
	}
	ctx, cancel := g.opCtx()
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

	items = maybeCleanSlice(items, g.cfg.CleanOutput)

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

func (g *SPGUI) GetItem(list string, id int, selectFields string) (map[string]any, error) {
	if err := g.ensureService(); err != nil {
		return nil, err
	}
	ctx, cancel := g.opCtx()
	defer cancel()
	m, err := g.svc.GetItemByID(ctx, list, id, selectFields)
	if err != nil {
		return nil, err
	}
	return maybeCleanMap(m, g.cfg.CleanOutput), nil
}

func (g *SPGUI) AddItem(list string, fields map[string]any, selectFields string) (map[string]any, error) {
	if err := g.ensureService(); err != nil {
		return nil, err
	}
	ctx, cancel := g.opCtx()
	defer cancel()

	created, err := g.svc.AddItem(ctx, list, fields)
	if err != nil {
		return nil, err
	}
	id := extractIDFromMap(created)
	if id <= 0 {
		return maybeCleanMap(created, g.cfg.CleanOutput), nil
	}
	m, err := g.svc.GetItemByID(ctx, list, id, selectFields)
	if err != nil {
		return nil, err
	}
	return maybeCleanMap(m, g.cfg.CleanOutput), nil
}

func (g *SPGUI) UpdateItem(list string, id int, fields map[string]any, selectFields string) (map[string]any, error) {
	if err := g.ensureService(); err != nil {
		return nil, err
	}
	if id <= 0 {
		return nil, fmt.Errorf("id inválido")
	}
	ctx, cancel := g.opCtx()
	defer cancel()

	if _, err := g.svc.UpdateItem(ctx, list, id, fields); err != nil {
		return nil, err
	}
	m, err := g.svc.GetItemByID(ctx, list, id, selectFields)
	if err != nil {
		return nil, err
	}
	return maybeCleanMap(m, g.cfg.CleanOutput), nil
}

func (g *SPGUI) DeleteItem(list string, id int) (bool, error) {
	if err := g.ensureService(); err != nil {
		return false, err
	}
	if id <= 0 {
		return false, fmt.Errorf("id inválido")
	}
	ctx, cancel := g.opCtx()
	defer cancel()

	if err := g.svc.DeleteItem(ctx, list, id); err != nil {
		return false, err
	}
	return true, nil
}

// ===== Métodos expostos (Anexos) =====

// ListAttachments devolve anexos do item.
func (g *SPGUI) ListAttachments(list string, id int) ([]SPAttachmentInfo, error) {
	if err := g.ensureService(); err != nil {
		return nil, err
	}
	ctx, cancel := g.opCtx()
	defer cancel()

	atts, err := g.svc.ListAttachments(ctx, list, id)
	if err != nil {
		return nil, err
	}
	out := make([]SPAttachmentInfo, 0, len(atts))
	for _, a := range atts {
		out = append(out, toSPA(a))
	}
	return out, nil
}

// AddAttachment faz upload (ou substitui) um anexo. Conteúdo recebido como []byte (Wails/JS → Go)
func (g *SPGUI) AddAttachment(list string, id int, fileName string, content []byte) (SPAttachmentInfo, error) {
	if err := g.ensureService(); err != nil {
		return SPAttachmentInfo{}, err
	}
	ctx, cancel := g.opCtx()
	defer cancel()

	info, err := g.svc.AddAttachment(ctx, list, id, fileName, bytes.NewReader(content))
	if err != nil {
		return SPAttachmentInfo{}, err
	}
	return toSPA(info), nil
}

// DownloadAttachment devolve os bytes do anexo (para guardar no disco no frontend).
func (g *SPGUI) DownloadAttachment(list string, id int, name string) ([]byte, error) {
	if err := g.ensureService(); err != nil {
		return nil, err
	}
	ctx, cancel := g.opCtx()
	defer cancel()

	b, err := g.svc.DownloadAttachment(ctx, list, id, name)
	if g.ctx != nil {
		runtime.LogDebugf(g.ctx, "DownloadAttachment %s: %d bytes", name, len(b))
	}
	return b, err
}

// DeleteAttachment remove o anexo por nome.
func (g *SPGUI) DeleteAttachment(list string, id int, fileName string) (bool, error) {
	if err := g.ensureService(); err != nil {
		return false, err
	}
	ctx, cancel := g.opCtx()
	defer cancel()

	if err := g.svc.DeleteAttachment(ctx, list, id, fileName); err != nil {
		return false, err
	}
	return true, nil
}

// DownloadByURL faz download de um ficheiro dado um URL absoluto
// ou um server-relative path. Devolve o conteúdo em bytes.
func (g *SPGUI) DownloadByURL(url string) ([]byte, error) {
	if err := g.ensureService(); err != nil {
		return nil, err
	}
	ctx, cancel := g.opCtx()
	defer cancel()

	b, err := g.svc.DownloadByURL(ctx, url)
	if g.ctx != nil {
		runtime.LogDebugf(g.ctx, "DownloadByURL %s: %d bytes", url, len(b))
	}
	return b, err
}

// ===== NOVO: variantes com Save Dialog (nativas) =====

// SaveAttachmentPick: abre save dialog e grava (stream) um anexo pelo nome.
func (g *SPGUI) SaveAttachmentPick(list string, id int, fileName string) (string, error) {
	if err := g.ensureService(); err != nil {
		return "", err
	}
	if g.ctx == nil {
		return "", fmt.Errorf("runtime context indisponível")
	}

	savePath, err := runtime.SaveFileDialog(g.ctx, runtime.SaveDialogOptions{
		Title:           "Guardar anexo como…",
		DefaultFilename: fileName,
	})
	if err != nil || savePath == "" {
		return savePath, err
	}

	ctx, cancel := g.opCtx()
	defer cancel()

	rc, err := g.svc.DownloadAttachmentReader(ctx, list, id, fileName)
	if err != nil {
		return "", err
	}
	defer rc.Close()

	out, err := os.Create(savePath)
	if err != nil {
		return "", err
	}
	defer out.Close()

	if _, err := io.Copy(out, rc); err != nil {
		return "", err
	}
	return savePath, nil
}

// SaveByURLPick: descarrega por URL absoluto ou server-relative e grava com save dialog.
func (g *SPGUI) SaveByURLPick(urlOrPath string) (string, error) {
	if err := g.ensureService(); err != nil {
		return "", err
	}
	if g.ctx == nil {
		return "", fmt.Errorf("runtime context indisponível")
	}

	base := filepath.Base(urlOrPath)
	if base == "" || base == "/" || base == "." {
		base = path.Base(urlOrPath)
		if base == "" || base == "/" || base == "." {
			base = "download.bin"
		}
	}

	savePath, err := runtime.SaveFileDialog(g.ctx, runtime.SaveDialogOptions{
		Title:           "Guardar ficheiro…",
		DefaultFilename: base,
	})
	if err != nil || savePath == "" {
		return savePath, err
	}

	ctx, cancel := g.opCtx()
	defer cancel()

	rc, err := g.svc.DownloadByURLReader(ctx, urlOrPath)
	if err != nil {
		return "", err
	}
	defer rc.Close()

	out, err := os.Create(savePath)
	if err != nil {
		return "", err
	}
	defer out.Close()

	if _, err := io.Copy(out, rc); err != nil {
		return "", err
	}
	return savePath, nil
}

// SaveBytesPick: utilitário para exports (frontend -> backend -> disco via diálogo)
func (g *SPGUI) SaveBytesPick(defaultFilename string, content []byte, mime string) (string, error) {
	if g.ctx == nil {
		return "", fmt.Errorf("runtime context indisponível")
	}
	savePath, err := runtime.SaveFileDialog(g.ctx, runtime.SaveDialogOptions{
		Title:           "Guardar ficheiro…",
		DefaultFilename: defaultFilename,
	})
	if err != nil || savePath == "" {
		return savePath, err
	}
	if err := os.WriteFile(savePath, content, 0o644); err != nil {
		return "", err
	}
	return savePath, nil
}

// ===== Helpers =====

func (g *SPGUI) JSON(v any) string {
	b, _ := json.MarshalIndent(v, "", "  ")
	return string(b)
}

func toSPA(a spapi.AttachmentInfo) SPAttachmentInfo {
	return SPAttachmentInfo{FileName: a.FileName, ServerRelativeURL: a.ServerRelativeURL}
}

func maybeCleanSlice(items []map[string]any, clean bool) []map[string]any {
	if !clean {
		return items
	}
	out := make([]map[string]any, 0, len(items))
	for _, it := range items {
		out = append(out, cleanMapInternal(it))
	}
	return out
}

func maybeCleanMap(m map[string]any, clean bool) map[string]any {
	if !clean {
		return m
	}
	return cleanMapInternal(m)
}

func cleanMapInternal(m map[string]any) map[string]any {
	cleaned := make(map[string]any, len(m))
	for k, v := range m {
		if strings.HasPrefix(k, "__") {
			continue
		}
		cleaned[k] = v
	}
	return cleaned
}

// ===== Internos =====

func (g *SPGUI) opCtx() (context.Context, context.CancelFunc) {
	g.mu.Lock()
	g.curOpID++
	myID := g.curOpID
	g.mu.Unlock()

	var (
		ctx    context.Context
		cancel context.CancelFunc
	)
	if g.cfg.GlobalTimeoutSec > 0 {
		ctx, cancel = context.WithTimeout(context.Background(), time.Duration(g.cfg.GlobalTimeoutSec)*time.Second)
	} else {
		ctx, cancel = context.WithCancel(context.Background())
	}

	g.mu.Lock()
	g.curCancel = cancel
	g.mu.Unlock()

	wrapped := func() {
		cancel()
		g.mu.Lock()
		if g.curOpID == myID {
			g.curCancel = nil
		}
		g.mu.Unlock()
	}
	return ctx, wrapped
}

func (g *SPGUI) ensureService() error {
	if g.svc != nil {
		return nil
	}
	if g.cfg.ConfigPath == "" {
		g.cfg.ConfigPath = "private.json"
	}

	auth := &edgeondemand.AuthCnfg{}
	if err := auth.ReadConfig(g.cfg.ConfigPath); err != nil {
		return fmt.Errorf("ler %s: %w", g.cfg.ConfigPath, err)
	}
	if g.cfg.SiteURL != "" {
		auth.SiteURL = g.cfg.SiteURL
	}
	if auth.EdgeOptions == nil {
		auth.EdgeOptions = &edgeondemand.EdgeConfig{}
	}
	effectiveTimeout := 30 * time.Second
	if auth.EdgeOptions.TimeoutSeconds > 0 {
		effectiveTimeout = time.Duration(auth.EdgeOptions.TimeoutSeconds) * time.Second
	}

	// ---- Defaults seguros para ambientes com proxy (Zscaler)
	maxIdle := g.cfg.HTTPMaxIdleConns
	if maxIdle <= 0 {
		maxIdle = 40
	}
	perHost := g.cfg.HTTPMaxIdleConnsPerHost
	if perHost <= 0 {
		perHost = 4
	}
	idleSec := g.cfg.HTTPIdleConnTimeoutSec
	if idleSec <= 0 {
		idleSec = 20 // fechar antes do proxy
	}
	tlsHS := g.cfg.HTTPTLSHandshakeTOsec
	if tlsHS <= 0 {
		tlsHS = 10
	}

	httpTransport := &http.Transport{
		Proxy:                 http.ProxyFromEnvironment,
		TLSHandshakeTimeout:   time.Duration(tlsHS) * time.Second,
		MaxIdleConns:          maxIdle,
		MaxIdleConnsPerHost:   perHost,
		IdleConnTimeout:       time.Duration(idleSec) * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}
	httpTransport.DisableKeepAlives = g.cfg.HTTPDisableKeepAlives

	// Ativa HTTP/2 quando possível
	_ = http2.ConfigureTransport(httpTransport)

	// “Janitor”: fecha ligações ociosas periodicamente (< IdleConnTimeout)
	go func(tr *http.Transport, idleSeconds int) {
		interval := time.Duration(idleSeconds/2) * time.Second
		if interval < 2*time.Second {
			interval = 2 * time.Second
		}
		t := time.NewTicker(interval)
		defer t.Stop()
		for range t.C {
			tr.CloseIdleConnections()
		}
	}(httpTransport, idleSec)

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

func extractIDFromMap(m map[string]any) int {
	for _, k := range []string{"ID", "Id", "id"} {
		if v, ok := m[k]; ok {
			switch vv := v.(type) {
			case int:
				return vv
			case int32:
				return int(vv)
			case int64:
				return int(vv)
			case float32:
				return int(vv)
			case float64:
				return int(vv)
			}
		}
	}
	return 0
}
