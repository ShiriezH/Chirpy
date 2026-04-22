package main

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"os"
	"sync/atomic"
	"time"
	"sort"

	_ "github.com/lib/pq"
	"github.com/google/uuid"
	"github.com/joho/godotenv"

	"Chirpy/internal/auth"
	"Chirpy/internal/database"
)

type apiConfig struct {
	fileserverHits atomic.Int32
	dbQueries      *database.Queries
	platform       string
	jwtSecret      string
	polkaKey       string
}

// =========================
// USER RESPONSE HELPER
// =========================
func userResponse(user database.User) map[string]interface{} {
	return map[string]interface{}{
		"id":            user.ID.String(),
		"created_at":    user.CreatedAt,
		"updated_at":    user.UpdatedAt,
		"email":         user.Email,
		"is_chirpy_red": user.IsChirpyRed,
	}
}

// =========================
// ADMIN RESET
// =========================
func (cfg *apiConfig) handlerReset(w http.ResponseWriter, r *http.Request) {
	if cfg.platform != "dev" {
		w.WriteHeader(403)
		return
	}
	cfg.dbQueries.DeleteAllUsers(r.Context())
	w.WriteHeader(200)
}

// =========================
// CREATE USER
// =========================
func (cfg *apiConfig) handlerCreateUser(w http.ResponseWriter, r *http.Request) {
	type params struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	var p params
	json.NewDecoder(r.Body).Decode(&p)

	hash, _ := auth.HashPassword(p.Password)

	user, err := cfg.dbQueries.CreateUser(r.Context(), database.CreateUserParams{
		Email:          p.Email,
		HashedPassword: hash,
	})
	if err != nil {
		w.WriteHeader(500)
		return
	}

	w.WriteHeader(201)
	json.NewEncoder(w).Encode(userResponse(user))
}

// =========================
// UPDATE USER
// =========================
func (cfg *apiConfig) handlerUpdateUser(w http.ResponseWriter, r *http.Request) {
	tokenStr, err := auth.GetBearerToken(r.Header)
	if err != nil {
		w.WriteHeader(401)
		return
	}

	userID, err := auth.ValidateJWT(tokenStr, cfg.jwtSecret)
	if err != nil {
		w.WriteHeader(401)
		return
	}

	type params struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	var p params
	json.NewDecoder(r.Body).Decode(&p)

	hash, _ := auth.HashPassword(p.Password)

	user, err := cfg.dbQueries.UpdateUser(r.Context(), database.UpdateUserParams{
		ID:             userID,
		Email:          p.Email,
		HashedPassword: hash,
	})
	if err != nil {
		w.WriteHeader(500)
		return
	}

	json.NewEncoder(w).Encode(userResponse(user))
}

// =========================
// LOGIN
// =========================
func (cfg *apiConfig) handlerLogin(w http.ResponseWriter, r *http.Request) {
	type params struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	var p params
	json.NewDecoder(r.Body).Decode(&p)

	user, err := cfg.dbQueries.GetUserByEmail(r.Context(), p.Email)
	if err != nil {
		w.WriteHeader(401)
		return
	}

	ok, _ := auth.CheckPasswordHash(p.Password, user.HashedPassword)
	if !ok {
		w.WriteHeader(401)
		return
	}

	token, _ := auth.MakeJWT(user.ID, cfg.jwtSecret, time.Hour)
	refresh := auth.MakeRefreshToken()

	cfg.dbQueries.CreateRefreshToken(r.Context(), database.CreateRefreshTokenParams{
		Token:     refresh,
		UserID:    user.ID,
		ExpiresAt: time.Now().Add(60 * 24 * time.Hour),
	})

	resp := userResponse(user)
	resp["token"] = token
	resp["refresh_token"] = refresh

	json.NewEncoder(w).Encode(resp)
}

// =========================
// CREATE CHIRP
// =========================
func (cfg *apiConfig) handlerCreateChirp(w http.ResponseWriter, r *http.Request) {
	tokenStr, err := auth.GetBearerToken(r.Header)
	if err != nil {
		w.WriteHeader(401)
		return
	}

	userID, err := auth.ValidateJWT(tokenStr, cfg.jwtSecret)
	if err != nil {
		w.WriteHeader(401)
		return
	}

	type params struct {
		Body string `json:"body"`
	}

	var p params
	json.NewDecoder(r.Body).Decode(&p)

	if len(p.Body) > 140 {
		w.WriteHeader(400)
		return
	}

	chirp, err := cfg.dbQueries.CreateChirp(r.Context(), database.CreateChirpParams{
		Body:   p.Body,
		UserID: userID,
	})
	if err != nil {
		w.WriteHeader(500)
		return
	}

	w.WriteHeader(201)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"id":         chirp.ID.String(),
		"created_at": chirp.CreatedAt,
		"updated_at": chirp.UpdatedAt,
		"body":       chirp.Body,
		"user_id":    chirp.UserID.String(),
	})
}

// =========================
// GET ALL CHIRPS (WITH FILTER)
// =========================
func (cfg *apiConfig) handlerGetChirps(w http.ResponseWriter, r *http.Request) {
	authorIDStr := r.URL.Query().Get("author_id")
	sortParam := r.URL.Query().Get("sort")

	var chirps []database.Chirp
	var err error

	// Filter by author (if provided)
	if authorIDStr != "" {
		authorID, parseErr := uuid.Parse(authorIDStr)
		if parseErr != nil {
			w.WriteHeader(400)
			return
		}
		chirps, err = cfg.dbQueries.GetChirpsByAuthor(r.Context(), authorID)
	} else {
		chirps, err = cfg.dbQueries.GetChirps(r.Context())
	}

	if err != nil {
		w.WriteHeader(500)
		return
	}

	// Default = ascending
	if sortParam == "desc" {
		sort.Slice(chirps, func(i, j int) bool {
			return chirps[i].CreatedAt.After(chirps[j].CreatedAt)
		})
	} else {
		// asc OR empty
		sort.Slice(chirps, func(i, j int) bool {
			return chirps[i].CreatedAt.Before(chirps[j].CreatedAt)
		})
	}

	// Build response
	var resp []map[string]interface{}
	for _, c := range chirps {
		resp = append(resp, map[string]interface{}{
			"id":         c.ID.String(),
			"created_at": c.CreatedAt,
			"updated_at": c.UpdatedAt,
			"body":       c.Body,
			"user_id":    c.UserID.String(),
		})
	}

	w.WriteHeader(200)
	json.NewEncoder(w).Encode(resp)
}

// =========================
// GET SINGLE CHIRP
// =========================
func (cfg *apiConfig) handlerGetChirp(w http.ResponseWriter, r *http.Request) {
	chirpID, _ := uuid.Parse(r.PathValue("chirpID"))

	chirp, err := cfg.dbQueries.GetChirp(r.Context(), chirpID)
	if err != nil {
		w.WriteHeader(404)
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"id":         chirp.ID.String(),
		"created_at": chirp.CreatedAt,
		"updated_at": chirp.UpdatedAt,
		"body":       chirp.Body,
		"user_id":    chirp.UserID.String(),
	})
}

// =========================
// DELETE CHIRP
// =========================
func (cfg *apiConfig) handlerDeleteChirp(w http.ResponseWriter, r *http.Request) {
	tokenStr, err := auth.GetBearerToken(r.Header)
	if err != nil {
		w.WriteHeader(401)
		return
	}

	userID, err := auth.ValidateJWT(tokenStr, cfg.jwtSecret)
	if err != nil {
		w.WriteHeader(401)
		return
	}

	chirpID, _ := uuid.Parse(r.PathValue("chirpID"))

	chirp, err := cfg.dbQueries.GetChirp(r.Context(), chirpID)
	if err != nil {
		w.WriteHeader(404)
		return
	}

	if chirp.UserID != userID {
		w.WriteHeader(403)
		return
	}

	cfg.dbQueries.DeleteChirp(r.Context(), chirpID)
	w.WriteHeader(204)
}

// =========================
// WEBHOOK (SECURED)
// =========================
func (cfg *apiConfig) handlerPolkaWebhooks(w http.ResponseWriter, r *http.Request) {
	apiKey, err := auth.GetAPIKey(r.Header)
	if err != nil || apiKey != cfg.polkaKey {
		w.WriteHeader(401)
		return
	}

	type req struct {
		Event string `json:"event"`
		Data  struct {
			UserID string `json:"user_id"`
		} `json:"data"`
	}

	var body req
	json.NewDecoder(r.Body).Decode(&body)

	if body.Event != "user.upgraded" {
		w.WriteHeader(204)
		return
	}

	userID, err := uuid.Parse(body.Data.UserID)
	if err != nil {
		w.WriteHeader(400)
		return
	}

	err = cfg.dbQueries.UpgradeUserToChirpyRed(r.Context(), userID)
	if err != nil {
		w.WriteHeader(404)
		return
	}

	w.WriteHeader(204)
}

// =========================
// MAIN
// =========================
func main() {
	godotenv.Load()

	db, _ := sql.Open("postgres", os.Getenv("DB_URL"))

	apiCfg := &apiConfig{
		dbQueries: database.New(db),
		jwtSecret: os.Getenv("JWT_SECRET"),
		polkaKey:  os.Getenv("POLKA_KEY"),
		platform:  os.Getenv("PLATFORM"),
	}

	mux := http.NewServeMux()

	mux.HandleFunc("/admin/reset", apiCfg.handlerReset)

	mux.HandleFunc("/api/users", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			apiCfg.handlerCreateUser(w, r)
		} else if r.Method == http.MethodPut {
			apiCfg.handlerUpdateUser(w, r)
		} else {
			w.WriteHeader(405)
		}
	})

	mux.HandleFunc("/api/login", apiCfg.handlerLogin)

	mux.HandleFunc("/api/chirps", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			apiCfg.handlerCreateChirp(w, r)
		} else if r.Method == http.MethodGet {
			apiCfg.handlerGetChirps(w, r)
		} else {
			w.WriteHeader(405)
		}
	})

	mux.HandleFunc("/api/chirps/{chirpID}", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			apiCfg.handlerGetChirp(w, r)
		} else if r.Method == http.MethodDelete {
			apiCfg.handlerDeleteChirp(w, r)
		} else {
			w.WriteHeader(405)
		}
	})

	mux.HandleFunc("/api/polka/webhooks", apiCfg.handlerPolkaWebhooks)

	http.ListenAndServe(":8080", mux)
}