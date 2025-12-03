package discovery

import (
	"context"
	"io"
	"mime"
	"net/http"
	"net/url"
	"path"
	"sort"
	"strconv"
	"strings"

	"faviconsvc/internal/fetch"
	"faviconsvc/internal/security"
	"faviconsvc/pkg/logger"

	"golang.org/x/net/html"
)

type IconCandidate struct {
	URL        string
	Type       string
	Sizes      []int
	SizeScore  int
	FormatRank int
	RelRank    int
}

func DiscoverFromPageThenRoot(ctx context.Context, pageURL *url.URL, targetSize int) []IconCandidate {
	cands := collectPageIcons(ctx, pageURL, targetSize)

	// Add fallback root paths
	rootHTTPS := "https://" + pageURL.Host + "/favicon.ico"
	rootHTTP := "http://" + pageURL.Host + "/favicon.ico"

	if pageURL.Scheme == "https" {
		cands = append(cands, IconCandidate{URL: rootHTTPS, RelRank: 3})
		cands = append(cands, IconCandidate{URL: rootHTTP, RelRank: 3})
	} else {
		cands = append(cands, IconCandidate{URL: rootHTTP, RelRank: 3})
		cands = append(cands, IconCandidate{URL: rootHTTPS, RelRank: 3})
	}

	// Sort by priority
	sort.SliceStable(cands, func(i, j int) bool {
		if cands[i].RelRank != cands[j].RelRank {
			return cands[i].RelRank < cands[j].RelRank
		}
		if cands[i].FormatRank != cands[j].FormatRank {
			return cands[i].FormatRank < cands[j].FormatRank
		}
		return cands[i].SizeScore < cands[j].SizeScore
	})

	// Deduplicate
	uniq := make(map[string]struct{})
	out := make([]IconCandidate, 0, len(cands))
	for _, c := range cands {
		k := CanonicalizeURLString(c.URL)
		if _, ok := uniq[k]; ok {
			continue
		}
		uniq[k] = struct{}{}
		c.URL = k
		out = append(out, c)
	}

	logger.Debug("Discovered %d icon candidates for %s", len(out), pageURL.String())
	return out
}

func collectPageIcons(ctx context.Context, pageURL *url.URL, targetSize int) []IconCandidate {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, pageURL.String(), nil)
	if err != nil {
		logger.Warn("Failed to create request for %s: %v", pageURL.String(), err)
		return nil
	}
	req.Header.Set("User-Agent", fetch.UABrowser)
	req.Header.Set("Accept", "text/html,*/*;q=0.8")

	resp, err := fetch.HTTPClient.Do(req)
	if err != nil {
		logger.Warn("Failed to fetch HTML for %s: %v", pageURL.String(), err)
		return nil
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		logger.Warn("Got status %d for HTML fetch of %s", resp.StatusCode, pageURL.String())
		return nil
	}

	lr := io.LimitReader(resp.Body, fetch.MaxHTMLBytes)
	root, err := html.Parse(lr)
	if err != nil {
		logger.Warn("Failed to parse HTML for %s: %v", pageURL.String(), err)
		return nil
	}

	var baseHref *url.URL
	baseURL := pageURL
	var out []IconCandidate

	var f func(*html.Node)
	f = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "base" {
			for _, a := range n.Attr {
				if strings.EqualFold(a.Key, "href") {
					if bu, err := url.Parse(strings.TrimSpace(a.Val)); err == nil {
						baseHref = pageURL.ResolveReference(bu)
					}
				}
			}
		}

		if n.Type == html.ElementNode && n.Data == "link" {
			var rel, href, typ, sizesAttr string
			for _, a := range n.Attr {
				switch strings.ToLower(a.Key) {
				case "rel":
					rel = strings.ToLower(strings.TrimSpace(a.Val))
				case "href":
					href = strings.TrimSpace(a.Val)
				case "type":
					typ = strings.ToLower(strings.TrimSpace(a.Val))
				case "sizes":
					sizesAttr = strings.ToLower(strings.TrimSpace(a.Val))
				}
			}

			if href != "" && rel != "" {
				rtoks := strings.Fields(rel)
				hasIcon := false
				isApple := false
				for _, t := range rtoks {
					switch t {
					case "icon":
						hasIcon = true
					case "apple-touch-icon", "apple-touch-icon-precomposed":
						isApple = true
					}
				}
				if strings.Contains(rel, "shortcut icon") {
					hasIcon = true
				}
				if strings.Contains(rel, "apple-touch-icon") {
					isApple = true
				}

				if hasIcon || isApple {
					base := baseURL
					if baseHref != nil {
						base = baseHref
					}
					if ru, err := url.Parse(href); err == nil {
						resolvedURL := base.ResolveReference(ru)
						if !security.IsAllowedScheme(resolvedURL) {
							goto NEXT
						}
						resolved := resolvedURL.String()
						edgeSizes, any := parseSizes(sizesAttr)
						score := computeSizeScore(edgeSizes, any, targetSize)
						formatRank := formatPreference(typ, resolved)
						relRank := 1
						if isApple && !hasIcon {
							relRank = 2
						}
						out = append(out, IconCandidate{
							URL:        resolved,
							Type:       typ,
							Sizes:      edgeSizes,
							SizeScore:  score,
							FormatRank: formatRank,
							RelRank:    relRank,
						})
					}
				}
			}
		}
	NEXT:
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			f(c)
		}
	}
	f(root)

	return out
}

func parseSizes(attr string) (edges []int, any bool) {
	if attr == "" {
		return nil, false
	}
	if attr == "any" {
		return nil, true
	}
	for _, p := range strings.Fields(attr) {
		xy := strings.Split(p, "x")
		if len(xy) == 2 {
			if w, err := strconv.Atoi(xy[0]); err == nil {
				edges = append(edges, w)
			}
		}
	}
	return edges, false
}

func computeSizeScore(edges []int, any bool, target int) int {
	if any || len(edges) == 0 {
		return 10000
	}
	best := int(^uint(0) >> 1)
	for _, e := range edges {
		if d := abs(e - target); d < best {
			best = d
		}
	}
	return best
}

func formatPreference(typ, resolved string) int {
	ext := strings.ToLower(path.Ext(resolved))
	ct, _, _ := mime.ParseMediaType(typ)
	if ct == "image/svg+xml" || ext == ".svg" {
		return 2
	}
	if ct == "image/png" || ext == ".png" || ct == "image/x-icon" || ext == ".ico" || ct == "image/webp" || ext == ".webp" || ct == "image/avif" || ext == ".avif" {
		return 0
	}
	return 1
}

// CanonicalizeURLString normalizes a URL string for consistent comparison.
// It removes fragments, normalizes scheme and host, cleans paths, and sorts query parameters.
func CanonicalizeURLString(raw string) string {
	u, err := url.Parse(raw)
	if err != nil {
		return raw
	}
	u.Fragment = ""
	u.Scheme = strings.ToLower(u.Scheme)
	h := strings.ToLower(u.Hostname())
	port := u.Port()
	if (u.Scheme == "http" && port == "80") || (u.Scheme == "https" && port == "443") {
		port = ""
	}
	if port != "" {
		u.Host = h + ":" + port
	} else {
		u.Host = h
	}
	if u.Path == "" {
		u.Path = "/"
	}
	u.Path = path.Clean(u.Path)
	if u.RawQuery != "" {
		q, _ := url.ParseQuery(u.RawQuery)
		keys := make([]string, 0, len(q))
		for k := range q {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		var parts []string
		for _, k := range keys {
			vals := q[k]
			sort.Strings(vals)
			for _, v := range vals {
				parts = append(parts, url.QueryEscape(k)+"="+url.QueryEscape(v))
			}
		}
		u.RawQuery = strings.Join(parts, "&")
	}
	return u.String()
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

func IsICO(contentType, srcURL string) bool {
	ct, _, _ := mime.ParseMediaType(contentType)
	if ct == "image/x-icon" || ct == "image/vnd.microsoft.icon" {
		return true
	}
	return strings.HasSuffix(strings.ToLower(srcURL), ".ico")
}

func IsSVGContentType(contentType, srcURL string) bool {
	ct, _, _ := mime.ParseMediaType(contentType)
	if ct == "image/svg+xml" {
		return true
	}
	return strings.EqualFold(path.Ext(srcURL), ".svg")
}

func LooksLikeHTML(b []byte, contentType string) bool {
	if contentType != "" {
		ct, _, _ := mime.ParseMediaType(contentType)
		if strings.Contains(ct, "html") {
			return true
		}
	}
	s := strings.TrimSpace(strings.ToLower(string(peek512(b))))
	return strings.HasPrefix(s, "<!doctype html") || strings.HasPrefix(s, "<html")
}

func peek512(b []byte) []byte {
	if len(b) > 512 {
		return b[:512]
	}
	return b
}
