package sse

import (
	"strconv"
	"strings"
)

type citationLinkCollector struct {
	ordered  []string
	seen     map[string]struct{}
	explicit map[int]string
}

func newCitationLinkCollector() *citationLinkCollector {
	return &citationLinkCollector{
		seen:     map[string]struct{}{},
		explicit: map[int]string{},
	}
}

func (c *citationLinkCollector) ingestChunk(chunk map[string]any) {
	if c == nil || len(chunk) == 0 {
		return
	}
	c.walkValue(chunk)
}

func (c *citationLinkCollector) build() map[int]string {
	out := make(map[int]string, len(c.explicit)+len(c.ordered))
	for idx, u := range c.explicit {
		if idx > 0 && strings.TrimSpace(u) != "" {
			out[idx] = u
		}
	}
	for i, u := range c.ordered {
		idx := i + 1
		if _, exists := out[idx]; !exists {
			out[idx] = u
		}
	}
	return out
}

func (c *citationLinkCollector) walkValue(v any) {
	switch x := v.(type) {
	case []any:
		for _, item := range x {
			c.walkValue(item)
		}
	case map[string]any:
		c.captureURLAndIndex(x)
		for _, vv := range x {
			c.walkValue(vv)
		}
	}
}

func (c *citationLinkCollector) captureURLAndIndex(m map[string]any) {
	url := strings.TrimSpace(asString(m["url"]))
	if !isWebURL(url) {
		return
	}
	c.addOrdered(url)

	idx, hasIdx := citationIndexFromAny(m["cite_index"])
	if !hasIdx {
		return
	}
	// DeepSeek citation indices in search results are zero-based (0,1,2,...),
	// while visible markers are one-based ([citation:1], [citation:2], ...).
	// Normalize all non-negative explicit indices to one-based to avoid
	// misalignment when 3+ citations are present.
	if idx >= 0 {
		idx = idx + 1
	}
	if idx <= 0 {
		return
	}
	if existing, ok := c.explicit[idx]; ok && strings.TrimSpace(existing) != "" {
		return
	}
	c.explicit[idx] = url
}

func (c *citationLinkCollector) addOrdered(url string) {
	if _, ok := c.seen[url]; ok {
		return
	}
	c.seen[url] = struct{}{}
	c.ordered = append(c.ordered, url)
}

func citationIndexFromAny(v any) (int, bool) {
	switch x := v.(type) {
	case int:
		return x, true
	case int32:
		return int(x), true
	case int64:
		return int(x), true
	case float32:
		return int(x), true
	case float64:
		return int(x), true
	case string:
		s := strings.TrimSpace(x)
		if s == "" {
			return 0, false
		}
		n, err := strconv.Atoi(s)
		if err != nil {
			return 0, false
		}
		return n, true
	default:
		return 0, false
	}
}

func isWebURL(v string) bool {
	v = strings.ToLower(strings.TrimSpace(v))
	return strings.HasPrefix(v, "http://") || strings.HasPrefix(v, "https://")
}

func asString(v any) string {
	s, _ := v.(string)
	return s
}
