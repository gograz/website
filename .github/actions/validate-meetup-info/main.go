package main

import (
	"context"
	"flag"
	"io"
	"io/fs"
	"log/slog"
	"os"
	"os/signal"
	"time"

	"github.com/machinebox/graphql"
	"github.com/willabides/actionslog"
	"github.com/willabides/actionslog/human"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/parser"
	"go.abhg.dev/goldmark/frontmatter"
)

type MeetupInfo struct {
	ID           string `yaml:"meetupID" toml:"meetupID"`
	RawStartTime string `yaml:"date" toml:"date"`
	Canceled     bool   `yaml:"canceled" toml:"canceled"`
	Filename     string `yaml:"-" toml:"-"`
}

func main() {
	ctx := context.Background()
	ctx, cancel := signal.NotifyContext(ctx, os.Interrupt)
	defer cancel()

	var groupName string
	flag.StringVar(&groupName, "group-name", "graz-open-source-meetup", "URL name of the meetup group")
	flag.Parse()

	handler := &human.Handler{}
	logger := slog.New(&actionslog.Wrapper{
		Handler: handler.WithOutput,
	})

	localMeetupEvents := make([]MeetupInfo, 0, 10)

	rootFS := os.DirFS(".")
	for _, pat := range flag.Args() {
		matches, err := fs.Glob(rootFS, pat)
		if err != nil {
			logger.Error("failed to glob", "pattern", pat, "err", err)
			os.Exit(1)
		}
		for _, match := range matches {
			info, err := getMeetupInfo(rootFS, match)
			if err != nil {
				logger.Error("failed parse metadata", "match", match, "err", err)
				os.Exit(1)
			}
			if info != nil {
				if info.Canceled {
					continue
				}
				info.Filename = match
				localMeetupEvents = append(localMeetupEvents, *info)
			}
		}
	}

	if len(localMeetupEvents) == 0 {
		logger.WarnContext(ctx, "no local events found")
		return
	}

	remoteEvents := make(map[string]RemoteMeetupData)

	pastEvents, err := fetchPastMeetupData(ctx, groupName)
	if err != nil {
		logger.Error("failed fetching remote events", "err", err)
		os.Exit(1)
	}
	logger.InfoContext(ctx, "past event data retrieved", "num_events", len(pastEvents))
	upcomingEvents, err := fetchUpcomingMeetupData(ctx, groupName)
	if err != nil {
		logger.Error("failed fetching remote events", "err", err)
		os.Exit(1)
	}
	logger.InfoContext(ctx, "upcoming event data retrieved", "num_events", len(upcomingEvents))

	for _, event := range pastEvents {
		remoteEvents[event.ID] = event
	}
	for _, event := range upcomingEvents {
		remoteEvents[event.ID] = event
	}

	var failed bool
	for _, event := range localMeetupEvents {
		remoteEvent, found := remoteEvents[event.ID]
		if !found {
			logger.ErrorContext(ctx, "unknown meetup id", "filename", event.Filename, "eventID", event.ID)
			failed = true
		}

		parsedRemoteTime, err := time.Parse("2006-01-02T15:04Z07:00", remoteEvent.DateTime)
		if err != nil {
			logger.ErrorContext(ctx, "parsing remote date failed", "err", err, "eventID", remoteEvent.ID)
			failed = true
			continue
		}
		parsedLocalTime, err := time.Parse(time.RFC3339, event.RawStartTime)
		if err != nil {
			logger.ErrorContext(ctx, "parsing local date failed", "err", err, "filename", event.Filename)
			failed = true
			continue
		}
		if parsedLocalTime != parsedRemoteTime {
			logger.ErrorContext(ctx, "time mismatch", "filename", event.Filename, "eventID", event.ID, "fileTime", parsedLocalTime, "meetupTime", parsedRemoteTime)
			failed = true
		}
	}

	if failed {
		os.Exit(1)
	}
}

func getMeetupInfo(rootFS fs.FS, path string) (*MeetupInfo, error) {
	raw, err := fs.ReadFile(rootFS, path)
	if err != nil {
		return nil, err
	}
	md := goldmark.New(goldmark.WithExtensions(&frontmatter.Extender{}))
	pCtx := parser.NewContext()
	if err := md.Convert(raw, io.Discard, parser.WithContext(pCtx)); err != nil {
		return nil, err
	}
	var info MeetupInfo
	if err := frontmatter.Get(pCtx).Decode(&info); err != nil {
		return nil, err
	}
	if info.ID == "" {
		return nil, nil
	}
	return &info, nil
}

type RemoteMeetupData struct {
	ID       string `json:"id"`
	DateTime string `json:"dateTime"`
}

type PastQueryResponse struct {
	GroupByURLName struct {
		Name       string `json:"name"`
		PastEvents struct {
			PageInfo struct {
				HasNextPage bool   `json:"hasNextPage"`
				EndCursor   string `json:"endCursor"`
			} `json:"pageInfo"`
			Edges []struct {
				Node RemoteMeetupData `json:"node"`
			} `json:"edges"`
		} `json:"pastEvents"`
	} `json:"groupByUrlname"`
}

type UpcomingQueryResponse struct {
	GroupByURLName struct {
		Name           string `json:"name"`
		UpcomingEvents struct {
			PageInfo struct {
				HasNextPage bool   `json:"hasNextPage"`
				EndCursor   string `json:"endCursor"`
			} `json:"pageInfo"`
			Edges []struct {
				Node RemoteMeetupData `json:"node"`
			} `json:"edges"`
		} `json:"upcomingEvents"`
	} `json:"groupByUrlname"`
}

func fetchPastMeetupData(ctx context.Context, groupName string) ([]RemoteMeetupData, error) {
	pastQuery := `
query($eventsCursor: String, $groupName: String!) {
  groupByUrlname(urlname: $groupName) {
	name
    pastEvents(input: {first: 50, after:$eventsCursor}) {
      edges {
        node {
          id
          dateTime
        }
      }
      pageInfo {
        hasNextPage
        endCursor
      }
    }
  }
}
`
	result := make([]RemoteMeetupData, 0, 10)
	client := graphql.NewClient("https://api.meetup.com/gql")
	cursor := ""
	for {
		req := graphql.NewRequest(pastQuery)
		req.Var("groupName", groupName)
		if cursor != "" {
			req.Var("eventsCursor", cursor)
		}
		var resp PastQueryResponse
		if err := client.Run(ctx, req, &resp); err != nil {
			return nil, err
		}
		for _, evt := range resp.GroupByURLName.PastEvents.Edges {
			result = append(result, evt.Node)
		}
		if !resp.GroupByURLName.PastEvents.PageInfo.HasNextPage {
			break
		}
		cursor = resp.GroupByURLName.PastEvents.PageInfo.EndCursor
	}
	return result, nil
}
func fetchUpcomingMeetupData(ctx context.Context, groupName string) ([]RemoteMeetupData, error) {
	pastQuery := `
query($eventsCursor: String, $groupName: String!) {
  groupByUrlname(urlname: $groupName) {
	name
    upcomingEvents(input: {first: 50, after:$eventsCursor}) {
      edges {
        node {
          id
          dateTime
        }
      }
      pageInfo {
        hasNextPage
        endCursor
      }
    }
  }
}
`
	result := make([]RemoteMeetupData, 0, 10)
	client := graphql.NewClient("https://api.meetup.com/gql")
	cursor := ""
	for {
		req := graphql.NewRequest(pastQuery)
		req.Var("groupName", groupName)
		if cursor != "" {
			req.Var("eventsCursor", cursor)
		}
		var resp UpcomingQueryResponse
		if err := client.Run(ctx, req, &resp); err != nil {
			return nil, err
		}
		for _, evt := range resp.GroupByURLName.UpcomingEvents.Edges {
			result = append(result, evt.Node)
		}
		if !resp.GroupByURLName.UpcomingEvents.PageInfo.HasNextPage {
			break
		}
		cursor = resp.GroupByURLName.UpcomingEvents.PageInfo.EndCursor
	}
	return result, nil
}
