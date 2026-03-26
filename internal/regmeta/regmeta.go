package regmeta

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/depscount/depscount/internal/httpclient"
	"github.com/depscount/depscount/pkg/types"
)

// Enrich fills License and PublishedAt on graph nodes (best-effort, cached in-memory per scan).
func Enrich(ctx context.Context, hc *httpclient.Client, graph *types.DependencyGraph) {
	var lim sync.Mutex
	seen := make(map[string]struct{})
	var wg sync.WaitGroup
	sem := make(chan struct{}, 8)
	for _, p := range graph.Nodes {
		p := p
		key := p.Ecosystem + "/" + p.Name + "@" + p.Version
		lim.Lock()
		if _, ok := seen[key]; ok {
			lim.Unlock()
			continue
		}
		seen[key] = struct{}{}
		lim.Unlock()
		wg.Add(1)
		sem <- struct{}{}
		go func() {
			defer wg.Done()
			defer func() { <-sem }()
			switch p.Ecosystem {
			case "npm":
				fillNPM(ctx, hc, p)
			case "pypi":
				fillPyPI(ctx, hc, p)
			}
		}()
	}
	wg.Wait()
}

func fillNPM(ctx context.Context, hc *httpclient.Client, p *types.Package) {
	enc := strings.ReplaceAll(p.Name, "/", "%2F")
	u := "https://registry.npmjs.org/" + enc
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return
	}
	resp, err := hc.Do(ctx, req)
	if err != nil {
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return
	}
	var doc struct {
		Versions map[string]struct {
			License any `json:"license"` // string or {type: ...}
		} `json:"versions"`
		Time map[string]string `json:"time"`
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if err != nil {
		return
	}
	if err := json.Unmarshal(body, &doc); err != nil {
		return
	}
	if v, ok := doc.Versions[p.Version]; ok {
		p.License = npmLicenseString(v.License)
	}
	if t, ok := doc.Time[p.Version]; ok {
		if ts, err := time.Parse(time.RFC3339, t); err == nil {
			p.PublishedAt = ts
		}
	}
	if t, ok := doc.Time["modified"]; ok {
		if ts, err := time.Parse(time.RFC3339, t); err == nil {
			p.RegistryModifiedAt = ts
		}
	}
}

func npmLicenseString(v any) string {
	switch x := v.(type) {
	case string:
		return x
	case map[string]any:
		if s, ok := x["type"].(string); ok {
			return s
		}
	default:
		return ""
	}
	return ""
}

func fillPyPI(ctx context.Context, hc *httpclient.Client, p *types.Package) {
	u := fmt.Sprintf("https://pypi.org/pypi/%s/%s/json", url.PathEscape(p.Name), url.PathEscape(p.Version))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return
	}
	resp, err := hc.Do(ctx, req)
	if err != nil {
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return
	}
	var doc struct {
		Info struct {
			License     string `json:"license"`
			LicenseExpr string `json:"license_expression"`
		} `json:"info"`
		URLs []struct {
			UploadTime time.Time `json:"upload_time"`
		} `json:"urls"`
		Releases map[string][]struct {
			UploadTime time.Time `json:"upload_time"`
		} `json:"releases"`
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if err != nil {
		return
	}
	if err := json.Unmarshal(body, &doc); err != nil {
		return
	}
	lic := doc.Info.LicenseExpr
	if lic == "" {
		lic = doc.Info.License
	}
	p.License = lic
	if len(doc.URLs) > 0 && !doc.URLs[0].UploadTime.IsZero() {
		p.PublishedAt = doc.URLs[0].UploadTime
	}
	var latest time.Time
	for _, rels := range doc.Releases {
		for _, r := range rels {
			if r.UploadTime.After(latest) {
				latest = r.UploadTime
			}
		}
	}
	if !latest.IsZero() {
		p.RegistryModifiedAt = latest
	}
}
