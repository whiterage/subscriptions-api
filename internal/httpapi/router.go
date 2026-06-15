package httpapi

import (
	"crypto/subtle"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"time"

	"effective-mobile-subscriptions/internal/service"
)

const maxJSONBodyBytes = 1 << 20

type Handler struct {
	service *service.Service
	logger  *slog.Logger
	apiKey  string
}

func NewRouter(service *service.Service, logger *slog.Logger, apiKey string) http.Handler {
	h := &Handler{service: service, logger: logger, apiKey: apiKey}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", h.health)
	mux.HandleFunc("GET /swagger/doc.yaml", h.swagger)
	mux.Handle("POST /subscriptions", h.requireAPIKey(http.HandlerFunc(h.createSubscription)))
	mux.Handle("GET /subscriptions", h.requireAPIKey(http.HandlerFunc(h.listSubscriptions)))
	mux.Handle("GET /subscriptions/total", h.requireAPIKey(http.HandlerFunc(h.totalSubscriptions)))
	mux.Handle("GET /subscriptions/{id}", h.requireAPIKey(http.HandlerFunc(h.getSubscription)))
	mux.Handle("PUT /subscriptions/{id}", h.requireAPIKey(http.HandlerFunc(h.updateSubscription)))
	mux.Handle("DELETE /subscriptions/{id}", h.requireAPIKey(http.HandlerFunc(h.deleteSubscription)))

	return h.recover(h.logRequests(mux))
}

func (h *Handler) health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *Handler) swagger(w http.ResponseWriter, r *http.Request) {
	data, err := os.ReadFile("docs/swagger.yaml")
	if err != nil {
		h.writeError(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/yaml; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data)
}

func (h *Handler) createSubscription(w http.ResponseWriter, r *http.Request) {
	var input service.CreateSubscription
	if err := readJSON(w, r, &input); err != nil {
		h.writeError(w, err)
		return
	}

	item, err := h.service.Create(r.Context(), input)
	if err != nil {
		h.writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, item)
}

func (h *Handler) listSubscriptions(w http.ResponseWriter, r *http.Request) {
	filter := service.ListFilter{
		UserID:      r.URL.Query().Get("user_id"),
		ServiceName: r.URL.Query().Get("service_name"),
		Limit:       parseInt(r.URL.Query().Get("limit"), 50),
		Offset:      parseInt(r.URL.Query().Get("offset"), 0),
	}

	items, err := h.service.List(r.Context(), filter)
	if err != nil {
		h.writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (h *Handler) getSubscription(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		h.writeError(w, err)
		return
	}

	item, err := h.service.Get(r.Context(), id)
	if err != nil {
		h.writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (h *Handler) updateSubscription(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		h.writeError(w, err)
		return
	}

	var input service.UpdateSubscription
	if err := readJSON(w, r, &input); err != nil {
		h.writeError(w, err)
		return
	}

	item, err := h.service.Update(r.Context(), id, input)
	if err != nil {
		h.writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (h *Handler) deleteSubscription(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		h.writeError(w, err)
		return
	}

	if err := h.service.Delete(r.Context(), id); err != nil {
		h.writeError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) totalSubscriptions(w http.ResponseWriter, r *http.Request) {
	from, err := service.ParseMonth(r.URL.Query().Get("from"))
	if err != nil {
		h.writeError(w, err)
		return
	}
	to, err := service.ParseMonth(r.URL.Query().Get("to"))
	if err != nil {
		h.writeError(w, err)
		return
	}

	total, err := h.service.Total(r.Context(), service.TotalFilter{
		From:        from,
		To:          to,
		UserID:      r.URL.Query().Get("user_id"),
		ServiceName: r.URL.Query().Get("service_name"),
	})
	if err != nil {
		h.writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]int64{"total": total})
}

func (h *Handler) writeError(w http.ResponseWriter, err error) {
	status := http.StatusInternalServerError
	message := "internal server error"
	if errors.Is(err, service.ErrInvalidInput) {
		status = http.StatusBadRequest
		message = err.Error()
	}
	if errors.Is(err, service.ErrNotFound) {
		status = http.StatusNotFound
		message = err.Error()
	}

	if status >= 500 {
		h.logger.Error("request failed", slog.String("error", err.Error()))
	}
	writeJSON(w, status, map[string]string{"error": message})
}

func (h *Handler) requireAPIKey(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if subtle.ConstantTimeCompare([]byte(r.Header.Get("X-API-Key")), []byte(h.apiKey)) != 1 {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid or missing API key"})
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (h *Handler) logRequests(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rec, r)
		h.logger.Info("http request",
			slog.String("method", r.Method),
			slog.String("path", r.URL.Path),
			slog.Int("status", rec.status),
			slog.Duration("duration", time.Since(start)),
		)
	})
}

func (h *Handler) recover(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if panicValue := recover(); panicValue != nil {
				h.logger.Error("panic recovered", slog.Any("panic", panicValue))
				writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
			}
		}()
		next.ServeHTTP(w, r)
	})
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func readJSON(w http.ResponseWriter, r *http.Request, dest any) error {
	r.Body = http.MaxBytesReader(w, r.Body, maxJSONBodyBytes)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(dest); err != nil {
		var maxBytesError *http.MaxBytesError
		if errors.As(err, &maxBytesError) {
			return fmt.Errorf("%w: JSON body must not exceed %d bytes", service.ErrInvalidInput, maxJSONBodyBytes)
		}
		if errors.Is(err, http.ErrBodyReadAfterClose) || errors.Is(err, io.ErrUnexpectedEOF) {
			return fmt.Errorf("%w: malformed JSON body", service.ErrInvalidInput)
		}
		return errors.Join(service.ErrInvalidInput, err)
	}
	return nil
}

func parseID(r *http.Request) (int64, error) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id <= 0 {
		return 0, errors.Join(service.ErrInvalidInput, errors.New("id must be positive integer"))
	}
	return id, nil
}

func parseInt(value string, fallback int) int {
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}
