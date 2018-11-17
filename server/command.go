package main

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/mattermost/mattermost-server/model"
	"github.com/mattermost/mattermost-server/plugin"
)

const (
	commandTrigger      = "rollbar"
	responseUsername    = "Rollbar"
	genericErrorMessage = "Something went wrong. Check server logs or try again later."
	usageErrorMessage   = "Usage: `/rollbar (notify|remove|list) @username`"
	usersListMessage    = "Users notified on each Rollbar posted to this channel: %s"
)

func getCommand() *model.Command {
	return &model.Command{
		Trigger:          commandTrigger,
		DisplayName:      "Rollbar",
		Description:      "Rollbar notifications",
		AutoComplete:     true,
		AutoCompleteDesc: "Notify specific users in this channel",
		AutoCompleteHint: "(notify|remove|list) @username",
	}
}

func getCommandResponse(message string) *model.CommandResponse {
	return &model.CommandResponse{
		ResponseType: model.COMMAND_RESPONSE_TYPE_EPHEMERAL,
		Text:         message,
	}
}

// Converts the KVGet->map into a comma separated string list of @usernames
// Returns "None" for an empty map
func GetUsernameList(m map[string]bool) string {
	usernameList := make([]string, 0, len(m))
	for k := range m {
		usernameList = append(usernameList, fmt.Sprintf("@%s", k))
	}

	sort.Strings(usernameList)

	var usernameListString string

	if len(usernameList) > 0 {
		usernameListString = strings.Join(usernameList, ", ")
	} else {
		usernameListString = "None"
	}
	return usernameListString
}

// Log and return the generic error command response for any plugin or other
// API call errors, e.g. json marshalling
func (p *RollbarPlugin) returnGenericError(err error) *model.CommandResponse {
	p.API.LogError(err.Error())
	return getCommandResponse(genericErrorMessage)
}

// List, add, or remove usernames from the plugin's KV store for the channel
// where the command was called from.
func (p *RollbarPlugin) ExecuteCommand(c *plugin.Context, args *model.CommandArgs) (*model.CommandResponse, *model.AppError) {
	tokens := strings.Split(args.Command, " ")

	if len(tokens) < 2 || len(tokens) > 3 {
		return getCommandResponse(usageErrorMessage), nil
	}

	channelId := args.ChannelId
	existingUsers, err := p.API.KVGet(channelId)
	if err != nil {
		return p.returnGenericError(err), nil
	}

	// Users that are currently being notified
	usersMap := make(map[string]bool)
	if len(existingUsers) > 0 {
		if err := json.Unmarshal(existingUsers, &usersMap); err != nil {
			return p.returnGenericError(err), nil
		}
	}

	action := tokens[1]
	switch action {
	case "list":
		if len(tokens) != 2 {
			return getCommandResponse("Usage: `/rollbar list`"), nil
		}
		return getCommandResponse(fmt.Sprintf(usersListMessage, GetUsernameList(usersMap))), nil
	case "notify", "remove":
		if len(tokens) != 3 {
			return getCommandResponse(fmt.Sprintf("Usage: `/rollbar %s @username`", action)), nil
		}

		username := tokens[2]
		username = strings.TrimPrefix(username, "@")
		user, _ := p.API.GetUserByUsername(username)
		if user == nil {
			return getCommandResponse(fmt.Sprintf("User `%s` not found.", username)), nil
		}

		if action == "remove" {
			if usersMap[username] {
				delete(usersMap, username)
			} else {
				return getCommandResponse(fmt.Sprintf("User `%s` is already not being notified.", username)), nil
			}
		} else if action == "notify" {
			if !usersMap[username] {
				usersMap[username] = true
			} else {
				return getCommandResponse(fmt.Sprintf("User `%s` is already being notified.", username)), nil
			}
		}

		newValue, _ := json.Marshal(usersMap)
		if err := p.API.KVSet(channelId, newValue); err != nil {
			return p.returnGenericError(err), nil
		}

		return getCommandResponse(fmt.Sprintf(usersListMessage, GetUsernameList(usersMap))), nil
	default:
		return getCommandResponse(usageErrorMessage), nil
	}
}
