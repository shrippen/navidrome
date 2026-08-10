package nativeapi

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/core/playback"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model/request"
)

func (api *Router) addSendspinRoute(r chi.Router) {
	r.Route("/sendspin", func(r chi.Router) {
		r.Use(sendspinAccessMiddleware)
		r.Get("/", api.getSendspinStatus)
		r.Post("/command", api.postSendspinCommand)
		r.Post("/gain", api.postSendspinGain)
	})
}

func sendspinAccessMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !playback.SendspinUIEnabled() {
			http.Error(w, "Sendspin jukebox is not enabled", http.StatusNotFound)
			return
		}
		user, ok := request.UserFrom(r.Context())
		if !ok {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		if conf.Server.Jukebox.AdminOnly && !user.IsAdmin {
			http.Error(w, "Jukebox is admin only", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (api *Router) playbackServer() playback.PlaybackServer {
	return playback.GetInstance(api.ds)
}

func (api *Router) getSendspinStatus(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	status, err := api.playbackServer().GetSendspinStatus(ctx)
	if err != nil {
		log.Error(ctx, "Error getting Sendspin status", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(status); err != nil {
		log.Error(ctx, "Error encoding Sendspin status", err)
	}
}

type sendspinCommandRequest struct {
	Command string `json:"command"`
}

func (api *Router) postSendspinCommand(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var req sendspinCommandRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Command == "" {
		http.Error(w, "missing command", http.StatusBadRequest)
		return
	}
	if user, ok := request.UserFrom(ctx); ok {
		if _, err := api.playbackServer().GetDeviceForUser(user.UserName); err != nil {
			log.Debug(ctx, "Could not bind jukebox user for Sendspin command", err)
		}
	}
	if err := api.playbackServer().SendspinCommand(ctx, req.Command); err != nil {
		log.Error(ctx, "Error running Sendspin command", "command", req.Command, err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	api.getSendspinStatus(w, r)
}

type sendspinGainRequest struct {
	Gain float32 `json:"gain"`
}

func (api *Router) postSendspinGain(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var req sendspinGainRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid gain", http.StatusBadRequest)
		return
	}
	if user, ok := request.UserFrom(ctx); ok {
		if _, err := api.playbackServer().GetDeviceForUser(user.UserName); err != nil {
			log.Debug(ctx, "Could not bind jukebox user for Sendspin gain", err)
		}
	}
	if err := api.playbackServer().SetSendspinGain(ctx, req.Gain); err != nil {
		log.Error(ctx, "Error setting Sendspin gain", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	api.getSendspinStatus(w, r)
}
