package main

import "encoding/json"
import "flag"
import "fmt"
import "net/http"
import "strings"
import "time"

type Feed struct {
	Response Response
}

type Response struct {
	FeedMessageResponse FeedMessageResponse
	Errors Error
}

type Error struct {
	Code        string
	Text        string
	Description string
}

type FeedMessageResponse struct {
	ActivityCount int
	Count         int
	TotalCount    int
	Feed          FeedMetadata
	Messages      Messages
}

type FeedMetadata struct {
	ID                   string
	Name                 string
	Description          string
	Status               string
	Usage                int
	DaysRange            int
	DetailedMessageShown bool
}

type Messages struct {
	Message      json.RawMessage // can be either []Message or Message
	MessageSlice []Message
}

type Message struct {
	ID            int
	UnixTime      int
	MessengerID   string
	MessengerName string
	ModelID       string
	MessageType   string
	Latitude      float64
	Longitude     float64
	BatteryState  string
}

type Fix struct {
	ID           int
	Time         time.Time
	MessengerID  string
	MessageType  string
	Position     Position
	BatteryState BatteryState
}

type Position struct {
	Latitude  float64
	Longitude float64
}

type BatteryState string

func panicIf(err error) {
	if err != nil {
		panic(err)
	}
}

func wait() {
	time.Sleep(3 * time.Minute)
}

func FormatTime(t time.Time) string {
	return strings.Replace(t.In(time.UTC).Format("2006-01-02T15:04:05-0700"), "+0000", "-0000", -1)
}

func Output(fixes []Fix) {
	for _, fix := range(fixes) {
		fmt.Printf("%s %d %s %s %+3.6f,%+3.6f %s\n", fix.Time, fix.ID, fix.MessengerID, fix.MessageType, fix.Position.Latitude, fix.Position.Longitude, fix.BatteryState)
	}
}

func LoadFeed(feedID string, after time.Time, before time.Time) Feed {
	url := "https://api.findmespot.com/spot-main-web/consumer/rest-api/2.0/public/feed/" +
		feedID + "/message.json?" +
		"startDate=" + FormatTime(after) +
		"&endDate=" + FormatTime(before)
	fmt.Println(" -- " + url)

	resp, err := http.Get(url)
	panicIf(err)
	defer resp.Body.Close()

	var data Feed
	err = json.NewDecoder(resp.Body).Decode(&data)
	panicIf(err)

	var msgs []Message
	err = json.Unmarshal(data.Response.FeedMessageResponse.Messages.Message, &msgs)
	if err != nil {
		var msg Message
		err = json.Unmarshal(data.Response.FeedMessageResponse.Messages.Message, &msg)
		if err != nil {
			// There feed seems to contain no messages at all.
			return data
		}
		msgs = append(msgs, msg)
	}
	data.Response.FeedMessageResponse.Messages.MessageSlice = msgs

	return data
}

func ParseChunk(feed Feed) (time.Time, time.Time, []Fix) {
	fixes := []Fix{}
	oldest := time.Now()
	newest := time.Unix(0, 0)
	for _, msg := range feed.Response.FeedMessageResponse.Messages.MessageSlice {
		fix := Fix {
			ID:           msg.ID,
			Time:         time.Unix(int64(msg.UnixTime), 0),
			MessengerID:  msg.MessengerID,
			MessageType:  msg.MessageType,
			Position:     Position {msg.Latitude, msg.Longitude},
			BatteryState: BatteryState(msg.BatteryState),
		}
		fixes = append(fixes, fix)
		if oldest.Sub(fix.Time) > 0 {
			oldest = fix.Time
		}
		if newest.Sub(fix.Time) < 0 {
			newest = fix.Time
		}
	}
	return oldest, newest, fixes
}

func LoadBackwards(feedID string) time.Time {
	oldest := time.Now()
	allTimeNewest := time.Unix(0, 0)
	var newest time.Time
	var fixes []Fix
	for {
		oldest, newest, fixes = ParseChunk(LoadFeed(feedID, time.Unix(0, 0), oldest))
		if len(fixes) == 0 {
			break
		}
		Output(fixes)
		if allTimeNewest.Sub(newest) < 0 {
			allTimeNewest = newest
		}
		wait()
		oldest = oldest.Add(-1 * time.Second)
	}
	return allTimeNewest
}

func BackfillAndPoll(feedID string) {
	newest := LoadBackwards(feedID).Add(1 * time.Second)
	for {
		wait()
		_, newNewest, fixes := ParseChunk(LoadFeed(feedID, newest, time.Now()))
		if len(fixes) > 0 {
			newest = newNewest.Add(1 * time.Second)
			Output(fixes)
		}
	}
}

func main() {
	flag.Parse()
	BackfillAndPoll(flag.Arg(0))
}
