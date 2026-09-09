package referro

// DesignType is the kind of artifact a design describes.
type DesignType string

const (
	TypeWebsite  DesignType = "website"
	TypeMobile   DesignType = "mobile"
	TypeUI       DesignType = "ui"
	TypeUnknown  DesignType = "unknown"
)

// Design is a design artifact on referro.design: a markdown spec plus the
// images and workflow that describe it.
type Design struct {
	ID          string     `json:"id"`
	Title       string     `json:"title"`
	Type        DesignType `json:"type"`
	URL         string     `json:"url"`
	Description string     `json:"description"`
	// MD is the raw markdown content of the design's spec file.
	MD string `json:"md,omitempty"`
}

// Image is an image asset associated with a design.
type Image struct {
	URL     string `json:"url"`
	Alt     string `json:"alt,omitempty"`
	Caption string `json:"caption,omitempty"`
}

// Step is one step in a design's workflow.
type Step struct {
	Order       int    `json:"order"`
	Title       string `json:"title"`
	Description string `json:"description,omitempty"`
}

// Workflow is the ordered set of steps that describe how a design is produced.
type Workflow struct {
	DesignID string `json:"design_id"`
	Title    string `json:"title,omitempty"`
	Steps    []Step `json:"steps"`
}
