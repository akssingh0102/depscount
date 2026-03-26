package security

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/depscount/depscount/internal/httpclient"
)

const osvBatchURL = "https://api.osv.dev/v1/querybatch"

type batchRequest struct {
	Queries []batchQuery `json:"queries"`
}

type batchQuery struct {
	Package struct {
		Name      string `json:"name"`
		Ecosystem string `json:"ecosystem"`
	} `json:"package"`
	Version string `json:"version"`
}

type batchResponse struct {
	Results []struct {
		Vulns []osvVuln `json:"vulns"`
	} `json:"results"`
}

type osvVuln struct {
	ID       string        `json:"id"`
	Summary  string        `json:"summary"`
	Details  string        `json:"details"`
	Severity []osvSeverity `json:"severity"`
}

type osvSeverity struct {
	Type  string `json:"type"`
	Score string `json:"score"`
}

func osvEcosystem(eco string) string {
	switch strings.ToLower(eco) {
	case "npm":
		return "npm"
	case "pypi":
		return "PyPI"
	default:
		return eco
	}
}

func queryBatch(ctx context.Context, hc *httpclient.Client, queries []batchQuery) (*batchResponse, error) {
	body, err := json.Marshal(batchRequest{Queries: queries})
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, osvBatchURL, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := hc.Do(ctx, req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		slurp, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return nil, fmt.Errorf("osv: %s: %s", resp.Status, string(slurp))
	}
	var out batchResponse
	if err := json.NewDecoder(io.LimitReader(resp.Body, 32<<20)).Decode(&out); err != nil {
		return nil, err
	}
	return &out, nil
}

func cvssBaseScore(v osvVuln) float64 {
	best := 0.0
	for _, s := range v.Severity {
		if x := parseFloatScore(s.Score); x > best {
			best = x
		}
	}
	return best
}

func parseFloatScore(s string) float64 {
	var x float64
	_, err := fmt.Sscanf(strings.TrimSpace(s), "%f", &x)
	if err != nil {
		return 0
	}
	return x
}
