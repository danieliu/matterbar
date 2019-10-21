package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/mattermost/mattermost-server/model"
	"github.com/mattermost/mattermost-server/plugin/plugintest"
	"github.com/mattermost/mattermost-server/plugin/plugintest/mock"
)

func loadJsonFile(t *testing.T, name string) []byte {
	path := filepath.Join("testdata", name)

	bytes, err := ioutil.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return bytes
}

func generatePost(channel string, user string, attachments []*model.SlackAttachment) *model.Post {
	return &model.Post{
		ChannelId: channel,
		UserId:    user,
		Type:      model.POST_SLACK_ATTACHMENT,
		Props: map[string]interface{}{
			"attachments": attachments,
		},
	}
}

func generateAttachmentFields(environment string, framework string, language string, uuid string) []*model.SlackAttachmentField {
	itemLink := fmt.Sprintf("https://rollbar.com/item/uuid/?uuid=%s", uuid)
	occurrenceLink := fmt.Sprintf("https://rollbar.com/occurrence/uuid/?uuid=%s", uuid)

	return []*model.SlackAttachmentField{
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
			Title: "Language",
			Value: language,
		},
		&model.SlackAttachmentField{
			Short: true,
			Title: "Links",
			Value: fmt.Sprintf("[Item](%s) | [Occurrence](%s)", itemLink, occurrenceLink),
		},
	}
}

func TestServeHttp(t *testing.T) {
	botUserID := "botUserID"
	emptyBody := []byte("{}")
	itemURL := "https://rollbar.com/organization/project/items/%s/"
	newItemUUID := "2e7cbf0a-a3af-402a-ab4f-95e07e5982f8"
	deployLink := "https://rollbar.com/deploy/%d/"
	itemLink := "https://rollbar.com/item/uuid/?uuid=%s"

	environment := "live"
	framework := "flask"
	language := "python 2.7.14"

	deployAttachment := []*model.SlackAttachment{
		&model.SlackAttachment{
			Color:     "#4bc6b9",
			Fallback:  "[Deploy] live - `2019-10-20 12:45:58 PDT-0700` **dliu** deployed `live` revision `780097be05cccf3e30ef3f90ad0c4cf9a085be22`",
			Pretext:   "@daniel, @eric",
			Title:     "Deploy",
			TitleLink: fmt.Sprintf(deployLink, 13752228),
			Text:      "`2019-10-20 12:45:58 PDT-0700` **dliu** deployed `live` revision `780097be05cccf3e30ef3f90ad0c4cf9a085be22`",
		},
	}
	deployAttachmentWithNoUsername := []*model.SlackAttachment{
		&model.SlackAttachment{
			Color:     "#4bc6b9",
			Fallback:  "[Deploy] live - `2019-10-20 12:45:58 PDT-0700` **unknown user** deployed `live` revision `780097be05cccf3e30ef3f90ad0c4cf9a085be22`",
			Title:     "Deploy",
			TitleLink: fmt.Sprintf(deployLink, 13835549),
			Text:      "`2019-10-20 12:45:58 PDT-0700` **unknown user** deployed `live` revision `780097be05cccf3e30ef3f90ad0c4cf9a085be22`",
		},
	}
	expRepeatAttachmentWithTraceChain := []*model.SlackAttachment{
		&model.SlackAttachment{
			Color:     "#800080",
			Fallback:  "[live] 10th Error - Exception: foo",
			Fields:    generateAttachmentFields(environment, framework, language, newItemUUID),
			Title:     "10th Error",
			TitleLink: fmt.Sprintf(itemLink, newItemUUID),
			Text:      "```\nException: foo\n```",
		},
	}
	itemVelocityAttachment := []*model.SlackAttachment{
		&model.SlackAttachment{
			Color:     "#ffa500",
			Fallback:  "5 occurrences in 5 minutes",
			Title:     "5 occurrences in 5 minutes",
			TitleLink: fmt.Sprintf(itemURL, "12343"),
			Text:      "```\nNo details available. High occurrence rate rollbar events are minimally supported.\n```",
		},
	}
	itemVelocityAttachmentWithNotify := []*model.SlackAttachment{
		&model.SlackAttachment{
			Color:     "#ffa500",
			Fallback:  "5 occurrences in 5 minutes",
			Pretext:   "@daniel, @eric",
			Title:     "5 occurrences in 5 minutes",
			TitleLink: fmt.Sprintf(itemURL, "12343"),
			Text:      "```\nNo details available. High occurrence rate rollbar events are minimally supported.\n```",
		},
	}
	newItemAttachment := []*model.SlackAttachment{
		&model.SlackAttachment{
			Color:     "#ff0000",
			Fallback:  "[live] New Error - TypeError: unsupported operand type(s) for +=: 'int' and 'str'",
			Fields:    generateAttachmentFields(environment, framework, language, newItemUUID),
			Title:     "New Error",
			TitleLink: fmt.Sprintf(itemLink, newItemUUID),
			Text:      "```\nTypeError: unsupported operand type(s) for +=: 'int' and 'str'\n```",
		},
	}
	newItemAttachmentWithNotify := []*model.SlackAttachment{
		&model.SlackAttachment{
			Color:     "#ff0000",
			Fallback:  "[live] New Error - TypeError: unsupported operand type(s) for +=: 'int' and 'str'",
			Fields:    generateAttachmentFields(environment, framework, language, newItemUUID),
			Pretext:   "@daniel, @eric",
			Title:     "New Error",
			TitleLink: fmt.Sprintf(itemLink, newItemUUID),
			Text:      "```\nTypeError: unsupported operand type(s) for +=: 'int' and 'str'\n```",
		},
	}
	newItemAttachmentFromJava := []*model.SlackAttachment{
		&model.SlackAttachment{
			Color:     "#ff0000",
			Fallback:  "[live] New Error - This is an error message",
			Fields:    generateAttachmentFields("live", "unknown", "java", "335e1144242f402ba6ed6f34065bbefd"),
			Title:     "New Error",
			TitleLink: fmt.Sprintf(itemLink, "335e1144242f402ba6ed6f34065bbefd"),
			Text:      "```\nThis is an error message\n```",
		},
	}
	newItemAttachmentWithLogMessage := []*model.SlackAttachment{
		&model.SlackAttachment{
			Color:     "#ff0000",
			Fallback:  "[live] New Error - User 8563892 is missing permissions",
			Fields:    generateAttachmentFields(environment, framework, language, "6fd2252b-3acd-4390-b722-466f5cbbd737"),
			Title:     "New Error",
			TitleLink: fmt.Sprintf(itemLink, "6fd2252b-3acd-4390-b722-466f5cbbd737"),
			Text:      "```\nUser 8563892 is missing permissions\n```",
		},
	}
	occurrenceAttachment := []*model.SlackAttachment{
		&model.SlackAttachment{
			Color:     "#ff0000",
			Fallback:  "[live] Occurrence - Error - TypeError: 'NoneType' object has no attribute '__getitem__'",
			Fields:    generateAttachmentFields(environment, framework, language, newItemUUID),
			Title:     "Occurrence - Error",
			TitleLink: fmt.Sprintf(itemLink, newItemUUID),
			Text:      "```\nTypeError: 'NoneType' object has no attribute '__getitem__'\n```",
		},
	}
	occurrenceAttachmentWithTraceChain := []*model.SlackAttachment{
		&model.SlackAttachment{
			Color:     "#ff0000",
			Fallback:  "[live] Occurrence - Error - Exception: foo",
			Fields:    generateAttachmentFields(environment, framework, language, "10aafec4-6dd8-40aa-9258-9635cfaf672c"),
			Title:     "Occurrence - Error",
			TitleLink: fmt.Sprintf(itemLink, "10aafec4-6dd8-40aa-9258-9635cfaf672c"),
			Text:      "```\nException: foo\n```",
		},
	}
	reactivatedAttachment := []*model.SlackAttachment{
		&model.SlackAttachment{
			Color:     "#ffff00",
			Fallback:  "[js-sandbox] Reactivated Error - TypeError: Assignment to constant variable.",
			Fields:    generateAttachmentFields("js-sandbox", "browser-js", "javascript", "271f5ed4-3cbe-4d3a-d4d6-b37a550c7b45"),
			Title:     "Reactivated Error",
			TitleLink: fmt.Sprintf(itemLink, "271f5ed4-3cbe-4d3a-d4d6-b37a550c7b45"),
			Text:      "```\nTypeError: Assignment to constant variable.\n```",
		},
	}
	reopenedAttachment := []*model.SlackAttachment{
		&model.SlackAttachment{
			Color:     "#add8e6",
			Fallback:  "[live] Reopened Error - TypeError: 'NoneType' object has no attribute '__getitem__'",
			Fields:    generateAttachmentFields(environment, framework, language, "16812560-c102-49e7-8fbf-39713c0014b4"),
			Title:     "Reopened Error",
			TitleLink: fmt.Sprintf(itemLink, "16812560-c102-49e7-8fbf-39713c0014b4"),
			Text:      "```\nTypeError: 'NoneType' object has no attribute '__getitem__'\n```",
		},
	}
	resolvedAttachment := []*model.SlackAttachment{
		&model.SlackAttachment{
			Color:     "#00ff00",
			Fallback:  "[live] Resolved Error - TypeError: 'NoneType' object has no attribute '__getitem__'",
			Fields:    generateAttachmentFields(environment, framework, language, "16812560-c102-49e7-8fbf-39713c0014b4"),
			Title:     "Resolved Error",
			TitleLink: fmt.Sprintf(itemLink, "16812560-c102-49e7-8fbf-39713c0014b4"),
			Text:      "```\nTypeError: 'NoneType' object has no attribute '__getitem__'\n```",
		},
	}

	deployPost := generatePost("channelId", botUserID, deployAttachment)
	deployPostWithNoUsername := generatePost("channelId", botUserID, deployAttachmentWithNoUsername)
	expRepeatPostWithTraceChain := generatePost("channelId", botUserID, expRepeatAttachmentWithTraceChain)
	itemVelocityPost := generatePost("channelId", botUserID, itemVelocityAttachment)
	itemVelocityPostWithNotify := generatePost("channelId", botUserID, itemVelocityAttachmentWithNotify)
	newItemPostWithChannelOverride := generatePost("existingChannelId", botUserID, newItemAttachment)
	newItemPost := generatePost("channelId", botUserID, newItemAttachment)
	newItemPostWithNotify := generatePost("channelId", botUserID, newItemAttachmentWithNotify)
	newItemPostFromJava := generatePost("channelId", botUserID, newItemAttachmentFromJava)
	newItemPostWithLogMessage := generatePost("channelId", botUserID, newItemAttachmentWithLogMessage)
	occurrencePost := generatePost("channelId", botUserID, occurrenceAttachment)
	occurrencePostWithTraceChain := generatePost("channelId", botUserID, occurrenceAttachmentWithTraceChain)
	reactivatedPost := generatePost("channelId", botUserID, reactivatedAttachment)
	reopenedPost := generatePost("channelId", botUserID, reopenedAttachment)
	resolvedPost := generatePost("channelId", botUserID, resolvedAttachment)

	testPost := &model.Post{
		ChannelId: "channelId",
		UserId:    botUserID,
		Message:   "This is a test payload from Rollbar. If you got this, it works!",
	}

	for name, test := range map[string]struct {
		SetupAPI         func(api *plugintest.API) *plugintest.API
		Method           string
		URL              string
		Body             []byte
		Configuration    *configuration
		ExpectedStatus   int
		ExpectedResponse string
	}{
		"error - 404 not found for non `/notify` path non-notify": {
			SetupAPI:         func(api *plugintest.API) *plugintest.API { return api },
			Method:           "GET",
			URL:              "/",
			Body:             emptyBody,
			Configuration:    &configuration{},
			ExpectedStatus:   http.StatusNotFound,
			ExpectedResponse: "404 page not found\n",
		},
		"error - 405 method not allowed for non-POST": {
			SetupAPI:         func(api *plugintest.API) *plugintest.API { return api },
			Method:           "GET",
			URL:              "/notify",
			Body:             emptyBody,
			Configuration:    &configuration{},
			ExpectedStatus:   http.StatusMethodNotAllowed,
			ExpectedResponse: "Method not allowed.\n",
		},
		"error - 401 unauthorized (unauthenticated) for no secret in request query params": {
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				api.On("LogWarn", mock.Anything).Return(nil)
				return api
			},
			Method:           "POST",
			URL:              "/notify",
			Body:             emptyBody,
			Configuration:    &configuration{Secret: "abc123"},
			ExpectedStatus:   http.StatusUnauthorized,
			ExpectedResponse: "Unauthenticated.\n",
		},
		"error - 400 bad request for no configured team and no team in query params": {
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				api.On("LogWarn", mock.Anything).Return(nil)
				return api
			},
			Method:           "POST",
			URL:              "/notify?auth=abc123",
			Body:             emptyBody,
			Configuration:    &configuration{Secret: "abc123"},
			ExpectedStatus:   http.StatusBadRequest,
			ExpectedResponse: "Missing 'team' query parameter.\n",
		},
		"error - 400 bad request for no configured channel and no channel in query params": {
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				api.On("LogWarn", mock.Anything).Return(nil)
				return api
			},
			Method: "POST",
			URL:    "/notify?auth=abc123",
			Body:   emptyBody,
			Configuration: &configuration{
				Secret: "abc123",
				teamId: "teamId",
			},
			ExpectedStatus:   http.StatusBadRequest,
			ExpectedResponse: "Missing 'channel' query parameter.\n",
		},
		"error - 400 bad request for no configured team and `api.get(query.team)` not found": {
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				api.On("LogWarn", mock.Anything).Return(nil)
				api.On("GetTeamByName", "nonexistentQueryTeam").Return(nil, nil)
				return api
			},
			Method: "POST",
			URL:    "/notify?auth=abc123&team=nonexistentQueryTeam",
			Body:   emptyBody,
			Configuration: &configuration{
				Secret:    "abc123",
				channelId: "teamId",
			},
			ExpectedStatus:   http.StatusBadRequest,
			ExpectedResponse: "Team 'nonexistentQueryTeam' does not exist.\n",
		},
		"error - 400 bad request for no configured channel and `api.get(query.channel)` not found": {
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				api.On("LogWarn", mock.Anything).Return(nil)
				api.On("GetChannelByName", "teamId", "nonexistentQueryChannel", false).Return(nil, nil)
				return api
			},
			Method: "POST",
			URL:    "/notify?auth=abc123&channel=nonexistentQueryChannel",
			Body:   emptyBody,
			Configuration: &configuration{
				Secret: "abc123",
				teamId: "teamId",
			},
			ExpectedStatus:   http.StatusBadRequest,
			ExpectedResponse: "Channel 'nonexistentQueryChannel' does not exist.\n",
		},
		"error - 400 bad request for config ok but request body json decode to rollbar err": {
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				api.On("LogError", mock.Anything).Return(nil)
				return api
			},
			Method: "POST",
			URL:    "/notify?auth=abc123",
			Body:   nil, // can't decode nil
			Configuration: &configuration{
				Secret:    "abc123",
				teamId:    "teamId",
				channelId: "channelId",
			},
			ExpectedStatus:   http.StatusBadRequest,
			ExpectedResponse: "EOF\n",
		},
		"error - 500 internal server error when failed to create post for whatever reason (event: new_item)": {
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				api.On("KVGet", "channelId").Return([]byte(""), nil)
				api.On("LogError", mock.AnythingOfType("string")).Return(nil)
				api.On("CreatePost", mock.AnythingOfType("*model.Post")).Return(nil, &model.AppError{Where: "server/http", Message: "error", DetailedError: "detailed error"})
				return api
			},
			Method: "POST",
			URL:    "/notify?auth=abc123",
			Body:   loadJsonFile(t, "new_item.json"),
			Configuration: &configuration{
				Secret:    "abc123",
				teamId:    "teamId",
				channelId: "channelId",
			},
			ExpectedStatus:   http.StatusInternalServerError,
			ExpectedResponse: "server/http: error, detailed error\n",
		},
		"error - 500 internal server error when failed to create post for whatever reason (event: item_velocity)": {
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				api.On("KVGet", "channelId").Return([]byte(""), nil)
				api.On("LogError", mock.AnythingOfType("string")).Return(nil)
				api.On("CreatePost", mock.AnythingOfType("*model.Post")).Return(nil, &model.AppError{Where: "server/http", Message: "error", DetailedError: "detailed error"})
				return api
			},
			Method: "POST",
			URL:    "/notify?auth=abc123",
			Body:   loadJsonFile(t, "item_velocity.json"),
			Configuration: &configuration{
				Secret:    "abc123",
				teamId:    "teamId",
				channelId: "channelId",
			},
			ExpectedStatus:   http.StatusInternalServerError,
			ExpectedResponse: "server/http: error, detailed error\n",
		},
		"error - 500 internal server error when failed to create post for whatever reason (event: test)": {
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				api.On("KVGet", "channelId").Return([]byte(""), nil)
				api.On("LogError", mock.AnythingOfType("string")).Return(nil)
				api.On("CreatePost", mock.AnythingOfType("*model.Post")).Return(nil, &model.AppError{Where: "server/http", Message: "error", DetailedError: "detailed error"})
				return api
			},
			Method: "POST",
			URL:    "/notify?auth=abc123",
			Body:   loadJsonFile(t, "test.json"),
			Configuration: &configuration{
				Secret:    "abc123",
				teamId:    "teamId",
				channelId: "channelId",
			},
			ExpectedStatus:   http.StatusInternalServerError,
			ExpectedResponse: "server/http: error, detailed error\n",
		},
		"error - 500 internal server error when failed to create post for whatever reason (event: deploy)": {
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				api.On("KVGet", "channelId").Return([]byte(""), nil)
				api.On("LogError", mock.AnythingOfType("string")).Return(nil)
				api.On("CreatePost", mock.AnythingOfType("*model.Post")).Return(nil, &model.AppError{Where: "server/http", Message: "error", DetailedError: "detailed error"})
				return api
			},
			Method: "POST",
			URL:    "/notify?auth=abc123",
			Body:   loadJsonFile(t, "deploy.json"),
			Configuration: &configuration{
				Secret:    "abc123",
				teamId:    "teamId",
				channelId: "channelId",
			},
			ExpectedStatus:   http.StatusInternalServerError,
			ExpectedResponse: "server/http: error, detailed error\n",
		},
		"ok - error in KVGet for users to notify is logged and ignored": {
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				api.On("KVGet", "channelId").Return([]byte(""), &model.AppError{})
				api.On("LogWarn", mock.AnythingOfType("string")).Return(nil)
				api.On("CreatePost", newItemPost).Return(nil, nil)
				return api
			},
			Method: "POST",
			URL:    "/notify?auth=abc123",
			Body:   loadJsonFile(t, "new_item.json"),
			Configuration: &configuration{
				Secret:    "abc123",
				teamId:    "teamId",
				channelId: "channelId",
			},
			ExpectedStatus:   http.StatusOK,
			ExpectedResponse: "",
		},
		"ok - error in json unmarshalling users to notify is logged and ignored": {
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				api.On("KVGet", "channelId").Return([]byte(`{"this":error`), nil)
				api.On("LogWarn", mock.AnythingOfType("string")).Return(nil)
				api.On("CreatePost", newItemPost).Return(nil, nil)
				return api
			},
			Method: "POST",
			URL:    "/notify?auth=abc123",
			Body:   loadJsonFile(t, "new_item.json"),
			Configuration: &configuration{
				Secret:    "abc123",
				teamId:    "teamId",
				channelId: "channelId",
			},
			ExpectedStatus:   http.StatusOK,
			ExpectedResponse: "",
		},
		// test cases for all testdata json
		"ok - deploy": {
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				api.On("KVGet", "channelId").Return([]byte(`{"daniel":true,"eric":true}`), nil)
				api.On("CreatePost", deployPost).Return(nil, nil)
				return api
			},
			Method: "POST",
			URL:    "/notify?auth=abc123",
			Body:   loadJsonFile(t, "deploy.json"),
			Configuration: &configuration{
				Secret:    "abc123",
				teamId:    "teamId",
				channelId: "channelId",
			},
			ExpectedStatus:   http.StatusOK,
			ExpectedResponse: "",
		},
		"ok - deploy with no username": {
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				api.On("KVGet", "channelId").Return([]byte(""), nil)
				api.On("CreatePost", deployPostWithNoUsername).Return(nil, nil)
				return api
			},
			Method: "POST",
			URL:    "/notify?auth=abc123",
			Body:   loadJsonFile(t, "deploy_no_username.json"),
			Configuration: &configuration{
				Secret:    "abc123",
				teamId:    "teamId",
				channelId: "channelId",
			},
			ExpectedStatus:   http.StatusOK,
			ExpectedResponse: "",
		},
		"ok - exp_repeat_item with trace_chain": {
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				api.On("KVGet", "channelId").Return([]byte(""), nil)
				api.On("CreatePost", expRepeatPostWithTraceChain).Return(nil, nil)
				return api
			},
			Method: "POST",
			URL:    "/notify?auth=abc123",
			Body:   loadJsonFile(t, "exp_repeat_item.json"),
			Configuration: &configuration{
				Secret:    "abc123",
				teamId:    "teamId",
				channelId: "channelId",
			},
			ExpectedStatus:   http.StatusOK,
			ExpectedResponse: "",
		},
		"ok - item_velocity": {
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				api.On("KVGet", "channelId").Return([]byte(""), nil)
				api.On("CreatePost", itemVelocityPost).Return(nil, nil)
				return api
			},
			Method: "POST",
			URL:    "/notify?auth=abc123",
			Body:   loadJsonFile(t, "item_velocity.json"),
			Configuration: &configuration{
				Secret:    "abc123",
				teamId:    "teamId",
				channelId: "channelId",
			},
			ExpectedStatus:   http.StatusOK,
			ExpectedResponse: "",
		},
		"ok - item_velocity with notify": {
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				api.On("KVGet", "channelId").Return([]byte(`{"daniel":true,"eric":true}`), nil)
				api.On("CreatePost", itemVelocityPostWithNotify).Return(nil, nil)
				return api
			},
			Method: "POST",
			URL:    "/notify?auth=abc123",
			Body:   loadJsonFile(t, "item_velocity.json"),
			Configuration: &configuration{
				Secret:    "abc123",
				teamId:    "teamId",
				channelId: "channelId",
			},
			ExpectedStatus:   http.StatusOK,
			ExpectedResponse: "",
		},
		"ok - new_item with query params for team and channel all present/exist": {
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				api.On("KVGet", "existingChannelId").Return([]byte(""), nil)
				api.On("GetTeamByName", "existingTeam").Return(&model.Team{Id: "existingTeamId"}, nil)
				api.On("GetChannelByName", "existingTeamId", "existingChannel", false).Return(&model.Channel{Id: "existingChannelId"}, nil)
				api.On("CreatePost", newItemPostWithChannelOverride).Return(nil, nil)
				return api
			},
			Method: "POST",
			URL:    "/notify?auth=abc123&team=existingTeam&channel=existingChannel",
			Body:   loadJsonFile(t, "new_item.json"),
			Configuration: &configuration{
				Secret:    "abc123",
				teamId:    "teamId",
				channelId: "channelId",
			},
			ExpectedStatus:   http.StatusOK,
			ExpectedResponse: "",
		},
		"ok - new_item with users to notify included in pretext": {
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				api.On("KVGet", "channelId").Return([]byte(`{"daniel":true,"eric":true}`), nil)
				api.On("CreatePost", newItemPostWithNotify).Return(nil, nil)
				return api
			},
			Method: "POST",
			URL:    "/notify?auth=abc123",
			Body:   loadJsonFile(t, "new_item.json"),
			Configuration: &configuration{
				Secret:    "abc123",
				teamId:    "teamId",
				channelId: "channelId",
			},
			ExpectedStatus:   http.StatusOK,
			ExpectedResponse: "",
		},
		"ok - new_item from java sdk with decimal customer timestamp": {
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				api.On("KVGet", "channelId").Return([]byte(""), nil)
				api.On("CreatePost", newItemPostFromJava).Return(nil, nil)
				return api
			},
			Method: "POST",
			URL:    "/notify?auth=abc123",
			Body:   loadJsonFile(t, "new_item_java.json"),
			Configuration: &configuration{
				Secret:    "abc123",
				teamId:    "teamId",
				channelId: "channelId",
			},
			ExpectedStatus:   http.StatusOK,
			ExpectedResponse: "",
		},
		"ok - new_item with log message body": {
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				api.On("KVGet", "channelId").Return([]byte(""), nil)
				api.On("CreatePost", newItemPostWithLogMessage).Return(nil, nil)
				return api
			},
			Method: "POST",
			URL:    "/notify?auth=abc123",
			Body:   loadJsonFile(t, "new_item_log_message.json"),
			Configuration: &configuration{
				Secret:    "abc123",
				teamId:    "teamId",
				channelId: "channelId",
			},
			ExpectedStatus:   http.StatusOK,
			ExpectedResponse: "",
		},
		"ok - occurrence": {
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				api.On("KVGet", "channelId").Return([]byte(""), nil)
				api.On("CreatePost", occurrencePost).Return(nil, nil)
				return api
			},
			Method: "POST",
			URL:    "/notify?auth=abc123",
			Body:   loadJsonFile(t, "occurrence.json"),
			Configuration: &configuration{
				Secret:    "abc123",
				teamId:    "teamId",
				channelId: "channelId",
			},
			ExpectedStatus:   http.StatusOK,
			ExpectedResponse: "",
		},
		"ok - occurrence with trace_chain": {
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				api.On("KVGet", "channelId").Return([]byte(""), nil)
				api.On("CreatePost", occurrencePostWithTraceChain).Return(nil, nil)
				return api
			},
			Method: "POST",
			URL:    "/notify?auth=abc123",
			Body:   loadJsonFile(t, "occurrence_trace_chain.json"),
			Configuration: &configuration{
				Secret:    "abc123",
				teamId:    "teamId",
				channelId: "channelId",
			},
			ExpectedStatus:   http.StatusOK,
			ExpectedResponse: "",
		},
		"ok - reactivated_item": {
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				api.On("KVGet", "channelId").Return([]byte(""), nil)
				api.On("CreatePost", reactivatedPost).Return(nil, nil)
				return api
			},
			Method: "POST",
			URL:    "/notify?auth=abc123",
			Body:   loadJsonFile(t, "reactivated_item.json"),
			Configuration: &configuration{
				Secret:    "abc123",
				teamId:    "teamId",
				channelId: "channelId",
			},
			ExpectedStatus:   http.StatusOK,
			ExpectedResponse: "",
		},
		"ok - reopened_item": {
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				api.On("KVGet", "channelId").Return([]byte(""), nil)
				api.On("CreatePost", reopenedPost).Return(nil, nil)
				return api
			},
			Method: "POST",
			URL:    "/notify?auth=abc123",
			Body:   loadJsonFile(t, "reopened_item.json"),
			Configuration: &configuration{
				Secret:    "abc123",
				teamId:    "teamId",
				channelId: "channelId",
			},
			ExpectedStatus:   http.StatusOK,
			ExpectedResponse: "",
		},
		"ok - resolved_item": {
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				api.On("KVGet", "channelId").Return([]byte(""), nil)
				api.On("CreatePost", resolvedPost).Return(nil, nil)
				return api
			},
			Method: "POST",
			URL:    "/notify?auth=abc123",
			Body:   loadJsonFile(t, "resolved_item.json"),
			Configuration: &configuration{
				Secret:    "abc123",
				teamId:    "teamId",
				channelId: "channelId",
			},
			ExpectedStatus:   http.StatusOK,
			ExpectedResponse: "",
		},
		"ok - test webhook": {
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				api.On("KVGet", "channelId").Return([]byte(`{"daniel":true,"eric":true}`), nil)
				api.On("CreatePost", testPost).Return(nil, nil)
				return api
			},
			Method: "POST",
			URL:    "/notify?auth=abc123",
			Body:   loadJsonFile(t, "test.json"),
			Configuration: &configuration{
				Secret:    "abc123",
				teamId:    "teamId",
				channelId: "channelId",
			},
			ExpectedStatus:   http.StatusOK,
			ExpectedResponse: "",
		},
	} {
		t.Run(name, func(t *testing.T) {
			api := test.SetupAPI(&plugintest.API{})
			defer api.AssertExpectations(t)
			p := &RollbarPlugin{}
			p.botUserID = botUserID
			p.SetAPI(api)
			p.setConfiguration(test.Configuration)
			w := httptest.NewRecorder()
			r := httptest.NewRequest(test.Method, test.URL, bytes.NewReader(test.Body))

			p.ServeHTTP(nil, w, r)

			response := w.Result()
			body, _ := ioutil.ReadAll(response.Body)

			if response.StatusCode != test.ExpectedStatus {
				t.Errorf("Expected status: %d\nActual: %d", test.ExpectedStatus, response.StatusCode)
			}
			if string(body) != test.ExpectedResponse {
				t.Errorf("Expected response: %s\nActual: %s", test.ExpectedResponse, string(body))
			}
		})
	}
}
