package main

import (
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"time"

	"github.com/mattermost/mattermost-server/v5/model"
	"github.com/mattermost/mattermost-server/v5/plugin"
)

const (
	postFallbackMaxLength = 500
	postTextMaxLength     = 500
)

var EventToColor = map[string]string{
	"new_item":         "#ff0000", // red
	"occurrence":       "#ff0000", // red
	"reactivated_item": "#ffff00", // yellow
	"exp_repeat_item":  "#800080", // purple
	"item_velocity":    "#ffa500", // orange
	"reopened_item":    "#add8e6", // light blue
	"resolved_item":    "#00ff00", // green
	"deploy":           "#4bc6b9", // green-ish
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

	if invalidMsg := rollbar.isValid(); invalidMsg != "" {
		rollbarJSON, _ := json.Marshal(rollbar)
		p.API.LogWarn(fmt.Sprintf("Invalid rollbar webhook received: %s: %s", invalidMsg, rollbarJSON))
		http.Error(w, invalidMsg, http.StatusBadRequest)
		return
	}

	//get project title and timezone
	projectTitle := ""
	timezone := ""

	if configuration.RollbarApiKey != "" {
		var projectData ProjectData
		rollbarApiKey := configuration.RollbarApiKey
		projectID := rollbar.Data.Item.ProjectID
		//p.API.LogError("rollbarData=", rollbar)
		rollbarURL := "https://api.rollbar.com/api/1/project/" + strconv.Itoa(projectID)
		client := &http.Client{}
		req, err := http.NewRequest("GET", rollbarURL, nil)
		if err != nil {
			p.API.LogError("Error creating http request", "err:", err.Error())
		} else {
			req.Header.Add("X-Rollbar-Access-Token", rollbarApiKey)
			//fmt.Println(req)
			response, err := client.Do(req)
			if err != nil {
				p.API.LogError("Error sending request to rollbar", "err:", err.Error())
			} else {
				responseData, err := ioutil.ReadAll(response.Body)
				if err != nil {
					p.API.LogError("Error reading response body", "err:", err.Error())
				} else {
					if err := json.Unmarshal(responseData, &projectData); err != nil {
						p.API.LogError("Error marshaling responseData", "err:", err.Error())
					} else {
						projectTitle = projectData.Result.Name
						timezone = projectData.Result.SettingsData.Timezone
					}
				}
			}
		}
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
	title := rollbar.eventNameToTitle()

	// non-standard webhook events, i.e. different available data
	switch rollbar.EventName {
	case "item_velocity":

		environment := rollbar.Data.Item.Environment

		lastOccurrenceTime := ""

		if timezone != "" {
			lastOccurrenceTimestamp := rollbar.Data.Item.LastOccurrenceTimestamp
			if zone, err := time.LoadLocation(timezone); err != nil {
				p.API.LogError("Error parsing timezone", err.Error, "timezone:", timezone)
			} else {
				lastOccurrenceTime = time.Unix(int64(lastOccurrenceTimestamp), 0).In(zone).Format(time.ANSIC)
			}

		}

		fields := []*model.SlackAttachmentField{
			&model.SlackAttachmentField{
				Short: true,
				Title: "Project",
				Value: projectTitle,
			},
			&model.SlackAttachmentField{
				Short: true,
				Title: "Last Occurrence",
				Value: lastOccurrenceTime,
			},
			&model.SlackAttachmentField{
				Short: true,
				Title: "Environment",
				Value: environment,
			},
			&model.SlackAttachmentField{
				Short: true,
				Title: "Links",
				Value: fmt.Sprintf("[Occurrences](%s)", rollbar.Data.URL),
			},
		}

		title = "#" + strconv.Itoa(rollbar.Data.Item.Counter) + " " + title + ":"

		text := rollbar.Data.Item.Title
		attachment := &model.SlackAttachment{
			Color:     EventToColor[rollbar.EventName],
			Fallback:  title,
			Title:     fmt.Sprintf("%s    %s", title, TruncateString(text, postTextMaxLength)),
			Fields:    fields,
			TitleLink: rollbar.Data.URL,
		}

		if pretext != "None" {
			attachment.Pretext = pretext
		}

		post := &model.Post{
			ChannelId: channelId,
			UserId:    p.botUserID,
			Type:      model.POST_SLACK_ATTACHMENT,
			Props: map[string]interface{}{
				"attachments": []*model.SlackAttachment{attachment},
			},
		}

		if _, err := p.API.CreatePost(post); err != nil {
			p.API.LogError(fmt.Sprintf("Error creating a post: %s", err.Error()))
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	case "deploy":
		environment := rollbar.Data.Deploy.Environment
		text := fmt.Sprintf(
			"`%s` **%s** deployed `%s` revision `%s`",
			rollbar.deployDateTime(),
			rollbar.deployUser(),
			environment,
			rollbar.Data.Deploy.Revision,
		)
		fallback := fmt.Sprintf("[%s] %s - %s", title, environment, text)
		attachment := &model.SlackAttachment{
			Color:     EventToColor[rollbar.EventName],
			Fallback:  fallback,
			Title:     title,
			TitleLink: fmt.Sprintf("https://rollbar.com/deploy/%d/", rollbar.Data.Deploy.ID),
			Text:      text,
		}

		if pretext != "None" {
			attachment.Pretext = pretext
		}

		post := &model.Post{
			ChannelId: channelId,
			UserId:    p.botUserID,
			Type:      model.POST_SLACK_ATTACHMENT,
			Props: map[string]interface{}{
				"attachments": []*model.SlackAttachment{attachment},
			},
		}

		if _, err := p.API.CreatePost(post); err != nil {
			p.API.LogError(fmt.Sprintf("Error creating a post: %s", err.Error()))
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	case "test":
		post := &model.Post{
			ChannelId: channelId,
			UserId:    p.botUserID,
			Message:   rollbar.Data.Message,
		}
		if _, err := p.API.CreatePost(post); err != nil {
			p.API.LogError(fmt.Sprintf("Error creating a post: %s", err.Error()))
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	// standard webhook events
	lastOccurrence := rollbar.Data.Item.LastOccurrence
	// event type `occurrence` has data under `occurrence` instead of `last_occurrence`
	if lastOccurrence == nil {
		lastOccurrence = rollbar.Data.Occurrence
	}
	environment := rollbar.Data.Item.Environment
	language := lastOccurrence.Language
	itemLink := fmt.Sprintf(
		"https://rollbar.com/item/uuid/?uuid=%s",
		lastOccurrence.UUID)
	occurrenceLink := fmt.Sprintf(
		"https://rollbar.com/occurrence/uuid/?uuid=%s",
		lastOccurrence.UUID)

	eventText := rollbar.eventText()
	if eventText == "" {
		p.API.LogWarn(fmt.Sprintf("No %s exception message found. Link: %s", rollbar.EventName, rollbar.Data.URL))
		eventText = "No exception message found in Rollbar webhook. Check mattermost server logs for more info."
	}

	fallback := fmt.Sprintf("[%s] %s - %s", environment, title, TruncateString(eventText, postFallbackMaxLength))

	fields := []*model.SlackAttachmentField{
		&model.SlackAttachmentField{
			Short: true,
			Title: "Project",
			Value: projectTitle,
		},
		&model.SlackAttachmentField{
			Short: true,
			Title: "Environment",
			Value: environment,
		},
		&model.SlackAttachmentField{
			Short: true,
			Title: "Language",
			Value: language,
		},
		&model.SlackAttachmentField{
			Short: true,
			Title: "Links",
			Value: fmt.Sprintf("[Item](%s) | [Occurrence](%s)", itemLink, occurrenceLink),
		},
	}

	//edit title
	title = "#" + strconv.Itoa(rollbar.Data.Item.Counter) + " " + title + ":"

	attachment := &model.SlackAttachment{
		Color:     EventToColor[rollbar.EventName],
		Fallback:  fallback,
		Fields:    fields,
		Title:     fmt.Sprintf("%s  %s", title, TruncateString(eventText, postTextMaxLength)),
		TitleLink: itemLink,
	}

	if pretext != "None" {
		attachment.Pretext = pretext
	}

	post := &model.Post{
		ChannelId: channelId,
		UserId:    p.botUserID,
		Type:      model.POST_SLACK_ATTACHMENT,
		Props: map[string]interface{}{
			"attachments": []*model.SlackAttachment{attachment},
		},
	}

	if _, err := p.API.CreatePost(post); err != nil {
		p.API.LogError(fmt.Sprintf("Error creating a post: %s", err.Error()))
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}
