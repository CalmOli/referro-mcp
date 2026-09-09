package referro

import (
	"context"
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

// designLinkRe matches hrefs that look like design spec pages. referro.design
// has no public API, so we heuristically recognize design URLs: paths ending in
// .md, or containing design/artifact-ish segments. Kept as a package var so it
// is easy to tune for the live site.
var designLinkRe = regexp.MustCompile(`(?i)(/designs?/|/artifacts?/|/specs?/|\.md(/|$)|/workflows?/)`)

// ListDesigns fetches the index (base URL) and returns the designs it links to.
// Each returned Design carries its ID (derived from the URL), title, and type;
// the heavier fields (MD, images, workflow) are filled in by GetDesign.
func (c *Client) ListDesigns(ctx context.Context) ([]Design, error) {
	f, err := c.fetch(ctx, "/")
	if err != nil {
		return nil, err
	}
	return c.parseIndex(f)
}

func (c *Client) parseIndex(f *fetched) ([]Design, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(f.body)))
	if err != nil {
		return nil, fmt.Errorf("parse index: %w", err)
	}

	seen := map[string]bool{}
	var designs []Design

	doc.Find("a[href]").Each(func(_ int, s *goquery.Selection) {
		href, _ := s.Attr("href")
		if href == "" || !designLinkRe.MatchString(href) {
			return
		}
		abs := c.Resolve(href)
		if seen[abs] {
			return
		}
		seen[abs] = true

		title := strings.TrimSpace(s.Text())
		if title == "" {
			title = titleFromPath(abs)
		}
		designs = append(designs, Design{
			ID:    idFromURL(abs),
			Title: title,
			Type:  classifyType(abs, title),
			URL:   abs,
		})
	})

	sort.Slice(designs, func(i, j int) bool { return designs[i].ID < designs[j].ID })
	return designs, nil
}

// resolveRef turns a caller-supplied reference into a fetchable path. It
// accepts a full URL, an absolute path, or a bare design ID (e.g. "landing-page")
// which it resolves through the index so agents can pass the IDs they got from
// list_designs.
func (c *Client) resolveRef(ctx context.Context, ref string) (string, error) {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return "", fmt.Errorf("empty reference")
	}
	if strings.HasPrefix(ref, "http://") || strings.HasPrefix(ref, "https://") || strings.HasPrefix(ref, "/") {
		return ref, nil
	}
	designs, err := c.ListDesigns(ctx)
	if err != nil {
		return "", err
	}
	for _, d := range designs {
		if d.ID == ref {
			return d.URL, nil
		}
	}
	return "", fmt.Errorf("no design with id %q", ref)
}

// GetDesign fetches a design page (by URL or ID) and extracts its markdown,
// images, and workflow. If the page is raw markdown it is used as-is; if it is
// HTML the markdown-ish text and linked assets are extracted.
func (c *Client) GetDesign(ctx context.Context, ref string) (*Design, error) {
	path, err := c.resolveRef(ctx, ref)
	if err != nil {
		return nil, err
	}
	f, err := c.fetch(ctx, path)
	if err != nil {
		return nil, err
	}

	abs := f.url
	d := &Design{
		ID:    idFromURL(abs),
		Title: titleFromPath(abs),
		Type:  classifyType(abs, ""),
		URL:   abs,
	}

	ctype := strings.ToLower(f.ctype)
	if strings.Contains(ctype, "markdown") || strings.HasSuffix(strings.ToLower(abs), ".md") {
		d.MD = string(f.body)
		return d, nil
	}

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(f.body)))
	if err != nil {
		return nil, fmt.Errorf("parse design: %w", err)
	}

	if t := strings.TrimSpace(doc.Find("title").First().Text()); t != "" {
		d.Title = t
	} else if h1 := strings.TrimSpace(doc.Find("h1").First().Text()); h1 != "" {
		d.Title = h1
	}
	if desc, _ := doc.Find(`meta[name="description"]`).Attr("content"); desc != "" {
		d.Description = strings.TrimSpace(desc)
	}

	// Markdown: prefer a linked .md file, else the visible text of the main
	// content region.
	if mdHref := firstLinkedMD(doc); mdHref != "" {
		if mf, err := c.fetch(ctx, mdHref); err == nil {
			d.MD = string(mf.body)
		}
	}
	if d.MD == "" {
		d.MD = extractMarkdown(doc)
	}

	return d, nil
}

// GetImages returns the image assets found on a design page.
func (c *Client) GetImages(ctx context.Context, ref string) ([]Image, error) {
	path, err := c.resolveRef(ctx, ref)
	if err != nil {
		return nil, err
	}
	f, err := c.fetch(ctx, path)
	if err != nil {
		return nil, err
	}
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(f.body)))
	if err != nil {
		return nil, fmt.Errorf("parse images: %w", err)
	}
	return extractImages(doc, c), nil
}

// GetWorkflow returns the workflow steps found on a design page.
func (c *Client) GetWorkflow(ctx context.Context, ref string) (*Workflow, error) {
	path, err := c.resolveRef(ctx, ref)
	if err != nil {
		return nil, err
	}
	f, err := c.fetch(ctx, path)
	if err != nil {
		return nil, err
	}
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(f.body)))
	if err != nil {
		return nil, fmt.Errorf("parse workflow: %w", err)
	}
	return extractWorkflow(doc, idFromURL(f.url)), nil
}

// --- helpers ---

func idFromURL(u string) string {
	u = strings.TrimSuffix(u, "/")
	segs := strings.Split(u, "/")
	// Walk back from the end, skipping file-ish segments (index.html, .md)
	// to find the design's own segment.
	for i := len(segs) - 1; i >= 0; i-- {
		seg := segs[i]
		low := strings.ToLower(seg)
		if low == "index.html" || low == "index" || strings.HasSuffix(low, ".md") {
			continue
		}
		seg = strings.TrimSuffix(seg, ".html")
		if seg != "" {
			return seg
		}
	}
	return "index"
}

func titleFromPath(u string) string {
	return strings.ReplaceAll(idFromURL(u), "-", " ")
}

func classifyType(u, title string) DesignType {
	low := strings.ToLower(u + " " + title)
	switch {
	case strings.Contains(low, "website") || strings.Contains(low, "web") || strings.Contains(low, "landing"):
		return TypeWebsite
	case strings.Contains(low, "mobile") || strings.Contains(low, "ios") || strings.Contains(low, "android"):
		return TypeMobile
	case strings.Contains(low, "ui") || strings.Contains(low, "interface") || strings.Contains(low, "component"):
		return TypeUI
	default:
		return TypeUnknown
	}
}

func firstLinkedMD(doc *goquery.Document) string {
	var found string
	doc.Find("a[href]").EachWithBreak(func(_ int, s *goquery.Selection) bool {
		href, _ := s.Attr("href")
		if strings.HasSuffix(strings.ToLower(href), ".md") {
			found = href
			return false
		}
		return true
	})
	return found
}

// extractMarkdown renders the main content of an HTML page back to a loose
// markdown-ish text: headings as #, paragraphs as text, list items as "- ".
func extractMarkdown(doc *goquery.Document) string {
	var b strings.Builder
	sel := doc.Find("article, main, .content, .markdown, .design-body, body").First()
	if sel.Length() == 0 {
		sel = doc.Selection
	}
	sel.Find("h1,h2,h3,h4,p,li,pre").Each(func(_ int, s *goquery.Selection) {
		tag := goquery.NodeName(s)
		text := strings.TrimSpace(s.Text())
		if text == "" {
			return
		}
		switch tag {
		case "h1":
			b.WriteString("# " + text + "\n\n")
		case "h2":
			b.WriteString("## " + text + "\n\n")
		case "h3":
			b.WriteString("### " + text + "\n\n")
		case "h4":
			b.WriteString("#### " + text + "\n\n")
		case "li":
			b.WriteString("- " + text + "\n")
		case "pre":
			b.WriteString("```\n" + text + "\n```\n\n")
		default:
			b.WriteString(text + "\n\n")
		}
	})
	return strings.TrimSpace(b.String())
}

func extractImages(doc *goquery.Document, c *Client) []Image {
	seen := map[string]bool{}
	var imgs []Image
	doc.Find("img").Each(func(_ int, s *goquery.Selection) {
		src, _ := s.Attr("src")
		if src == "" {
			return
		}
		abs := c.Resolve(src)
		if seen[abs] {
			return
		}
		seen[abs] = true
		alt, _ := s.Attr("alt")
		imgs = append(imgs, Image{URL: abs, Alt: alt})
	})
	// og:image is the canonical hero image.
	if og, ok := doc.Find(`meta[property="og:image"]`).Attr("content"); ok && og != "" && !seen[og] {
		imgs = append([]Image{{URL: c.Resolve(og), Alt: "og:image"}}, imgs...)
	}
	return imgs
}

// extractWorkflow pulls ordered steps from an ordered list or from headings
// that look like steps ("Step 1:", "1.", etc.).
func extractWorkflow(doc *goquery.Document, designID string) *Workflow {
	w := &Workflow{DesignID: designID}
	if t := strings.TrimSpace(doc.Find("h2, h3").First().Text()); t != "" {
		w.Title = t
	}

	// Prefer an <ol>; fall back to headings that read like steps.
	ol := doc.Find("ol").First()
	if ol.Length() > 0 {
		ol.Find("li").Each(func(i int, s *goquery.Selection) {
			w.Steps = append(w.Steps, Step{
				Order:       i + 1,
				Title:       strings.TrimSpace(s.Text()),
				Description: "",
			})
		})
		return w
	}

	stepRe := regexp.MustCompile(`(?i)^(step\s*)?\d+[.:)]\s*(.*)$`)
	doc.Find("h3, h4, p").Each(func(_ int, s *goquery.Selection) {
		text := strings.TrimSpace(s.Text())
		m := stepRe.FindStringSubmatch(text)
		if m == nil {
			return
		}
		w.Steps = append(w.Steps, Step{
			Order: len(w.Steps) + 1,
			Title: strings.TrimSpace(m[2]),
		})
	})
	return w
}
