package api

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/DevMatrix/server/internal/auth"
	"github.com/DevMatrix/server/internal/db"
	"github.com/rs/zerolog/log"
)

// Handler exposes HTTP endpoints for auth, shop, loadout, and leaderboard.
type Handler struct {
	auth    *auth.Service
	queries *db.Queries
}

func NewHandler(authSvc *auth.Service, queries *db.Queries) *Handler {
	return &Handler{auth: authSvc, queries: queries}
}

// Register registers all API routes on the given mux.
func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/register", h.handleRegister)
	mux.HandleFunc("POST /api/login", h.handleLogin)
	mux.HandleFunc("GET /api/profile", h.withAuth(h.handleGetProfile))
	mux.HandleFunc("GET /api/shop/items", h.handleListItems)
	mux.HandleFunc("POST /api/shop/buy", h.withAuth(h.handleBuy))
	mux.HandleFunc("POST /api/loadout/equip", h.withAuth(h.handleEquip))
	mux.HandleFunc("GET /api/leaderboard", h.handleLeaderboard)
}

// withAuth wraps a handler requiring JWT auth. Injects claims into context.
func (h *Handler) withAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if !strings.HasPrefix(authHeader, "Bearer ") {
			http.Error(w, `{"error":"missing token"}`, http.StatusUnauthorized)
			return
		}
		claims, err := h.auth.ValidateToken(authHeader[7:])
		if err != nil {
			http.Error(w, `{"error":"invalid token"}`, http.StatusUnauthorized)
			return
		}
		r = r.WithContext(withClaims(r.Context(), claims))
		next(w, r)
	}
}

func (h *Handler) handleRegister(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Username string `json:"username"`
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	user, err := h.auth.Register(r.Context(), req.Username, req.Email, req.Password)
	if err != nil {
		jsonError(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Auto-login after registration.
	token, err := h.auth.Login(r.Context(), req.Email, req.Password)
	if err != nil {
		jsonError(w, "registered but login failed", http.StatusInternalServerError)
		return
	}

	jsonResp(w, http.StatusCreated, map[string]interface{}{
		"token":    token,
		"user_id":  user.ID.String(),
		"username": user.Username,
	})
}

func (h *Handler) handleLogin(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	token, err := h.auth.Login(r.Context(), req.Email, req.Password)
	if err != nil {
		jsonError(w, err.Error(), http.StatusUnauthorized)
		return
	}

	jsonResp(w, http.StatusOK, map[string]string{"token": token})
}

func (h *Handler) handleGetProfile(w http.ResponseWriter, r *http.Request) {
	claims := getClaims(r.Context())
	profile, err := h.queries.GetProfile(r.Context(), claims.UserID)
	if err != nil {
		jsonError(w, "profile not found", http.StatusNotFound)
		return
	}

	inv, err := h.queries.GetInventory(r.Context(), claims.UserID)
	if err != nil {
		inv = []string{}
	}

	jsonResp(w, http.StatusOK, map[string]interface{}{
		"username":  claims.Username,
		"coins":     profile.Coins,
		"kills":     profile.Kills,
		"deaths":    profile.Deaths,
		"ai_tier":   profile.AITier,
		"inventory": inv,
		"loadout": map[string]interface{}{
			"hull":             profile.HullID,
			"primary_weapon":   profile.PrimaryWeaponID,
			"secondary_weapon": profile.SecondaryWeaponID,
			"shield":           profile.ShieldID,
		},
	})
}

func (h *Handler) handleListItems(w http.ResponseWriter, r *http.Request) {
	items, err := h.queries.ListShopItems(r.Context())
	if err != nil {
		jsonError(w, "failed to load items", http.StatusInternalServerError)
		return
	}
	jsonResp(w, http.StatusOK, items)
}

func (h *Handler) handleBuy(w http.ResponseWriter, r *http.Request) {
	claims := getClaims(r.Context())
	var req struct {
		ItemID string `json:"item_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if err := h.queries.PurchaseItem(r.Context(), claims.UserID, req.ItemID); err != nil {
		jsonError(w, err.Error(), http.StatusBadRequest)
		return
	}

	profile, _ := h.queries.GetProfile(r.Context(), claims.UserID)
	coins := 0
	if profile != nil {
		coins = profile.Coins
	}
	jsonResp(w, http.StatusOK, map[string]interface{}{"ok": true, "coins": coins})
}

func (h *Handler) handleEquip(w http.ResponseWriter, r *http.Request) {
	claims := getClaims(r.Context())
	var req struct {
		ItemID string `json:"item_id"`
		Slot   string `json:"slot"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if err := h.queries.EquipItem(r.Context(), claims.UserID, req.ItemID, req.Slot); err != nil {
		jsonError(w, err.Error(), http.StatusBadRequest)
		return
	}
	jsonResp(w, http.StatusOK, map[string]bool{"ok": true})
}

func (h *Handler) handleLeaderboard(w http.ResponseWriter, r *http.Request) {
	entries, err := h.queries.GetLeaderboard(r.Context(), 20)
	if err != nil {
		log.Error().Err(err).Msg("leaderboard query failed")
		jsonError(w, "failed to load leaderboard", http.StatusInternalServerError)
		return
	}
	jsonResp(w, http.StatusOK, entries)
}

// --- helpers ---

type contextKey string

const claimsKey contextKey = "claims"

func withClaims(ctx context.Context, claims *auth.Claims) context.Context {
	return context.WithValue(ctx, claimsKey, claims)
}

func getClaims(ctx context.Context) *auth.Claims {
	return ctx.Value(claimsKey).(*auth.Claims)
}

func jsonResp(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func jsonError(w http.ResponseWriter, msg string, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
}
