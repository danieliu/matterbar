package main

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

const (
	timeLayout = "2006-01-02 15:04:05 MST-0700"
)

type Trace struct {
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
}

type OccurrenceBody struct {
	Message struct {
		Body string `json:"body"`
	} `json:"message"`
	Trace      Trace   `json:"trace"`
	TraceChain []Trace `json:"trace_chain"`
}

type OccurrenceMetadata struct {
	AccessToken       string      `json:"access_token"`
	APIServerHostname string      `json:"api_server_hostname"`
	CustomerTimestamp json.Number `json:"customer_timestamp"`
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
	UUID      string `json:"uuid"`
}

type Deploy struct {
	Comment       *string `json:"comment"`
	Environment   string  `json:"environment"`
	FinishTime    int64   `json:"finish_time"`
	ID            int     `json:"id"`
	LocalUsername *string `json:"local_username"`
	ProjectID     int     `json:"project_id"`
	Revision      string  `json:"revision"`
	StartTime     int64   `json:"start_time"`
	UserID        *int    `json:"user_id"`
}

type Rollbar struct {
	EventName string `json:"event_name"`
	Data      struct {
		Deploy Deploy `json:"deploy"`
		Item   struct {
			ActivatingOccurrenceID   int             `json:"activating_occurrence_id"`
			AssignedUserID           *int            `json:"assigned_user_id"`
			Counter                  int             `json:"counter"`
			Environment              string          `json:"environment"`
			FirstOccurrenceID        int             `json:"first_occurrence_id"`
			FirstOccurrenceTimestamp int             `json:"first_occurrence_timestamp"`
			Framework                int             `json:"framework"`
			Hash                     string          `json:"hash"`
			ID                       int             `json:"id"`
			IntegrationsData         struct{}        `json:"integrations_data"`
			LastActivatedTimestamp   int             `json:"last_activated_timestamp"`
			LastModifiedBy           *int            `json:"last_modified_by"`
			LastOccurrence           *LastOccurrence `json:"last_occurrence"`
			LastOccurrenceID         int             `json:"last_occurrence_id"`
			LastOccurrenceTimestamp  int             `json:"last_occurrence_timestamp"`
			Level                    int             `json:"level"`
			LevelLock                int             `json:"level_lock"`
			Platform                 int             `json:"platform"`
			ProjectID                int             `json:"project_id"`
			PublicItemID             *int            `json:"public_item_id"`
			ResolvedInVersion        *string         `json:"resolved_in_version"`
			Status                   int             `json:"status"`
			Title                    string          `json:"title"`
			TitleLock                int             `json:"title_lock"`
			TotalOccurrences         int             `json:"total_occurrences"`
			UniqueOccurrences        *int            `json:"unique_occurrences"`
		} `json:"item"`
		Message     string          `json:"message"`
		Occurrence  *LastOccurrence `json:"occurrence"`
		Occurrences int             `json:"occurrences"`
		Trigger     struct {
			Threshold             int    `json:"threshold"`
			WindowSize            int    `json:"window_size"`
			WindowSizeDescription string `json:"window_size_description"`
		} `json:"trigger"`
		URL string `json:"url"`
	} `json:"data"`
}

func (rollbar *Rollbar) eventNameToTitle() string {
	prefix := ""
	title := ""

	switch rollbar.EventName {
	case "test":
		return ""
	case "deploy":
		return "Deploy"
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
		// TODO: include level, e.g. error or warning, when rollbar fixes item_velocity
		triggerData := rollbar.Data.Trigger
		title = fmt.Sprintf("%d occurrences in %s", triggerData.Threshold, triggerData.WindowSizeDescription)
	}

	// item_velocity (high occurrence) doesn't include occurrence data
	if rollbar.EventName != "item_velocity" {
		lastOccurrence := rollbar.Data.Item.LastOccurrence

		// event_name `occurrence` has data under `occurrence` instead of `last_occurrence`
		if lastOccurrence == nil {
			lastOccurrence = rollbar.Data.Occurrence
		}

		level := strings.Title(lastOccurrence.Level)
		title = fmt.Sprintf("%s %s", prefix, level)
	}

	return title
}

func (rollbar *Rollbar) eventText() string {
	// item_velocity (high occurrence) doesn't include occurrence data
	if rollbar.Data.Item.LastOccurrence == nil && rollbar.Data.Occurrence == nil {
		return ""
	}

	eventMessage := ""
	occurrenceData := rollbar.Data.Item.LastOccurrence
	// event_name `occurrence` has data under `occurrence` instead of `last_occurrence`
	if occurrenceData == nil {
		occurrenceData = rollbar.Data.Occurrence
	}

	if occurrenceData.Body.Trace.Exception.Message != "" {
		// Option 1: trace; single exception with stack trace
		exception := occurrenceData.Body.Trace.Exception
		eventMessage = fmt.Sprintf("%s: %s", exception.Class, exception.Message)
	} else if len(occurrenceData.Body.TraceChain) > 0 {
		// Option 2: trace_chain; a list of `trace`s (inner exceptions or causes)
		// first `trace` is the most recent
		exception := occurrenceData.Body.TraceChain[0].Exception
		eventMessage = fmt.Sprintf("%s: %s", exception.Class, exception.Message)
	} else if occurrenceData.Body.Message.Body != "" {
		// Option 3: message with no stack trace
		eventMessage = occurrenceData.Body.Message.Body
	}
	// TODO: Option 4: crash_report; iOS crash report

	return eventMessage
}

func (rollbar *Rollbar) deployUser() string {
	data := rollbar.Data.Deploy
	username := "unknown user"
	if data.LocalUsername != nil {
		username = *data.LocalUsername
	}

	return username
}

func (rollbar *Rollbar) deployDateTime() string {
	data := rollbar.Data.Deploy
	finishTime := time.Unix(data.FinishTime, 0)
	return finishTime.Format(timeLayout)
}
