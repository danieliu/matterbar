package main

import (
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/mattermost/mattermost-server/model"
	"github.com/mattermost/mattermost-server/plugin"
)

var EventToColor = map[string]string{
	"new_item":         "#ff0000", // red
	"reactivated_item": "#ffff00", // yellow
	"exp_repeat_item":  "#800080", // purple
	"item_velocity":    "#ffa500", // orange
	"reopened_item":    "#add8e6", // light blue
	"resolved_item":    "#00ff00", // green
}

func (p *RollbarPlugin) ServeHTTP(c *plugin.Context, w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	case "/notify":
		p.handleWebhook(w, r)
	default:
		http.NotFound(w, r)
	}
}

func (p *RollbarPlugin) handleWebhook(w http.ResponseWriter, r *http.Request) {
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
		p.API.LogWarn("Unauthenticated matterbar webhook request.")
		http.Error(w, "Unauthenticated.", http.StatusUnauthorized)
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
			errorMessage := fmt.Sprintf("Team '%s' does not exist.", queryTeam)
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
			errorMessage := fmt.Sprintf("Channel '%s' does not exist.", queryChannel)
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

	title := rollbar.eventNameToTitle()
	lastOccurrence := rollbar.Data.Item.LastOccurrence
	environment := rollbar.Data.Item.Environment
	framework := lastOccurrence.Framework
	itemLink := fmt.Sprintf(
		"https://rollbar.com/item/uuid/?uuid=%s",
		lastOccurrence.Uuid)
	occurrenceLink := fmt.Sprintf(
		"https://rollbar.com/occurrence/uuid/?uuid=%s",
		lastOccurrence.Uuid)
	eventText := lastOccurrence.Body.Message.Body
	if eventText == "" {
		exceptionClass := lastOccurrence.Body.Trace.Exception.Class
		exceptionMessage := lastOccurrence.Body.Trace.Exception.Message
		eventText = fmt.Sprintf("%s: %s", exceptionClass, exceptionMessage)
	}

	fallback := fmt.Sprintf("[%s] %s - %s", environment, title, eventText)

	fields := []*model.SlackAttachmentField{
		&model.SlackAttachmentField{
			Short: true,
			Title: "Environment",
			Value: environment,
		},
		&model.SlackAttachmentField{
			Short: true,
			Title: "Framework",
			Value: framework,
		},
		&model.SlackAttachmentField{
			Short: true,
			Title: "Links",
			Value: fmt.Sprintf("[Item](%s) | [Occurrence](%s)", itemLink, occurrenceLink),
		},
	}

	usersToNotify, err := p.API.KVGet(channelId)
	if err != nil {
		p.API.LogWarn(fmt.Sprintf("Error fetching users to notify in channel %s", channelId))
	}
	usersMap := make(map[string]bool)
	if len(usersToNotify) > 0 {
		if err := json.Unmarshal(usersToNotify, &usersMap); err != nil {
			p.API.LogWarn(fmt.Sprintf("Error parsing users to notify: %s", err.Error()))
		}
	}
	pretext := GetUsernameList(usersMap)

	attachment := &model.SlackAttachment{
		Color:     EventToColor[rollbar.EventName],
		Fallback:  fallback,
		Fields:    fields,
		Title:     title,
		TitleLink: itemLink,
		Text:      fmt.Sprintf("```\n%s\n```", eventText),
	}

	if pretext != "None" {
		attachment.Pretext = pretext
	}

	post := &model.Post{
		ChannelId: channelId,
		UserId:    configuration.userId,
		Type:      model.POST_SLACK_ATTACHMENT,
		Props: map[string]interface{}{
			"from_webhook":  "true",
			"use_user_icon": "true",
			"attachments":   []*model.SlackAttachment{attachment},
		},
	}

	if _, err := p.API.CreatePost(post); err != nil {
		p.API.LogError(fmt.Sprintf("Error creating a post: %s", err.Error()))
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}
