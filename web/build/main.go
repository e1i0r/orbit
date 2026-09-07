// Command build writes the landing page, once per language.
//
// The page is one template and two catalogues rather than two pages, because
// the landing is edited often and a second copy is a copy that goes stale:
// the day a bullet is added in English and not in Spanish, the Spanish page
// still looks finished. Here there is one place to add it, and a page that
// asks for a sentence neither catalogue has does not build at all.
//
// It writes site/index.html and site/es/index.html, and both are committed —
// GitHub Pages serves site/ as it finds it, so nothing runs this at deploy
// time. TestTheCommittedPagesAreWhatTheTemplateSays is what keeps the
// committed files and the template from parting company.
package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"text/template"
)

// A language is one page: which catalogue it reads, where it is written, and
// what a link inside it has to reach back up through to find the videos.
//
// The videos, the posters and the icon are not duplicated per language —
// they are terminal recordings with no words of ours in them — so the
// translated page lives in a directory beside them and reaches up.
type language struct {
	code    string // what goes in <html lang> and in the pill
	catalog string // the file its sentences come from
	out     string // where the page is written, under the site directory
	assets  string // what a video's src is prefixed with
	url     string // the page's own address, for the canonical link
}

var languages = []language{
	{code: "en", catalog: "en.json", out: "index.html", assets: "", url: "https://getorbit.sh/"},
	{code: "es", catalog: "es.json", out: "es/index.html", assets: "../", url: "https://getorbit.sh/es/"},
}

// page is what the template is given.
//
// The sentences go in as text/template and not html/template, for two
// reasons. Several of them carry a <strong> or a <code> that is part of the
// sentence rather than around it, and escaping those would print the tags at
// the reader. And html/template elides comments as it renders, which would
// take with it every argument written into the page's own stylesheet and the
// banner at the top saying the file is generated.
//
// What makes that safe to say rather than hope is where the sentences come
// from: two files in this repository, reviewed like the rest of it. Nothing
// a reader types reaches here.
type page struct {
	Lang      string
	Canonical string
	Assets    string
	T         map[string]string
}

func main() {
	root := flag.String("root", ".", "the module root, holding web/ and site/")

	flag.Parse()

	written, err := build(*root)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	for _, name := range slices.Sorted(maps.Keys(written)) {
		fmt.Printf("wrote %s\n", filepath.Join(*root, "site", name))
	}
}

// build renders every language and writes each page under root/site,
// answering what it wrote so a test can render without touching the disk.
func build(root string) (map[string][]byte, error) {
	pages, err := render(root)
	if err != nil {
		return nil, err
	}

	for name, body := range pages {
		path := filepath.Join(root, "site", name)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return nil, fmt.Errorf("make room for %q: %w", path, err)
		}

		if err := os.WriteFile(path, body, 0o644); err != nil { //nolint:gosec // a page the world reads
			return nil, fmt.Errorf("write %q: %w", path, err)
		}
	}

	return pages, nil
}

// render is every page by the name it is written under.
func render(root string) (map[string][]byte, error) {
	web := filepath.Join(root, "web")

	tmpl, err := template.New("page.tmpl.html").Option("missingkey=error").
		ParseFiles(filepath.Join(web, "page.tmpl.html"))
	if err != nil {
		return nil, fmt.Errorf("read the template: %w", err)
	}

	catalogs := map[string]map[string]string{}

	for _, l := range languages {
		words, err := catalog(filepath.Join(web, l.catalog))
		if err != nil {
			return nil, err
		}

		catalogs[l.code] = words
	}

	if err := sameKeys(catalogs); err != nil {
		return nil, err
	}

	pages := map[string][]byte{}

	for _, l := range languages {
		body, err := one(tmpl, l, catalogs[l.code])
		if err != nil {
			return nil, err
		}

		pages[l.out] = body
	}

	return pages, nil
}

// one renders a single language.
func one(tmpl *template.Template, l language, words map[string]string) ([]byte, error) {
	var b bytes.Buffer

	err := tmpl.Execute(&b, page{
		Lang:      l.code,
		Canonical: l.url,
		Assets:    l.assets,
		T:         words,
	})
	if err != nil {
		return nil, fmt.Errorf("render %s: %w", l.code, err)
	}

	return b.Bytes(), nil
}

// catalog reads one language's sentences.
func catalog(path string) (map[string]string, error) {
	body, err := os.ReadFile(path) //nolint:gosec // a path this program names itself
	if err != nil {
		return nil, fmt.Errorf("read %q: %w", path, err)
	}

	var words map[string]string
	if err := json.Unmarshal(body, &words); err != nil {
		return nil, fmt.Errorf("read %q: %w", path, err)
	}

	return words, nil
}

// sameKeys refuses a translation that is missing a sentence, or that carries
// one nothing asks for.
//
// Without it a key added in English and not in Spanish renders as the empty
// string, and an empty bullet in the middle of a list is the kind of thing
// nobody sees until somebody who reads Spanish opens the page.
func sameKeys(catalogs map[string]map[string]string) error {
	first := languages[0]

	want := slices.Sorted(maps.Keys(catalogs[first.code]))

	for _, l := range languages[1:] {
		got := slices.Sorted(maps.Keys(catalogs[l.code]))
		if slices.Equal(want, got) {
			continue
		}

		return fmt.Errorf("%s and %s do not say the same things: %s is missing %v, and has %v that %s does not",
			first.catalog, l.catalog, l.catalog,
			missing(want, got), missing(got, want), first.catalog)
	}

	return nil
}

// missing is everything in want that got does not have.
func missing(want, got []string) []string {
	var absent []string

	for _, k := range want {
		if !slices.Contains(got, k) {
			absent = append(absent, k)
		}
	}

	return absent
}
