package main

import (
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/mattermost/mattermost-server/model"
	"github.com/mattermost/mattermost-server/plugin"
)

const DefaultTemplate = "{{ .EventName }} - [{{ .Data.Item.Environment }}] - {{ .Data.Item.LastOccurrence.Level }}"

var EventToColor = map[string]string{
	"reactivated_item": "#ffff00",
	"new_item":         "#ff0000",
	"exp_repeat_item":  "#800080",
	"item_velocity":    "#ffa500",
}

func (p *Plugin) ServeHTTP(c *plugin.Context, w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	case "/notify":
		p.handleWebhook(w, r)
	default:
		http.NotFound(w, r)
	}
}

func (p *Plugin) handleWebhook(w http.ResponseWriter, r *http.Request) {
	// TODO: Clean up / refactor validation

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

	var rollbar Rollbar
	if err := json.NewDecoder(r.Body).Decode(&rollbar); err != nil {
		p.API.LogError(fmt.Sprintf("Error in json decoding webhook: %s", err.Error()))
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	text, err := rollbar.interpolateMessage(DefaultTemplate)
	if err != nil {
		p.API.LogError(fmt.Sprintf("Error interpolating Rollbar message: %s", err.Error()))
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	eventMessage := rollbar.Data.Item.LastOccurrence.Body.Message.Body
	if eventMessage == "" {
		eventMessage = rollbar.Data.Item.LastOccurrence.Body.Trace.Exception.Message
	}

	// TODO: Rollbar custom webhooks payload seem to provide neither a link
	// nor the necessary data (user/organization, project name) to build out
	// a direct link back to the item or specific occurrence.
	// link := fmt.Sprintf("https://rollbar.com/<org>/<project>/items/%d", rollbar.Data.Item.Counter)

	fallback := fmt.Sprintf(
		"%s [%s] - %s",
		rollbar.EventName,
		rollbar.Data.Item.LastOccurrence.Environment,
		eventMessage)

	if _, err := p.API.CreatePost(&model.Post{
		ChannelId: channelId,
		UserId:    configuration.userId,
		Type:      model.POST_SLACK_ATTACHMENT,
		Props: map[string]interface{}{
			"from_webhook":  "true",
			"use_user_icon": "true",
			"attachments": []*model.SlackAttachment{
				&model.SlackAttachment{
					Color:    EventToColor[rollbar.EventName],
					Fallback: fallback,
					Title:    eventMessage,
					// TitleLink: link,
					Text: text,
				},
			},
		},
	}); err != nil {
		p.API.LogError(fmt.Sprintf("Error creating a post: %s", err.Error()))
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}
