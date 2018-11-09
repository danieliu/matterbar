package main

import (
	"crypto/subtle"
	"fmt"
	"net/http"

	"github.com/mattermost/mattermost-server/plugin"
)

func (p *Plugin) ServeHTTP(c *plugin.Context, w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	case "/notify":
		p.handleWebhook(w, r)
	default:
		http.NotFound(w, r)
	}
}

func (p *Plugin) handleWebhook(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed.", http.StatusMethodNotAllowed)
		return
	}

	configuration := p.getConfiguration()
	query := r.URL.Query()
	queryTeam := query.Get("team")
	queryChannel := query.Get("channel")

	if subtle.ConstantTimeCompare([]byte(query.Get("auth")), []byte(configuration.Secret)) != 1 {
		p.API.LogWarn("Unauthorized matterbar webhook request.")
		http.Error(w, "Unauthorized.", http.StatusUnauthorized)
		return
	}

	if configuration.teamId == "" && queryTeam == "" {
		p.API.LogWarn("Default team not configured; expected team name in query param.")
		http.Error(w, "Missing 'team' query parameter.", http.StatusBadRequest)
		return
	}

	if configuration.channelId == "" && queryChannel == "" {
		p.API.LogWarn("Default channel not configured; expected channel name in query param.")
		http.Error(w, "Missing 'channel' query parameter.", http.StatusBadRequest)
		return
	}

	var teamId string
	var channelId string

	// Use the query parameter team if it exists, else default to the config.
	if queryTeam == "" {
		teamId = configuration.teamId
	} else {
		team, _ := p.API.GetTeamByName(queryTeam)

		if team == nil {
			errorMessage := fmt.Sprintf("Team '%s' does not exist", queryTeam)
			p.API.LogWarn(errorMessage)
			http.Error(w, errorMessage, http.StatusBadRequest)
			return
		}

		teamId = team.Id
	}

	// Use the query parameter channel if it exists, else default to the config.
	if queryChannel == "" {
		channelId = configuration.channelId
	} else {
		channel, _ := p.API.GetChannelByName(teamId, queryChannel, false)

		if channel == nil {
			errorMessage := fmt.Sprintf("Channel '%s' not found.", queryChannel)
			p.API.LogWarn(errorMessage)
			http.Error(w, errorMessage, http.StatusBadRequest)
			return
		}

		channelId = channel.Id
	}

	// TODO: parse incoming rollbar post body
	// TODO: create mattermost post from rollbar data
}
