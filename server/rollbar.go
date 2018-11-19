package main

import (
	"fmt"
	"strings"
)

type OccurrenceBody struct {
	Message struct {
		Body string `json:"body"`
	} `json:"message"`
	Trace struct {
		Exception struct {
			Class   string `json:"class"`
			Message string `json:"message"`
		} `json:"exception"`
		Frames []struct {
			Code     string `json:"code"`
			Filename string `json:"filename"`
			LineNo   int    `json:"lineno"`
			Locals   struct {
				Builtins string `json:"__builtins__"`
				Doc      string `json:"__doc__"`
				File     string `json:"__file__"`
				Name     string `json:"__name__"`
				Package  string `json:"__package__"`
				Rollbar  string `json:"rollbar"`
			} `json:"locals"`
			Method string `json:"method"`
		} `json:"frames"`
	} `json:"trace"`
}

type OccurrenceMetadata struct {
	AccessToken       string `json:"access_token"`
	ApiServerHostname string `json:"api_server_hostname"`
	CustomerTimestamp int    `json:"customer_timestamp"`
	Debug             struct {
		Routes struct {
			Counters struct {
				PostItem int `json:"post_item"`
			} `json:"counters"`
			StartTime int `json:"start_time"`
		} `json:"routes"`
	} `json:"debug"`
	TimestampMs int `json:"timestamp_ms"`
}

type LastOccurrence struct {
	Body        *OccurrenceBody     `json:"body"`
	Environment string              `json:"environment"`
	Framework   string              `json:"framework"`
	Language    string              `json:"language"`
	Level       string              `json:"level"`
	Metadata    *OccurrenceMetadata `json:"metadata"`
	Notifier    struct {
		Name    string `json:"name"`
		Version string `json:"version"`
	} `json:"notifier"`
	Server struct {
		Argv []string `json:"argv"`
		Host string   `json:"host"`
		Pid  int      `json:"pid"`
	} `json:"server"`
	Timestamp int    `json:"timestamp"`
	Uuid      string `json:"uuid"`
}

type Rollbar struct {
	EventName string `json:"event_name"`
	Data      struct {
		Item struct {
			ActivatingOccurrenceId   int             `json:"activating_occurrence_id"`
			AssignedUserId           *int            `json:"assigned_user_id"`
			Counter                  int             `json:"counter"`
			Environment              string          `json:"environment"`
			FirstOccurrenceId        int             `json:"first_occurrence_id"`
			FirstOccurrenceTimestamp int             `json:"first_occurrence_timestamp"`
			Framework                int             `json:"framework"`
			Hash                     string          `json:"hash"`
			Id                       int             `json:"id"`
			IntegrationsData         struct{}        `json:"integrations_data"`
			LastActivatedTimestamp   int             `json:"last_activated_timestamp"`
			LastModifiedBy           *int            `json:"last_modified_by"`
			LastOccurrence           *LastOccurrence `json:"last_occurrence"`
			LastOccurrenceId         int             `json:"last_occurrence_id"`
			LastOccurrenceTimestamp  int             `json:"last_occurrence_timestamp"`
			Level                    int             `json:"level"`
			LevelLock                int             `json:"level_lock"`
			Platform                 int             `json:"platform"`
			ProjectId                int             `json:"project_id"`
			PublicItemId             *int            `json:"public_item_id"`
			ResolvedInVersion        *string         `json:"resolved_in_version"`
			Status                   int             `json:"status"`
			Title                    string          `json:"title"`
			TitleLock                int             `json:"title_lock"`
			TotalOccurrences         int             `json:"total_occurrences"`
			UniqueOccurrences        *int            `json:"unique_occurrences"`
		} `json:"item"`
		Occurrence  *LastOccurrence `json:"occurrence"`
		Occurrences int             `json:"occurrences"`
		Trigger     struct {
			Threshold             int    `json:"threshold"`
			WindowSize            int    `json:"window_size"`
			WindowSizeDescription string `json:"window_size_description"`
		} `json:"trigger"`
	} `json:"data"`
}

func (rollbar *Rollbar) eventNameToTitle() string {
	prefix := ""
	title := ""
	lastOccurrence := rollbar.Data.Item.LastOccurrence

	// event type `occurrence` has data under `occurrence` instead of `last_occurrence`
	if lastOccurrence == nil {
		lastOccurrence = rollbar.Data.Occurrence
	}

	level := strings.Title(lastOccurrence.Level)

	switch rollbar.EventName {
	case "new_item":
		prefix = "New"
	case "occurrence":
		prefix = "Occurrence -"
	case "reactivated_item":
		prefix = "Reactivated"
	case "reopened_item":
		prefix = "Reopened"
	case "resolved_item":
		prefix = "Resolved"
	case "exp_repeat_item":
		prefix = fmt.Sprintf("%dth", rollbar.Data.Occurrences)
	case "item_velocity":
		triggerData := rollbar.Data.Trigger
		title = fmt.Sprintf("%d occurrences in %s", triggerData.Threshold, triggerData.WindowSizeDescription)
	}

	// non item_velocity case
	if title == "" {
		title = fmt.Sprintf("%s %s", prefix, level)
	}

	return title
}
