package fraudadmin

import (
	"encoding/hex"
	"encoding/json"
	"net/http"
	"os"

	"ad-event-processor/internal/track"
	"ad-event-processor/pkg/coldpath"
	"ad-event-processor/pkg/httpresponse"
	"ad-event-processor/pkg/wasmattest"
)

type wasmAttestDryRunRequest struct {
	SaltHex     string `json:"salt_hex"`
	Difficulty  uint8  `json:"difficulty"`
	MaxTries    uint32 `json:"max_tries"`
	BenchRounds uint32 `json:"bench_rounds"`
}

type wasmAttestDryRunResponse struct {
	Nonce          uint32 `json:"nonce"`
	FloatNoiseHex  string `json:"float_noise_hex"`
	BenchAcc       uint32 `json:"bench_acc"`
	Sha256EmptyHex string `json:"sha256_empty_hex"`
	ParityOK       bool   `json:"parity_ok"`
}

func (h *HTTPHandlers) registerWasmAttestRoutes(mux *http.ServeMux, limit func(http.HandlerFunc) http.HandlerFunc, perm func(string, http.HandlerFunc) http.HandlerFunc) {
	mux.HandleFunc("POST /api/v1/fraud/wasm-attest/dry-run", limit(perm("fraud:read", h.postWasmAttestDryRun)))
}

func (h *HTTPHandlers) postWasmAttestDryRun(w http.ResponseWriter, r *http.Request) {
	if os.Getenv("WASM_ATTEST_ENABLED") == "0" {
		httpresponse.Error(w, http.StatusNotImplemented, "NOT_IMPLEMENTED", "wasm attest disabled")
		return
	}
	body, err := coldpath.ReadLimitedBody(w, r, coldpath.DefaultMaxBody)
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid body")
		return
	}
	var req wasmAttestDryRunRequest
	if err := json.Unmarshal(body, &req); err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid json")
		return
	}
	if req.Difficulty == 0 {
		req.Difficulty = 2
	}
	if req.MaxTries == 0 {
		req.MaxTries = wasmattest.DefaultMaxTries
	}
	if req.BenchRounds == 0 {
		req.BenchRounds = 100_000
	}
	salt, err := hex.DecodeString(req.SaltHex)
	if err != nil || len(salt) != wasmattest.SaltLen {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "salt_hex must be 32 hex chars")
		return
	}
	wasmBytes := track.AttestWasm
	if len(wasmBytes) == 0 {
		httpresponse.Error(w, http.StatusServiceUnavailable, "UNAVAILABLE", "wasm module not embedded")
		return
	}
	sb, err := wasmattest.LoadSandbox(r.Context(), wasmBytes, wasmattest.DefaultLimits())
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	defer sb.Close(r.Context())

	nonce, err := sb.PoWSolve(r.Context(), salt, req.Difficulty, req.MaxTries)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	floatNoise, err := sb.FloatNoiseIEEE(r.Context())
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	shaEmpty, err := sb.SHA256OneShot(r.Context(), nil)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	benchAcc, err := sb.BenchMul(r.Context(), req.BenchRounds)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	resp := wasmAttestDryRunResponse{
		Nonce:          nonce,
		FloatNoiseHex:  hex.EncodeToString(floatNoise),
		BenchAcc:       benchAcc,
		Sha256EmptyHex: hex.EncodeToString(shaEmpty),
		ParityOK:       hex.EncodeToString(floatNoise) == wasmattest.FloatNoiseIEEEHex,
	}
	httpresponse.JSON(w, http.StatusOK, resp)
}
