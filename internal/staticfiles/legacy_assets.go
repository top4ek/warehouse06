package staticfiles

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

var indexScriptRe = regexp.MustCompile(`src="/assets/(index-[^"?]+\.js)`)

// CurrentIndexAsset returns the absolute path to the main bundle named in dist/index.html.
func CurrentIndexAsset(frontendDist string) (string, bool) {
	htmlPath := filepath.Join(frontendDist, "index.html")
	b, err := os.ReadFile(htmlPath)
	if err != nil {
		return "", false
	}
	m := indexScriptRe.FindSubmatch(b)
	if len(m) < 2 {
		return "", false
	}
	p := filepath.Join(frontendDist, "assets", string(m[1]))
	if st, err := os.Stat(p); err != nil || st.IsDir() {
		return "", false
	}
	return p, true
}

// ResolveLegacyAsset maps a missing hashed asset URL to the current build artifact.
func ResolveLegacyAsset(assetsDir, requested string) (resolved string, ok bool) {
	direct := filepath.Join(assetsDir, requested)
	if st, err := os.Stat(direct); err == nil && !st.IsDir() {
		return direct, true
	}

	ext := filepath.Ext(requested)
	base := strings.TrimSuffix(requested, ext)
	dash := strings.LastIndex(base, "-")
	if dash <= 0 {
		return "", false
	}
	prefix := base[:dash]

	if prefix == "Entry" {
		p := filepath.Join(assetsDir, "entry.js")
		if st, err := os.Stat(p); err == nil && !st.IsDir() {
			return p, true
		}
	}

	entries, err := os.ReadDir(assetsDir)
	if err != nil {
		return "", false
	}

	var best string
	var bestMod time.Time
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if name == requested || !strings.HasSuffix(name, ext) {
			continue
		}
		if !strings.HasPrefix(name, prefix+"-") && (prefix != "Entry" || name != "entry.js") {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		if best == "" || info.ModTime().After(bestMod) {
			best = filepath.Join(assetsDir, name)
			bestMod = info.ModTime()
		}
	}
	if best != "" {
		return best, true
	}
	return "", false
}
