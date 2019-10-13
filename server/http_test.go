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

func TestServeHttp(t *testing.T) {
	emptyBody := []byte("{}")
	itemURL := "https://rollbar.com/organization/project/items/%s/"
	deployLink := "https://rollbar.com/deploy/%d/"
	itemLink := "https://rollbar.com/item/uuid/?uuid=2e7cbf0a-a3af-402a-ab4f-95e07e5982f8"
	occurrenceLink := "https://rollbar.com/occurrence/uuid/?uuid=2e7cbf0a-a3af-402a-ab4f-95e07e5982f8"
	attachmentFields := []*model.SlackAttachmentField{
		&model.SlackAttachmentField{
			Short: true,
			Title: "Environment",
			Value: "live",
		},
		&model.SlackAttachmentField{
			Short: true,
			Title: "Framework",
			Value: "flask",
		},
		&model.SlackAttachmentField{
			Short: true,
			Title: "Language",
			Value: "python 2.7.14",
		},
		&model.SlackAttachmentField{
			Short: true,
			Title: "Links",
			Value: fmt.Sprintf("[Item](%s) | [Occurrence](%s)", itemLink, occurrenceLink),
		},
	}
	nonNotifyAttachment := []*model.SlackAttachment{
		&model.SlackAttachment{
			Color:     "#ff0000",
			Fallback:  "[live] New Error - TypeError: unsupported operand type(s) for +=: 'int' and 'str'",
			Fields:    attachmentFields,
			Title:     "New Error",
			TitleLink: itemLink,
			Text:      "```\nTypeError: unsupported operand type(s) for +=: 'int' and 'str'\n```",
		},
	}
	withNotifyAttachment := []*model.SlackAttachment{
		&model.SlackAttachment{
			Color:     "#ff0000",
			Fallback:  "[live] New Error - TypeError: unsupported operand type(s) for +=: 'int' and 'str'",
			Fields:    attachmentFields,
			Pretext:   "@daniel, @eric",
			Title:     "New Error",
			TitleLink: itemLink,
			Text:      "```\nTypeError: unsupported operand type(s) for +=: 'int' and 'str'\n```",
		},
	}
	occurrenceAttachment := []*model.SlackAttachment{
		&model.SlackAttachment{
			Color:     "#ff0000",
			Fallback:  "[live] Occurrence - Error - TypeError: 'NoneType' object has no attribute '__getitem__'",
			Fields:    attachmentFields,
			Title:     "Occurrence - Error",
			TitleLink: itemLink,
			Text:      "```\nTypeError: 'NoneType' object has no attribute '__getitem__'\n```",
		},
	}
	expRepeatTraceChainAttachment := []*model.SlackAttachment{
		&model.SlackAttachment{
			Color:     "#800080",
			Fallback:  "[live] 10th Error - Exception: foo",
			Fields:    attachmentFields,
			Title:     "10th Error",
			TitleLink: itemLink,
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

	nonNotifyOverridePost := &model.Post{
		ChannelId: "existingChannelId",
		UserId:    "userId",
		Type:      model.POST_SLACK_ATTACHMENT,
		Props: map[string]interface{}{
			"from_webhook":  "true",
			"use_user_icon": "true",
			"attachments":   nonNotifyAttachment,
		},
	}
	nonNotifyPost := &model.Post{
		ChannelId: "channelId",
		UserId:    "userId",
		Type:      model.POST_SLACK_ATTACHMENT,
		Props: map[string]interface{}{
			"from_webhook":  "true",
			"use_user_icon": "true",
			"attachments":   nonNotifyAttachment,
		},
	}
	withNotifyPost := &model.Post{
		ChannelId: "channelId",
		UserId:    "userId",
		Type:      model.POST_SLACK_ATTACHMENT,
		Props: map[string]interface{}{
			"from_webhook":  "true",
			"use_user_icon": "true",
			"attachments":   withNotifyAttachment,
		},
	}
	occurrencePost := &model.Post{
		ChannelId: "channelId",
		UserId:    "userId",
		Type:      model.POST_SLACK_ATTACHMENT,
		Props: map[string]interface{}{
			"from_webhook":  "true",
			"use_user_icon": "true",
			"attachments":   occurrenceAttachment,
		},
	}
	expRepeatTraceChainPost := &model.Post{
		ChannelId: "channelId",
		UserId:    "userId",
		Type:      model.POST_SLACK_ATTACHMENT,
		Props: map[string]interface{}{
			"from_webhook":  "true",
			"use_user_icon": "true",
			"attachments":   expRepeatTraceChainAttachment,
		},
	}
	itemVelocityPost := &model.Post{
		ChannelId: "channelId",
		UserId:    "userId",
		Type:      model.POST_SLACK_ATTACHMENT,
		Props: map[string]interface{}{
			"from_webhook":  "true",
			"use_user_icon": "true",
			"attachments":   itemVelocityAttachment,
		},
	}
	itemVelocityPostWithNotify := &model.Post{
		ChannelId: "channelId",
		UserId:    "userId",
		Type:      model.POST_SLACK_ATTACHMENT,
		Props: map[string]interface{}{
			"from_webhook":  "true",
			"use_user_icon": "true",
			"attachments":   itemVelocityAttachmentWithNotify,
		},
	}
	testPost := &model.Post{
		ChannelId: "channelId",
		UserId:    "userId",
		Message:   "This is a test payload from Rollbar. If you got this, it works!",
		Props: map[string]interface{}{
			"from_webhook":  "true",
			"use_user_icon": "true",
		},
	}
	deployPost := &model.Post{
		ChannelId: "channelId",
		UserId:    "userId",
		Type:      model.POST_SLACK_ATTACHMENT,
		Props: map[string]interface{}{
			"from_webhook":  "true",
			"use_user_icon": "true",
			"attachments":   deployAttachment,
		},
	}

	for name, test := range map[string]struct {
		SetupAPI         func(api *plugintest.API) *plugintest.API
		Method           string
		Url              string
		Body             []byte
		Configuration    *configuration
		ExpectedStatus   int
		ExpectedResponse string
	}{
		"error - non-notify defaults to 404": {
			SetupAPI:         func(api *plugintest.API) *plugintest.API { return api },
			Method:           "GET",
			Url:              "/",
			Body:             emptyBody,
			Configuration:    &configuration{},
			ExpectedStatus:   http.StatusNotFound,
			ExpectedResponse: "404 page not found\n",
		},
		"error - non-post method not allowed": {
			SetupAPI:         func(api *plugintest.API) *plugintest.API { return api },
			Method:           "GET",
			Url:              "/notify",
			Body:             emptyBody,
			Configuration:    &configuration{},
			ExpectedStatus:   http.StatusMethodNotAllowed,
			ExpectedResponse: "Method not allowed.\n",
		},
		"error - unauthenticated request": {
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				api.On("LogWarn", mock.Anything).Return(nil)
				return api
			},
			Method:           "POST",
			Url:              "/notify",
			Body:             emptyBody,
			Configuration:    &configuration{Secret: "abc123"},
			ExpectedStatus:   http.StatusUnauthorized,
			ExpectedResponse: "Unauthenticated.\n",
		},
		"error - no configured team, no team in query params": {
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				api.On("LogWarn", mock.Anything).Return(nil)
				return api
			},
			Method:           "POST",
			Url:              "/notify?auth=abc123",
			Body:             emptyBody,
			Configuration:    &configuration{Secret: "abc123"},
			ExpectedStatus:   http.StatusBadRequest,
			ExpectedResponse: "Missing 'team' query parameter.\n",
		},
		"error - no configured channel, no channel in query params": {
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				api.On("LogWarn", mock.Anything).Return(nil)
				return api
			},
			Method: "POST",
			Url:    "/notify?auth=abc123",
			Body:   emptyBody,
			Configuration: &configuration{
				Secret: "abc123",
				teamId: "teamId",
			},
			ExpectedStatus:   http.StatusBadRequest,
			ExpectedResponse: "Missing 'channel' query parameter.\n",
		},
		"error - no configured team, api.get query team not found": {
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				api.On("LogWarn", mock.Anything).Return(nil)
				api.On("GetTeamByName", "nonexistentQueryTeam").Return(nil, nil)
				return api
			},
			Method: "POST",
			Url:    "/notify?auth=abc123&team=nonexistentQueryTeam",
			Body:   emptyBody,
			Configuration: &configuration{
				Secret:    "abc123",
				channelId: "teamId",
			},
			ExpectedStatus:   http.StatusBadRequest,
			ExpectedResponse: "Team 'nonexistentQueryTeam' does not exist.\n",
		},
		"error - no configured channel, api.get query channel not found": {
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				api.On("LogWarn", mock.Anything).Return(nil)
				api.On("GetChannelByName", "teamId", "nonexistentQueryChannel", false).Return(nil, nil)
				return api
			},
			Method: "POST",
			Url:    "/notify?auth=abc123&channel=nonexistentQueryChannel",
			Body:   emptyBody,
			Configuration: &configuration{
				Secret: "abc123",
				teamId: "teamId",
			},
			ExpectedStatus:   http.StatusBadRequest,
			ExpectedResponse: "Channel 'nonexistentQueryChannel' does not exist.\n",
		},
		"error - good configs, request body json decode to rollbar err": {
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				api.On("LogError", mock.Anything).Return(nil)
				return api
			},
			Method: "POST",
			Url:    "/notify?auth=abc123",
			Body:   nil, // can't decode nil
			Configuration: &configuration{
				Secret:    "abc123",
				teamId:    "teamId",
				channelId: "channelId",
			},
			ExpectedStatus:   http.StatusBadRequest,
			ExpectedResponse: "EOF\n",
		},
		"error - failed to create post for whatever reason": {
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				api.On("KVGet", "channelId").Return([]byte(""), nil)
				api.On("LogError", mock.AnythingOfType("string")).Return(nil)
				api.On("CreatePost", mock.AnythingOfType("*model.Post")).Return(nil, &model.AppError{Where: "server/http", Message: "error", DetailedError: "detailed error"})
				return api
			},
			Method: "POST",
			Url:    "/notify?auth=abc123",
			Body:   loadJsonFile(t, "new_item.json"),
			Configuration: &configuration{
				Secret:    "abc123",
				teamId:    "teamId",
				channelId: "channelId",
			},
			ExpectedStatus:   http.StatusInternalServerError,
			ExpectedResponse: "server/http: error, detailed error\n",
		},
		"error - item_velocity failed to create post for whatever reason": {
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				api.On("KVGet", "channelId").Return([]byte(""), nil)
				api.On("LogError", mock.AnythingOfType("string")).Return(nil)
				api.On("CreatePost", mock.AnythingOfType("*model.Post")).Return(nil, &model.AppError{Where: "server/http", Message: "error", DetailedError: "detailed error"})
				return api
			},
			Method: "POST",
			Url:    "/notify?auth=abc123",
			Body:   loadJsonFile(t, "item_velocity.json"),
			Configuration: &configuration{
				Secret:    "abc123",
				teamId:    "teamId",
				channelId: "channelId",
			},
			ExpectedStatus:   http.StatusInternalServerError,
			ExpectedResponse: "server/http: error, detailed error\n",
		},
		"error - test webhook failed to create post for whatever reason": {
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				api.On("KVGet", "channelId").Return([]byte(""), nil)
				api.On("LogError", mock.AnythingOfType("string")).Return(nil)
				api.On("CreatePost", mock.AnythingOfType("*model.Post")).Return(nil, &model.AppError{Where: "server/http", Message: "error", DetailedError: "detailed error"})
				return api
			},
			Method: "POST",
			Url:    "/notify?auth=abc123",
			Body:   loadJsonFile(t, "test.json"),
			Configuration: &configuration{
				Secret:    "abc123",
				teamId:    "teamId",
				channelId: "channelId",
			},
			ExpectedStatus:   http.StatusInternalServerError,
			ExpectedResponse: "server/http: error, detailed error\n",
		},
		"error - deploy failed to create post for whatever reason": {
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				api.On("KVGet", "channelId").Return([]byte(""), nil)
				api.On("LogError", mock.AnythingOfType("string")).Return(nil)
				api.On("CreatePost", mock.AnythingOfType("*model.Post")).Return(nil, &model.AppError{Where: "server/http", Message: "error", DetailedError: "detailed error"})
				return api
			},
			Method: "POST",
			Url:    "/notify?auth=abc123",
			Body:   loadJsonFile(t, "deploy.json"),
			Configuration: &configuration{
				Secret:    "abc123",
				teamId:    "teamId",
				channelId: "channelId",
			},
			ExpectedStatus:   http.StatusInternalServerError,
			ExpectedResponse: "server/http: error, detailed error\n",
		},
		"ok - error in KVGet for notify users logged and ignored": {
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				api.On("KVGet", "channelId").Return([]byte(""), &model.AppError{})
				api.On("LogWarn", mock.AnythingOfType("string")).Return(nil)
				api.On("CreatePost", nonNotifyPost).Return(nil, nil)
				return api
			},
			Method: "POST",
			Url:    "/notify?auth=abc123",
			Body:   loadJsonFile(t, "new_item.json"),
			Configuration: &configuration{
				Secret:    "abc123",
				userId:    "userId",
				teamId:    "teamId",
				channelId: "channelId",
			},
			ExpectedStatus:   http.StatusOK,
			ExpectedResponse: "",
		},
		"ok - error in json unmarshal for notify users logged and ignored": {
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				api.On("KVGet", "channelId").Return([]byte(`{"this":error`), nil)
				api.On("LogWarn", mock.AnythingOfType("string")).Return(nil)
				api.On("CreatePost", nonNotifyPost).Return(nil, nil)
				return api
			},
			Method: "POST",
			Url:    "/notify?auth=abc123",
			Body:   loadJsonFile(t, "new_item.json"),
			Configuration: &configuration{
				Secret:    "abc123",
				userId:    "userId",
				teamId:    "teamId",
				channelId: "channelId",
			},
			ExpectedStatus:   http.StatusOK,
			ExpectedResponse: "",
		},
		"ok - query params team, channel present and exist": {
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				api.On("KVGet", "existingChannelId").Return([]byte(""), nil)
				api.On("GetTeamByName", "existingTeam").Return(&model.Team{Id: "existingTeamId"}, nil)
				api.On("GetChannelByName", "existingTeamId", "existingChannel", false).Return(&model.Channel{Id: "existingChannelId"}, nil)
				api.On("CreatePost", nonNotifyOverridePost).Return(nil, nil)
				return api
			},
			Method: "POST",
			Url:    "/notify?auth=abc123&team=existingTeam&channel=existingChannel",
			Body:   loadJsonFile(t, "new_item.json"),
			Configuration: &configuration{
				Secret:    "abc123",
				userId:    "userId",
				teamId:    "teamId",
				channelId: "channelId",
			},
			ExpectedStatus:   http.StatusOK,
			ExpectedResponse: "",
		},
		"ok - notifies users in pretext": {
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				api.On("KVGet", "channelId").Return([]byte(`{"daniel":true,"eric":true}`), nil)
				api.On("CreatePost", withNotifyPost).Return(nil, nil)
				return api
			},
			Method: "POST",
			Url:    "/notify?auth=abc123",
			Body:   loadJsonFile(t, "new_item.json"),
			Configuration: &configuration{
				Secret:    "abc123",
				userId:    "userId",
				teamId:    "teamId",
				channelId: "channelId",
			},
			ExpectedStatus:   http.StatusOK,
			ExpectedResponse: "",
		},
		"ok - occurrence json": {
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				api.On("KVGet", "channelId").Return([]byte(""), nil)
				api.On("CreatePost", occurrencePost).Return(nil, nil)
				return api
			},
			Method: "POST",
			Url:    "/notify?auth=abc123",
			Body:   loadJsonFile(t, "occurrence.json"),
			Configuration: &configuration{
				Secret:    "abc123",
				userId:    "userId",
				teamId:    "teamId",
				channelId: "channelId",
			},
			ExpectedStatus:   http.StatusOK,
			ExpectedResponse: "",
		},
		"ok - exp_repeat_item json with trace_chain": {
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				api.On("KVGet", "channelId").Return([]byte(""), nil)
				api.On("CreatePost", expRepeatTraceChainPost).Return(nil, nil)
				return api
			},
			Method: "POST",
			Url:    "/notify?auth=abc123",
			Body:   loadJsonFile(t, "exp_repeat_item.json"),
			Configuration: &configuration{
				Secret:    "abc123",
				userId:    "userId",
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
			Url:    "/notify?auth=abc123",
			Body:   loadJsonFile(t, "item_velocity.json"),
			Configuration: &configuration{
				Secret:    "abc123",
				userId:    "userId",
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
			Url:    "/notify?auth=abc123",
			Body:   loadJsonFile(t, "item_velocity.json"),
			Configuration: &configuration{
				Secret:    "abc123",
				userId:    "userId",
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
			Url:    "/notify?auth=abc123",
			Body:   loadJsonFile(t, "test.json"),
			Configuration: &configuration{
				Secret:    "abc123",
				userId:    "userId",
				teamId:    "teamId",
				channelId: "channelId",
			},
			ExpectedStatus:   http.StatusOK,
			ExpectedResponse: "",
		},
		"ok - deploy webhook": {
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				api.On("KVGet", "channelId").Return([]byte(`{"daniel":true,"eric":true}`), nil)
				api.On("CreatePost", deployPost).Return(nil, nil)
				return api
			},
			Method: "POST",
			Url:    "/notify?auth=abc123",
			Body:   loadJsonFile(t, "deploy.json"),
			Configuration: &configuration{
				Secret:    "abc123",
				userId:    "userId",
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
			p.SetAPI(api)
			p.setConfiguration(test.Configuration)
			w := httptest.NewRecorder()
			r := httptest.NewRequest(test.Method, test.Url, bytes.NewReader(test.Body))

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
