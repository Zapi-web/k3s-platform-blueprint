package server

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/url"

	"github.com/Zapi-web/url-shortener/internal/domain"
)

type Handlers struct {
	shortener Shortener
}

func NewHandlers(shortener Shortener) *Handlers {
	return &Handlers{
		shortener: shortener,
	}
}

func (h *Handlers) ServeHealthz(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("OK"))
}

func (h *Handlers) ServeSaveUrl(w http.ResponseWriter, r *http.Request) {
	var req saveRequest

	r.Body = http.MaxBytesReader(w, r.Body, 1024*10)
	err := json.NewDecoder(r.Body).Decode(&req)

	if err != nil {
		slog.DebugContext(r.Context(), "failed to decode request body", "err", err)
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	if req.LongURL == "" {
		slog.DebugContext(r.Context(), "empty long_url provided in payload", "user_id", req.UserID)
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	parsedURL, err := url.ParseRequestURI(req.LongURL)
	if err != nil || (parsedURL.Scheme != "http" && parsedURL.Scheme != "https") || parsedURL.Host == "" {
		http.Error(w, "invalid url format", http.StatusBadRequest)
		return
	}

	shortURL, err := h.shortener.Create(r.Context(), req.LongURL, req.UserID)

	if err != nil {
		if errors.Is(err, domain.ErrInvalidInput) || errors.Is(err, domain.ErrInputisEmpty) {
			http.Error(w, "invalid input", http.StatusBadRequest)
			return
		}
		slog.ErrorContext(r.Context(), "failed to create short url", "user_id", req.UserID, "err", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	res := saveResponse{
		ShortURL: shortURL,
	}

	if err := json.NewEncoder(w).Encode(res); err != nil {
		slog.ErrorContext(r.Context(), "failed to encode save response", "short_url", shortURL, "err", err)
	}
}

func (h *Handlers) ServeGetURL(w http.ResponseWriter, r *http.Request) {
	shortURL := r.PathValue("id")

	if shortURL == "" {
		slog.DebugContext(r.Context(), "missing short_url parameter in request path")
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	longURL, err := h.shortener.Get(r.Context(), shortURL)

	if err != nil {
		switch {
		case errors.Is(err, domain.ErrUrlNotFound):
			slog.DebugContext(r.Context(), "requested url not found", "short_url", shortURL)
			http.Error(w, "not found", http.StatusNotFound)
			return
		case errors.Is(err, domain.ErrInvalidInput):
			http.Error(w, "invalid input", http.StatusBadRequest)
		}

		slog.ErrorContext(r.Context(), "failed to retrieve url", "short_url", shortURL, "err", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, longURL, http.StatusFound)
}
