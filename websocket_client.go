package plex

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"time"

	"github.com/gorilla/websocket"
)

const (
	writeWait  = time.Second * 10
	pongWait   = time.Second * 10
	pingPeriod = time.Second * 5
)

// TimelineEntry ...
type TimelineEntry struct {
	Identifier    string `json:"identifier"`
	ItemID        string `json:"itemID"`
	ParentItemID  string `json:"parentItemID"`
	RootItemID    string `json:"rootItemID"`
	MetadataState string `json:"metadataState"`
	SectionID     string `json:"sectionID"`
	State         int64  `json:"state"`
	Title         string `json:"title"`
	Type          int64  `json:"type"`
	UpdatedAt     int64  `json:"updatedAt"`
}

// ActivityNotification ...
type ActivityNotification struct {
	Activity struct {
		Cancellable bool   `json:"cancellable"`
		Progress    int64  `json:"progress"`
		Subtitle    string `json:"subtitle"`
		Title       string `json:"title"`
		Type        string `json:"type"`
		UserID      int64  `json:"userID"`
		UUID        string `json:"uuid"`
	} `json:"Activity"`
	Event string `json:"event"`
	UUID  string `json:"uuid"`
}

// StatusNotification ...
type StatusNotification struct {
	Description      string `json:"description"`
	NotificationName string `json:"notificationName"`
	Title            string `json:"title"`
}

// PlaySessionStateNotification ...
type PlaySessionStateNotification struct {
	GUID             string `json:"guid"`
	Key              string `json:"key"`
	PlayQueueItemID  int64  `json:"playQueueItemID"`
	RatingKey        string `json:"ratingKey"`
	SessionKey       string `json:"sessionKey"`
	State            string `json:"state"`
	URL              string `json:"url"`
	ViewOffset       int64  `json:"viewOffset"`
	TranscodeSession string `json:"transcodeSession"`
}

// ReachabilityNotification ...
type ReachabilityNotification struct {
	Reachability bool `json:"reachability"`
}

// BackgroundProcessingQueueEventNotification ...
type BackgroundProcessingQueueEventNotification struct {
	Event   string `json:"event"`
	QueueID int64  `json:"queueID"`
}

// TranscodeSession ...
type TranscodeSession struct {
	AudioChannels        int64   `json:"audioChannels"`
	AudioCodec           string  `json:"audioCodec"`
	AudioDecision        string  `json:"audioDecision"`
	Complete             bool    `json:"complete"`
	Container            string  `json:"container"`
	Context              string  `json:"context"`
	Duration             int64   `json:"duration"`
	Key                  string  `json:"key"`
	Progress             float64 `json:"progress"`
	Protocol             string  `json:"protocol"`
	Remaining            int64   `json:"remaining"`
	SourceAudioCodec     string  `json:"sourceAudioCodec"`
	SourceVideoCodec     string  `json:"sourceVideoCodec"`
	Speed                float64 `json:"speed"`
	Throttled            bool    `json:"throttled"`
	TranscodeHwRequested bool    `json:"transcodeHwRequested"`
	VideoCodec           string  `json:"videoCodec"`
	VideoDecision        string  `json:"videoDecision"`
}

type ConfigValue string

func (c *ConfigValue) UnmarshalJSON(data []byte) error {
	const quote = rune('"')
	if len(data) >= 2 && rune(data[0]) == quote && rune(data[len(data)-1]) == quote {
		*c = ConfigValue(data[1 : len(data)-1])
	} else {
		*c = ConfigValue(data)
	}
	return nil
}

func (c ConfigValue) String() string {
	return string(c)
}

func (c ConfigValue) Boolean() bool {
	if value, err := strconv.ParseBool(string(c)); err == nil {
		return value
	}
	return false
}

func (c ConfigValue) Int() int {
	if value, err := strconv.Atoi(string(c)); err == nil {
		return value
	}
	return 0
}

func (c ConfigValue) Float64() float64 {
	if value, err := strconv.ParseFloat(string(c), 64); err == nil {
		return value
	}
	return 0
}

// Setting ...
type Setting struct {
	Advanced bool        `json:"advanced"`
	Default  ConfigValue `json:"default"`
	Group    string      `json:"group"`
	Hidden   bool        `json:"hidden"`
	ID       string      `json:"id"`
	Label    string      `json:"label"`
	Summary  string      `json:"summary"`
	Type     string      `json:"type"`
	Value    ConfigValue `json:"value"`
}

// NotificationContainer read pms notifications
type NotificationContainer struct {
	TimelineEntry []TimelineEntry `json:"TimelineEntry"`

	ActivityNotification []ActivityNotification `json:"ActivityNotification"`

	StatusNotification []StatusNotification `json:"StatusNotification"`

	PlaySessionStateNotification []PlaySessionStateNotification `json:"PlaySessionStateNotification"`

	ReachabilityNotification []ReachabilityNotification `json:"ReachabilityNotification"`

	BackgroundProcessingQueueEventNotification []BackgroundProcessingQueueEventNotification `json:"BackgroundProcessingQueueEventNotification"`

	TranscodeSession []TranscodeSession `json:"TranscodeSession"`

	Setting []Setting `json:"Setting"`

	Size int64 `json:"size"`
	// Type can be one of:
	// playing,
	// reachability,
	// transcode.end,
	// preference,
	// update.statechange,
	// activity,
	// backgroundProcessingQueue,
	// transcodeSession.update
	// transcodeSession.end
	Type string `json:"type"`
}

// WebsocketNotification websocket payload of notifications from a plex media server
type WebsocketNotification struct {
	NotificationContainer `json:"NotificationContainer"`
}

// NotificationEvents hold callbacks that correspond to notifications
type NotificationEvents struct {
	events map[string]func(n NotificationContainer)
}

// NewNotificationEvents initializes the event callbacks
func NewNotificationEvents() *NotificationEvents {
	return &NotificationEvents{
		events: map[string]func(n NotificationContainer){
			"account":                   func(n NotificationContainer) {},
			"activity":                  func(n NotificationContainer) {},
			"backgroundProcessingQueue": func(n NotificationContainer) {},
			"playing":                   func(n NotificationContainer) {},
			"preference":                func(n NotificationContainer) {},
			"progress":                  func(n NotificationContainer) {},
			"reachability":              func(n NotificationContainer) {},
			"status":                    func(n NotificationContainer) {},
			"timeline":                  func(n NotificationContainer) {},
			"transcode.end":             func(n NotificationContainer) {},
			"transcodeSession.end":      func(n NotificationContainer) {},
			"transcodeSession.start":    func(n NotificationContainer) {},
			"transcodeSession.update":   func(n NotificationContainer) {},
			"update.statechange":        func(n NotificationContainer) {},
		},
	}
}

// OnActivity shows activities happened on plex
func (e *NotificationEvents) OnActivity(fn func(n NotificationContainer)) {
	e.events["activity"] = fn
}

// OnPlaying shows state information (resume, stop, pause) on a user consuming media in plex
func (e *NotificationEvents) OnPlaying(fn func(n NotificationContainer)) {
	e.events["playing"] = fn
}

// OnTimeline shows recently added/deleted items in plex libraries
func (e *NotificationEvents) OnTimeline(fn func(n NotificationContainer)) {
	e.events["timeline"] = fn
}

// OnTranscodeUpdate shows transcode information when a transcoding stream changes parameters
func (e *NotificationEvents) OnTranscodeUpdate(fn func(n NotificationContainer)) {
	e.events["transcodeSession.update"] = fn
}

// SubscribeToNotifications connects to your server via websockets listening for events
func (p *Plex) SubscribeToNotifications(events *NotificationEvents, interrupt <-chan os.Signal, fn func(error)) {
	plexURL, err := url.Parse(p.URL)

	if err != nil {
		fn(err)
		return
	}

	var scheme string
	if plexURL.Scheme == "https" {
		scheme = "wss"
	} else {
		scheme = "ws"
	}
	websocketURL := url.URL{Scheme: scheme, Host: plexURL.Host, Path: "/:/websockets/notifications"}

	headers := http.Header{
		"X-Plex-Token": []string{p.Token},
	}

	c, _, err := websocket.DefaultDialer.Dial(websocketURL.String(), headers)

	if err != nil {
		fn(err)
		return
	}

	go func() {
		defer c.Close()

		_ = c.SetReadDeadline(time.Now().Add(pongWait))
		c.SetPongHandler(func(string) error {
			_ = c.SetReadDeadline(time.Now().Add(pongWait))
			return nil
		})

		for {
			_, message, err := c.ReadMessage()

			if err != nil {
				fn(err)
				return
			}

			var notif WebsocketNotification

			if err := json.Unmarshal(message, &notif); err != nil {
				fmt.Printf("convert message to json failed: %v\n", err)
				continue
			}

			fn, ok := events.events[notif.Type]

			if !ok {
				fmt.Printf("unknown websocket event name: %v\n", notif.Type)
				continue
			}

			fn(notif.NotificationContainer)
		}
	}()

	go func() {
		ticker := time.NewTicker(pingPeriod)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				_ = c.SetWriteDeadline(time.Now().Add(writeWait))
				if err := c.WriteMessage(websocket.PingMessage, []byte("Hi?")); err != nil {
					return
				}
			case <-interrupt:
				// To cleanly close a connection, a client should send a close
				// frame and wait for the server to close the connection.
				_ = c.SetWriteDeadline(time.Now().Add(writeWait))
				_ = c.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
		}
	}()
}
