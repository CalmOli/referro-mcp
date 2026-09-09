package referro

import "encoding/json"

// The structs below mirror the JSON returned by refero.design's public API
// (api.refero.design/v1). Only the fields the MCP tools surface are modeled;
// unknown fields are ignored by encoding/json.

// APIPagination is the pagination envelope every list endpoint returns.
type APIPagination struct {
	Current  int `json:"current"`
	Previous any `json:"previous"`
	Next     any `json:"next"`
	PerPage  int `json:"per_page"`
	Pages    int `json:"pages"`
	Count    int `json:"count"`
}

// Site is a website product that has captured screenshots.
type Site struct {
	ID          int    `json:"id"`
	Domain      string `json:"domain"`
	Name        string `json:"name"`
	Description string `json:"description"`
	FaviconURL  string `json:"favicon_url"`
	Categories  []struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	} `json:"categories"`
	ScreenshotsCount int `json:"screenshots_count"`
}

// App is an iOS app that has captured screenshots.
type App struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	StoreURL    string `json:"store_url"`
	IconURL     string `json:"icon_url"`
	Screenshots int    `json:"screenshots_count"`
}

// Screen is a single captured screenshot of a site or app page.
type Screen struct {
	ID           int        `json:"id"`
	UUID         string     `json:"uuid"`
	Width        int        `json:"width"`
	Height       int        `json:"height"`
	URL          StringList `json:"url"`
	ThumbnailURL string     `json:"thumbnail_url"`
	PreviewURL   string     `json:"preview_url"`
	PreviewFull  string     `json:"preview_full_url"`
	SingleScreen bool       `json:"single_screen"`
	PageURL      string     `json:"page_url"`
	CreatedAt    string     `json:"created_at"`
	Colors       [][]int    `json:"colors"`
	Site         *Site      `json:"site"`
	App          *App       `json:"app"`
	SiteID       int        `json:"site_id"`
	AppID        int        `json:"app_id"`
	PageTypes    []Labeled  `json:"page_types"`
	Patterns     []Labeled  `json:"design_patterns"`
	PageElements []Labeled  `json:"page_elements"`
	Fonts        FontList   `json:"fonts"`
	FlowIDs      []int      `json:"flow_ids"`
}

// StringList accepts either a single JSON string or an array of strings, since
// refero's API returns url as a string on some endpoints and an array on others.
type StringList []string

func (s *StringList) UnmarshalJSON(b []byte) error {
	if len(b) == 0 || string(b) == "null" {
		*s = nil
		return nil
	}
	if b[0] == '"' {
		var one string
		if err := json.Unmarshal(b, &one); err != nil {
			return err
		}
		*s = []string{one}
		return nil
	}
	var many []string
	if err := json.Unmarshal(b, &many); err != nil {
		return err
	}
	*s = many
	return nil
}

// FontList accepts either an array of strings or an array of font objects,
// since refero's API returns fonts in both shapes depending on the endpoint.
type FontList []string

func (f *FontList) UnmarshalJSON(b []byte) error {
	if len(b) == 0 || string(b) == "null" {
		*f = nil
		return nil
	}
	if b[0] == '[' && len(b) > 1 && b[1] == '"' {
		var many []string
		if err := json.Unmarshal(b, &many); err != nil {
			return err
		}
		*f = many
		return nil
	}
	// Object form: [{id, name, display_name}, ...]
	var objs []struct {
		ID          int    `json:"id"`
		Name        string `json:"name"`
		DisplayName string `json:"display_name"`
	}
	if err := json.Unmarshal(b, &objs); err != nil {
		return err
	}
	var names []string
	for _, o := range objs {
		n := o.DisplayName
		if n == "" {
			n = o.Name
		}
		if n != "" {
			names = append(names, n)
		}
	}
	*f = names
	return nil
}

// Labeled is a named facet (page type, design pattern, element) on a screen.
type Labeled struct {
	ID   int    `json:"id"`
	Kind string `json:"kind"`
	Name string `json:"name"`
}

// ScreenSearchResult is one record from /v1/search.
type ScreenSearchResult struct {
	Screen
}

// AppSearchResult is one record from /v1/search/apps.
type AppSearchResult struct {
	Screen
}

// ScreenSearchResponse is the /v1/search response envelope.
type ScreenSearchResponse struct {
	Pagination APIPagination `json:"pagination"`
	Records    []Screen      `json:"records"`
}

// AppSearchResponse is the /v1/search/apps response envelope.
type AppSearchResponse struct {
	Pagination APIPagination `json:"pagination"`
	Records    []Screen      `json:"records"`
}

// Flow is a multi-step user journey (checkout, onboarding, reset, etc.).
type Flow struct {
	ID          int        `json:"id"`
	Name        string     `json:"name"`
	Description any        `json:"description"`
	Site        *Site      `json:"site"`
	App         *App       `json:"app"`
	Steps       []FlowStep `json:"screenshots"`
}

// FlowStep is one step (screen) within a flow.
type FlowStep struct {
	ID           int      `json:"id"`
	UUID         string   `json:"uuid"`
	URL          []string `json:"url"`
	ThumbnailURL string   `json:"thumbnail_url"`
	PreviewURL   string   `json:"preview_url"`
}

// DesignPattern is a selectable design pattern (page state, element, etc.).
type DesignPattern struct {
	ID          int      `json:"id"`
	Kind        string   `json:"kind"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Aliases     []string `json:"aliases"`
}
