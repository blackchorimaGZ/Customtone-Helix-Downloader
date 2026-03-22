package main

import (
	"bytes"
	_ "embed"
	"context"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/PuerkitoBio/goquery"
	"database/sql"
	_ "modernc.org/sqlite"

	"descargadorhelix/i18n"
)

//go:embed Icon.png
var logoBytes []byte

//go:embed logo.jpg
var logoJpgBytes []byte

//go:embed pch.jpg
var pchJpgBytes []byte

//go:embed manual.html
var manualHtmlBytes []byte

// ── Constants ──────────────────────────────────────────────────────────
const (
	AppTitle       = "CustomTone Helix Downloader"
	AppVersion     = ""
	BrowseURL      = "https://line6.com/customtone/browse/helix"
	DeliverURL     = "https://line6.com/customtone/tone/deliver"
	SearchURL      = "https://line6.com/customtone/search/"
	UserAgent      = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36"
	ConfigFile     = "descargador_config.json"
	DBFilename     = "presets.db"
	PresetsSubdir  = "PRESETS"
	TonesPerPage   = 10
	Workers        = 8
	DefaultPageSz  = 50
)

// ── Neon Theme (matching logo colors) ─────────────────────────────────
var (
	colBG        = color.NRGBA{R: 0x0D, G: 0x0B, B: 0x1E, A: 0xFF}
	colSurface   = color.NRGBA{R: 0x1A, G: 0x16, B: 0x33, A: 0xFF}
	colAccent    = color.NRGBA{R: 0x00, G: 0xBF, B: 0xFF, A: 0xFF} // cyan
	colMagenta   = color.NRGBA{R: 0xE0, G: 0x30, B: 0xA0, A: 0xFF}
	colRed       = color.NRGBA{R: 0xFF, G: 0x30, B: 0x30, A: 0xFF}
	colGreen     = color.NRGBA{R: 0x30, G: 0xE0, B: 0x80, A: 0xFF}
	colYellow    = color.NRGBA{R: 0xFF, G: 0xD0, B: 0x40, A: 0xFF}
	colFG        = color.NRGBA{R: 0xFF, G: 0xFF, B: 0xFF, A: 0xFF} // Pure White
	colDimFG     = color.NRGBA{R: 0xC0, G: 0xC0, B: 0xE0, A: 0xFF} // Brighter Dim
)

type neonTheme struct{}

func (n *neonTheme) Color(name fyne.ThemeColorName, v fyne.ThemeVariant) color.Color {
	switch name {
	case theme.ColorNameBackground:
		return colBG
	case theme.ColorNameForeground:
		return colFG
	case theme.ColorNameButton:
		return colSurface
	case theme.ColorNamePrimary:
		return colAccent
	case theme.ColorNameFocus:
		return colAccent
	case theme.ColorNameSelection:
		return color.NRGBA{R: 0, G: 0xBF, B: 0xFF, A: 0x40}
	case theme.ColorNameInputBackground:
		return colSurface
	case theme.ColorNameInputBorder:
		return colDimFG
	case theme.ColorNamePlaceHolder:
		return colDimFG
	case theme.ColorNameDisabled:
		return colDimFG
	case theme.ColorNameScrollBar:
		return colDimFG
	case theme.ColorNameHeaderBackground:
		return color.NRGBA{R: 0x15, G: 0x12, B: 0x2B, A: 0xFF}
	case theme.ColorNameShadow:
		return color.Transparent
	case theme.ColorNameSuccess:
		return colGreen
	case theme.ColorNameError:
		return colRed
	case theme.ColorNameWarning:
		return colYellow
	case theme.ColorNameOverlayBackground:
		return color.Black
	default:
		return theme.DefaultTheme().Color(name, theme.VariantDark)
	}
}

func (n *neonTheme) Font(s fyne.TextStyle) fyne.Resource {
	return theme.DefaultTheme().Font(s)
}

func (n *neonTheme) Icon(name fyne.ThemeIconName) fyne.Resource {
	return theme.DefaultTheme().Icon(name)
}

func (n *neonTheme) Size(name fyne.ThemeSizeName) float32 {
	if name == theme.SizeNamePadding {
		return 6
	}
	if name == theme.SizeNameInputRadius || name == theme.SizeNameSelectionRadius {
		return 8
	}
	return theme.DefaultTheme().Size(name)
}

// ── Data structures ───────────────────────────────────────────────────
type Tone struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Style     string `json:"style"`
	Band      string `json:"band"`
	Author    string `json:"author"`
	Date      string `json:"date"`
	Downloads string `json:"downloads"`
}

type DownloadResult struct {
	ID    string
	Name  string
	Error error
}

type Config struct {
	DownloadFolder string `json:"download_folder"`
	Sort           string `json:"sort"`
	LastSearchDate string `json:"last_search_date"`
	Language       string `json:"language"`
}

type LocalDBEntry struct {
	Name     string `json:"name"`
	Filename string `json:"filename"`
	Date     string `json:"date"`
}

// ── Sort options ──────────────────────────────────────────────────────
func getSortOptions() []string {
	return []string{
		i18n.Get("sort_rating"),
		i18n.Get("sort_recent"),
		i18n.Get("sort_downloads"),
		i18n.Get("sort_name"),
	}
}

func getSortKey(option string) string {
	switch option {
	case i18n.Get("sort_rating"): return "rating"
	case i18n.Get("sort_recent"): return "posted"
	case i18n.Get("sort_downloads"): return "thecount"
	case i18n.Get("sort_name"): return "name"
	}
	return "rating"
}

// ── Helpers ───────────────────────────────────────────────────────────
func safeFilename(name string) string {
	re := regexp.MustCompile(`[\\/*?:"<>|]`)
	return strings.TrimSpace(re.ReplaceAllString(name, ""))
}

func appDir() string {
	exe, _ := os.Executable()
	return filepath.Dir(exe)
}

func configPath() string {
	return filepath.Join(appDir(), ConfigFile)
}

func loadConfig() Config {
	def := Config{
		DownloadFolder: "",
		Sort:           "rating",
		Language:       "es",
	}
	data, err := os.ReadFile(configPath())
	if err != nil {
		i18n.SetLang(def.Language)
		return def
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		i18n.SetLang(def.Language)
		return def
	}
	// Note: We don't force a default if it's empty, 
	// because doDownload and other parts handle empty paths by prompting.
	if cfg.Sort == "" {
		cfg.Sort = "rating"
	}
	if cfg.Language == "" {
		cfg.Language = "es"
	}
	i18n.SetLang(cfg.Language)
	return cfg
}

func saveConfig(cfg Config) {
	data, _ := json.MarshalIndent(cfg, "", "  ")
	os.WriteFile(configPath(), data, 0644)
}

func loadLocalDB(folder string) map[string]LocalDBEntry {
	path := filepath.Join(folder, DBFilename)
	data, err := os.ReadFile(path)
	if err != nil {
		return make(map[string]LocalDBEntry)
	}
	var db map[string]LocalDBEntry
	if err := json.Unmarshal(data, &db); err != nil {
		return make(map[string]LocalDBEntry)
	}
	return db
}

func saveLocalDB(folder string, db map[string]LocalDBEntry) {
	// Legacy function kept for possible migration or replaced by SQLite logic
}

func applyGradient(obj fyne.CanvasObject) fyne.CanvasObject {
	grad := canvas.NewLinearGradient(colBG, colSurface, 315)
	return container.NewMax(grad, container.NewPadded(obj))
}

func showSplash(a fyne.App, isExit bool, onDone func()) {
	loadingText := i18n.Get("starting")
	if isExit {
		loadingText = i18n.Get("closing")
	}
	
	w := a.NewWindow("")
	w.SetFixedSize(true)
	w.SetPadded(false)
	
	// Create logo from embed
	img := canvas.NewImageFromResource(fyne.NewStaticResource("Icon.png", logoBytes))
	img.FillMode = canvas.ImageFillContain
	img.SetMinSize(fyne.NewSize(400, 200))

	branding := widget.NewLabel("Blackchorima 2026")
	branding.TextStyle = fyne.TextStyle{Bold: true, Italic: true}
	branding.Alignment = fyne.TextAlignCenter
	
	status := widget.NewLabel(loadingText)
	status.Alignment = fyne.TextAlignCenter

	content := container.NewVBox(
		layout.NewSpacer(),
		img,
		branding,
		status,
		layout.NewSpacer(),
	)
	
	// Black background for splash
	bg := canvas.NewRectangle(color.Black)
	w.SetContent(container.NewMax(bg, container.NewPadded(content)))
	w.Resize(fyne.NewSize(500, 350))
	w.CenterOnScreen()
	w.Show()

	time.AfterFunc(2*time.Second, func() {
		fyne.Do(func() {
			w.Close()
			if onDone != nil {
				onDone()
			}
		})
	})
}

func initDB(folder string) (*sql.DB, error) {
	dbPath := filepath.Join(folder, DBFilename)
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, err
	}

	query := `
	CREATE TABLE IF NOT EXISTS presets (
		id TEXT PRIMARY KEY,
		name TEXT,
		style TEXT,
		band TEXT,
		author TEXT,
		date TEXT,
		downloads TEXT,
		filename TEXT,
		local_path TEXT,
		download_date TEXT
	);`
	_, err = db.Exec(query)
	return db, err
}

func savePresetToDB(db *sql.DB, t Tone, filename, localPath string) error {
	query := `
	INSERT INTO presets (id, name, style, band, author, date, downloads, filename, local_path, download_date)
	VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	ON CONFLICT(id) DO UPDATE SET
		name=excluded.name,
		style=excluded.style,
		band=excluded.band,
		author=excluded.author,
		date=excluded.date,
		downloads=excluded.downloads,
		filename=excluded.filename,
		local_path=excluded.local_path,
		download_date=excluded.download_date;`
	_, err := db.Exec(query, t.ID, t.Name, t.Style, t.Band, t.Author, t.Date, t.Downloads, filename, localPath, time.Now().Format("2006-01-02 15:04:05"))
	return err
}

func newClient() *http.Client {
	return &http.Client{Timeout: 20 * time.Second}
}

func doGet(ctx context.Context, client *http.Client, rawURL string) (*http.Response, error) {
	var lastErr error
	for i := 0; i < 4; i++ {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		req, _ := http.NewRequestWithContext(ctx, "GET", rawURL, nil)
		req.Header.Set("User-Agent", UserAgent)
		req.Header.Set("Accept", "text/html,application/xhtml+xml,*/*;q=0.8")
		resp, err := client.Do(req)
		if err == nil {
			return resp, nil
		}
		lastErr = err
		
		// Wait with context awareness
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(time.Duration(1<<i) * time.Second):
		}
	}
	return nil, lastErr
}

func buildPageURL(pageNum int, term, sortKey string) string {
	var base string
	params := url.Values{}
	if term != "" {
		base = fmt.Sprintf("%shelix/%d/", SearchURL, pageNum)
		params.Set("submitted", "1")
		params.Set("family", "helix")
		params.Set("search_term", term)
	} else {
		base = fmt.Sprintf("%s/%d/", BrowseURL, pageNum)
	}
	if sortKey != "" {
		params.Set("sort", sortKey)
	}
	if len(params) > 0 {
		return base + "?" + params.Encode()
	}
	return base
}

var toneIDRe = regexp.MustCompile(`tone-(\d+)`)
var toneNameRe = regexp.MustCompile(`(?i)^TONE\s*NAME:\s*`)
var dlCountRe = regexp.MustCompile(`(?i)([\d,]+)\s+downloads?`)
var dateCleanRe = regexp.MustCompile(`^\s*\S+\s*`)

func parseTones(doc *goquery.Document) []Tone {
	var tones []Tone
	doc.Find("div.tone").Each(func(i int, s *goquery.Selection) {
		dlBtn := s.Find("a.login-modal").First()
		if dlBtn.Length() == 0 {
			return
		}
		btnID, _ := dlBtn.Attr("id")
		m := toneIDRe.FindStringSubmatch(btnID)
		if len(m) < 2 {
			return
		}
		tid := m[1]

		name := "Preset_" + tid
		s.Find("a").EachWithBreak(func(j int, a *goquery.Selection) bool {
			txt := strings.TrimSpace(a.Text())
			if strings.Contains(strings.ToUpper(txt), "TONE NAME:") {
				name = strings.TrimSpace(toneNameRe.ReplaceAllString(txt, ""))
				return false
			}
			return true
		})

		author := "—"
		profLink := s.Find("a.inline-cta").First()
		if profLink.Length() > 0 {
			a := strings.TrimSpace(profLink.Text())
			if a != "" {
				author = a
			}
		}

		style := "—"
		band := "—"
		// The first ul.col-xs-10 usually holds the Tone Name. 
		// The second ul.col-xs-10 or ul.col-xs-4 might hold Band or Style
		detailsStrs := []string{}
		s.Find("div.details ul li").Each(func(j int, li *goquery.Selection) {
			txt := strings.TrimSpace(li.Text())
			if txt != "" && !strings.Contains(strings.ToUpper(txt), "TONE NAME:") {
				detailsStrs = append(detailsStrs, txt)
			}
		})

		// Customtone typically shows Style, then Band/Artist in these extra LIs if provided
		if len(detailsStrs) > 0 {
			style = detailsStrs[0]
			if len(detailsStrs) > 1 {
				band = detailsStrs[1]
			}
		}

		// If Band isn't explicitly found, we might combine Style/Band later, 
		// but let's try to parse the Band exactly from 'STYLE: ... BAND: ...' if customtone formatted it that way.
		if strings.Contains(strings.ToUpper(style), "STYLE") {
			parts := strings.SplitAfter(style, "STYLE:")
			if len(parts) > 1 {
				style = strings.TrimSpace(parts[1])
			}
		}

		dateText := ""
		dateDiv := s.Find("div.date").First()
		if dateDiv.Length() > 0 {
			// Extract all text, which might include "1/21/16" after the <span> icon
			dateText = strings.TrimSpace(dateDiv.Text())
			// Clean up newlines or extra spaces that GoQuery might capture
			dateText = strings.ReplaceAll(dateText, "\n", "")
			dateText = strings.ReplaceAll(dateText, "\t", "")
			// The original regex dateCleanRe (`^\s*\S+\s*`) was stripping the date itself sometimes.
			// Let's just remove everything up to the first digit.
			idx := strings.IndexAny(dateText, "0123456789")
			if idx != -1 {
				dateText = dateText[idx:]
			}
		}

		dlCount := ""
		s.Find("*").Each(func(j int, el *goquery.Selection) {
			if dlCount != "" {
				return
			}
			txt := el.Text()
			dm := dlCountRe.FindStringSubmatch(txt)
			if len(dm) > 1 {
				dlCount = dm[1]
			}
		})

		tones = append(tones, Tone{
			ID: tid, Name: name, Style: style,
			Author: author, Band: band, Date: dateText, Downloads: dlCount,
		})
	})
	return tones
}

func fetchPage(ctx context.Context, client *http.Client, pageNum int, term, sortKey string) []Tone {
	u := buildPageURL(pageNum, term, sortKey)
	resp, err := doGet(ctx, client, u)
	if err != nil || resp.StatusCode == 404 {
		return nil
	}
	defer resp.Body.Close()
	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil
	}
	return parseTones(doc)
}

func parseDate(s string) (time.Time, bool) {
	s = strings.TrimSpace(s)
	// Some dates look like: "1/21/16", "02/01/2006", "Jan 2, 2006", "2006-01-02"
	for _, fmt := range []string{
		"1/2/06",
		"01/02/06",
		"1/02/06",
		"01/2/06",
		"1/2/2006",
		"01/02/2006",
		"2006-01-02",
		"02/01/2006",
		"Jan 2, 2006",
	} {
		t, err := time.Parse(fmt, s)
		if err == nil {
			return t, true
		}
	}
	return time.Time{}, false
}

func formatDate(s string) string {
	t, ok := parseDate(s)
	if !ok {
		return s
	}
	return t.Format("02/01/2006")
}

// ── Custom Date Picker ────────────────────────────────────────────────
func showDatePicker(title string, w fyne.Window, target *widget.Entry) {
	now := time.Now()
	if t, ok := parseDate(target.Text); ok {
		now = t
	}

	months := i18n.GetMonths()
	
	lblMonth := widget.NewLabel(fmt.Sprintf("%s %d", months[now.Month()-1], now.Year()))
	lblMonth.Alignment = fyne.TextAlignCenter

	var calGrid *fyne.Container
	var refreshGrid func()

	btnPrev := widget.NewButton("<", func() {
		now = now.AddDate(0, -1, 0)
		refreshGrid()
	})
	btnNext := widget.NewButton(">", func() {
		now = now.AddDate(0, 1, 0)
		refreshGrid()
	})

	calGrid = container.NewGridWithColumns(7)
	
	refreshGrid = func() {
		calGrid.RemoveAll()
		lblMonth.SetText(fmt.Sprintf("%s %d", months[now.Month()-1], now.Year()))
		
		for _, d := range i18n.GetDays() {
			l := widget.NewLabel(d)
			l.Alignment = fyne.TextAlignCenter
			l.TextStyle = fyne.TextStyle{Bold: true}
			calGrid.Add(l)
		}

		firstOfMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
		wd := int(firstOfMonth.Weekday())
		if wd == 0 { wd = 7 } // Sunday = 7
		
		for i := 1; i < wd; i++ {
			calGrid.Add(layout.NewSpacer())
		}
		
		daysInMonth := time.Date(now.Year(), now.Month()+1, 0, 0, 0, 0, 0, now.Location()).Day()
		var dlg dialog.Dialog
		
		for d := 1; d <= daysInMonth; d++ {
			dayNum := d
			btn := widget.NewButton(fmt.Sprintf("%d", dayNum), func() {
				selDate := time.Date(now.Year(), now.Month(), dayNum, 0, 0, 0, 0, now.Location())
				target.SetText(selDate.Format("02/01/2006"))
				if dlg != nil { dlg.Hide() }
			})
			calGrid.Add(btn)
		}
	}
	
	refreshGrid()
	
	content := container.NewVBox(
		container.NewBorder(nil, nil, btnPrev, btnNext, lblMonth),
		calGrid,
	)

	dlg := dialog.NewCustom(title, i18n.Get("cancel"), content, w)
	// Hacky way to close dlg inside the button callback:
	// We re-assign the buttons to hide this exact dlg reference
	// but closures in Go capture the reference. We defined dlg before its use.
	// We'll just define an external wrapper.
	
	wrapperDlg := &dlg
	
	// redefined refresh to use the wrapper
	refreshGrid = func() {
		calGrid.RemoveAll()
		lblMonth.SetText(fmt.Sprintf("%s %d", months[now.Month()-1], now.Year()))
		
		for _, d := range i18n.GetDays() {
			l := widget.NewLabel(d)
			l.Alignment = fyne.TextAlignCenter
			l.TextStyle = fyne.TextStyle{Bold: true}
			calGrid.Add(l)
		}

		firstOfMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
		wd := int(firstOfMonth.Weekday())
		if wd == 0 { wd = 7 }
		
		for i := 1; i < wd; i++ {
			calGrid.Add(layout.NewSpacer())
		}
		
		daysInMonth := time.Date(now.Year(), now.Month()+1, 0, 0, 0, 0, 0, now.Location()).Day()
		
		for d := 1; d <= daysInMonth; d++ {
			dayNum := d
			btn := widget.NewButton(fmt.Sprintf("%d", dayNum), func() {
				selDate := time.Date(now.Year(), now.Month(), dayNum, 0, 0, 0, 0, now.Location())
				target.SetText(selDate.Format("02/01/2006"))
				if *wrapperDlg != nil { (*wrapperDlg).Hide() }
			})
			calGrid.Add(btn)
		}
	}
	refreshGrid()
	dlg.Show()
}

// ── Custom Table Wrappers ─────────────────────────────────────────────
// ── Custom Table Wrappers ─────────────────────────────────────────────
type tappableContainer struct {
	widget.BaseWidget
	bg      *canvas.Rectangle
	content *fyne.Container
	onMouse func(me *desktop.MouseEvent)
}

func newTappableContainer(bg *canvas.Rectangle, content *fyne.Container, onMouse func(me *desktop.MouseEvent)) *tappableContainer {
	tc := &tappableContainer{bg: bg, content: content, onMouse: onMouse}
	tc.ExtendBaseWidget(tc)
	return tc
}

func (tc *tappableContainer) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(container.NewMax(tc.bg, tc.content))
}

func (tc *tappableContainer) MouseDown(me *desktop.MouseEvent) {
	if tc.onMouse != nil {
		tc.onMouse(me)
	}
}

func (tc *tappableContainer) TappedSecondary(me *desktop.MouseEvent) {
	// Secondary tap (right click) handled here if needed, 
	// but MouseDown already receives me.Button.
}

func (tc *tappableContainer) MouseUp(me *desktop.MouseEvent) {}
func (tc *tappableContainer) MouseIn(me *desktop.MouseEvent) {}
func (tc *tappableContainer) MouseOut()                     {}

type tableWrapper struct {
	widget.BaseWidget
	table *widget.Table
}

func (tw *tableWrapper) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(tw.table)
}

func (tw *tableWrapper) Resize(size fyne.Size) {
	tw.BaseWidget.Resize(size)
	tw.table.Resize(size)

	avail := float32(size.Width - 16)
	if avail < 800 {
		avail = 800
	}
	
	tw.table.SetColumnWidth(0, avail*0.35)
	tw.table.SetColumnWidth(1, avail*0.13)
	tw.table.SetColumnWidth(2, avail*0.20)
	tw.table.SetColumnWidth(3, avail*0.13)
	tw.table.SetColumnWidth(4, avail*0.10)
	tw.table.SetColumnWidth(5, avail*0.09)
}

func (tw *tableWrapper) MinSize() fyne.Size {
	return tw.table.MinSize()
}

// ── Intercepting Overlay ──────────────────────────────────────────────
type interceptingOverlay struct {
	widget.BaseWidget
	bg *canvas.Rectangle
}

func newInterceptingOverlay() *interceptingOverlay {
	io := &interceptingOverlay{}
	io.bg = canvas.NewRectangle(color.NRGBA{R: 0, G: 0, B: 0, A: 200}) // Semi-transparent black
	io.ExtendBaseWidget(io)
	return io
}

func (io *interceptingOverlay) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(io.bg)
}

func (io *interceptingOverlay) Tapped(*fyne.PointEvent)          {}
func (io *interceptingOverlay) TappedSecondary(*fyne.PointEvent) {}
func (io *interceptingOverlay) MouseDown(*desktop.MouseEvent)    {}
func (io *interceptingOverlay) MouseUp(*desktop.MouseEvent)      {}
func (io *interceptingOverlay) MouseIn(*desktop.MouseEvent)      {}
func (io *interceptingOverlay) MouseOut()                        {}

// ── Pulsing Gradient Button ───────────────────────────────────────────
type pulsingButton struct {
	widget.BaseWidget
	topText    *canvas.Text
	bottomText *canvas.Text
	icon       *canvas.Image
	grad       *canvas.LinearGradient
	anim       *fyne.Animation
	onTap      func()
	hovered    bool
}

func newPulsingButton(topLabel, bottomLabel string, iconRes fyne.Resource, onTap func()) *pulsingButton {
	pb := &pulsingButton{onTap: onTap}
	pb.topText = canvas.NewText(topLabel, color.White)
	pb.topText.TextStyle = fyne.TextStyle{Bold: true}
	pb.topText.Alignment = fyne.TextAlignCenter

	pb.bottomText = canvas.NewText(bottomLabel, color.NRGBA{R: 0xCC, G: 0xCC, B: 0xCC, A: 0xFF})
	pb.bottomText.TextStyle = fyne.TextStyle{Italic: true}
	pb.bottomText.TextSize = 13
	pb.bottomText.Alignment = fyne.TextAlignCenter
	
	pb.icon = canvas.NewImageFromResource(iconRes)
	pb.icon.FillMode = canvas.ImageFillContain
	pb.icon.SetMinSize(fyne.NewSize(24, 24))

	// Fire gradient colors
	color1 := color.NRGBA{R: 0xFF, G: 0x45, B: 0x00, A: 0xFF} // OrangeRed
	color2 := color.NRGBA{R: 0xFF, G: 0x8C, B: 0x00, A: 0xFF} // DarkOrange
	
	pb.grad = canvas.NewLinearGradient(color1, color2, 45)

	pb.ExtendBaseWidget(pb)

	// Pulsing animation by shifting the gradient angle slightly and fading
	pb.anim = canvas.NewColorRGBAAnimation(
		color.NRGBA{R: 0xFF, G: 0x1A, B: 0x00, A: 0xFF}, // Deep Red
		color.NRGBA{R: 0xFF, G: 0xA5, B: 0x00, A: 0xFF}, // Bright Orange
		time.Millisecond*500,
		func(c color.Color) {
			pb.grad.StartColor = c
			// Create a secondary color somewhat darker for the end of the gradient
			r, g, b, a := c.RGBA()
			c2 := color.NRGBA{R: uint8(r >> 9), G: uint8(g >> 10), B: uint8(b >> 9), A: uint8(a >> 8)}
			pb.grad.EndColor = c2
			pb.grad.Refresh()
		},
	)
	pb.anim.AutoReverse = true
	pb.anim.RepeatCount = fyne.AnimationRepeatForever
	
	pb.anim.Start()

	return pb
}

func (pb *pulsingButton) CreateRenderer() fyne.WidgetRenderer {
	textContainer := container.NewVBox(pb.topText, pb.bottomText)
	content := container.NewHBox(layout.NewSpacer(), pb.icon, textContainer, layout.NewSpacer())
	padded := container.NewPadded(content)
	bg := container.NewMax(pb.grad)
	
	// Add a subtle border or solid background logic if needed, but gradient is fine
	return widget.NewSimpleRenderer(container.NewMax(bg, padded))
}

func (pb *pulsingButton) Tapped(*fyne.PointEvent) {
	if pb.onTap != nil {
		pb.onTap()
	}
}

func (pb *pulsingButton) MouseIn(*desktop.MouseEvent) {
	pb.hovered = true
	// Could stop animation or change color on hover, but we'll leave it pulsing
}

func (pb *pulsingButton) MouseOut() {
	pb.hovered = false
}
func (pb *pulsingButton) MouseDown(*desktop.MouseEvent) {}
func (pb *pulsingButton) MouseUp(*desktop.MouseEvent)   {}
func (pb *pulsingButton) TappedSecondary(*fyne.PointEvent) {}

// ── Main App ──────────────────────────────────────────────────────────
func main() {
	cfg := loadConfig()

	descargador := app.NewWithID("com.customtone.helix.downloader")
	descargador.SetIcon(theme.SettingsIcon())
	descargador.Settings().SetTheme(&neonTheme{})

	// Show startup splash
	showSplash(descargador, false, func() {
		title := AppTitle
		if AppVersion != "" {
			title += " v" + AppVersion
		}
		w := descargador.NewWindow(title)
		w.Resize(fyne.NewSize(1150, 820))

		// Set exit intercept
		w.SetCloseIntercept(func() {
			w.Hide()
			showSplash(descargador, true, func() {
				descargador.Quit()
			})
		})
	// Set app icon from embedded logo
	logoImg, _, _ := image.Decode(bytes.NewReader(logoBytes))
	_ = logoImg
	logoResource := fyne.NewStaticResource("logo.png", logoBytes)
	w.SetIcon(logoResource)
	
	// Ensure PRESETS folder and DB exist on startup only if folder is set
	if cfg.DownloadFolder != "" {
		presetsFolder := filepath.Join(cfg.DownloadFolder, PresetsSubdir)
		os.MkdirAll(presetsFolder, 0755)
		
		if db, err := initDB(cfg.DownloadFolder); err == nil {
			db.Close()
		} else {
			log.Println(i18n.Get("error_init_db"), err)
		}
	}

	var doDownload func(ids []string, skipExisting bool)
	_ = doDownload

	localDB := loadLocalDB(cfg.DownloadFolder)
	
	// ── Shared State ──────────────────────────────────────────────────
	var mu sync.Mutex
	var lastSelectedIdx int = -1
	var table *widget.Table
	var allResults []Tone
	visibleResults := &allResults
	selectedIDs := make(map[string]bool)
	var cancelFlag atomic.Bool
	var isShiftDown, isCtrlDown atomic.Bool
	viewPage := 1
	viewPageSize := DefaultPageSz

	deployManager := func(folder string) {
		managerName := "CTHD_Manager.exe"
		targetPath := filepath.Join(folder, managerName)
		sourcePath := filepath.Join(appDir(), managerName)
		
		if _, err := os.Stat(sourcePath); err == nil {
			// Copy if source exists
			data, err := os.ReadFile(sourcePath)
			if err == nil {
				os.WriteFile(targetPath, data, 0755)
			}
		}
	}

	// Fyne Canvas Desktop shortcuts for Shift and Ctrl detection
	if deskCanvas, ok := w.Canvas().(desktop.Canvas); ok {
		deskCanvas.SetOnKeyDown(func(ke *fyne.KeyEvent) {
			if ke.Name == desktop.KeyShiftLeft || ke.Name == desktop.KeyShiftRight {
				isShiftDown.Store(true)
			}
			if ke.Name == desktop.KeyControlLeft || ke.Name == desktop.KeyControlRight {
				isCtrlDown.Store(true)
			}
		})
		deskCanvas.SetOnKeyUp(func(ke *fyne.KeyEvent) {
			if ke.Name == desktop.KeyShiftLeft || ke.Name == desktop.KeyShiftRight {
				isShiftDown.Store(false)
			}
			if ke.Name == desktop.KeyControlLeft || ke.Name == desktop.KeyControlRight {
				isCtrlDown.Store(false)
			}
		})
	}

	// ── Pagination Helpers (needed by widgets) ────────────────────────
	totalViewPages := func() int {
		mu.Lock()
		defer mu.Unlock()
		n := len(*visibleResults)
		if n == 0 {
			return 0
		}
		return (n + viewPageSize - 1) / viewPageSize
	}

	pageLabel := widget.NewLabel("0 / 0")
	pageLabel.TextStyle = fyne.TextStyle{Bold: true}
	pageLabel.Alignment = fyne.TextAlignCenter

	localFilterEntry := widget.NewEntry()
	localFilterEntry.SetPlaceHolder(i18n.Get("filter_results"))
	
	btnLocalFilter := widget.NewButtonWithIcon(i18n.Get("filter_btn"), theme.SearchIcon(), func() {
		localFilterEntry.OnChanged(localFilterEntry.Text)
	})

	checkAndSearch := widget.NewCheck(i18n.Get("and_search"), nil)
	checkAndSearch.OnChanged = func(b bool) {
		localFilterEntry.OnChanged(localFilterEntry.Text)
	}
	
	actionGroup := container.NewHBox(
		container.NewGridWrap(fyne.NewSize(300, 35), localFilterEntry),
		btnLocalFilter,
		checkAndSearch,
	)
	actionGroup.Hide() // Hide initially until search results are loaded

	refreshPage := func() {
		tp := totalViewPages()
		if viewPage > tp && tp > 0 {
			viewPage = tp
		}
		if viewPage < 1 {
			viewPage = 1
		}
		pageLabel.SetText(fmt.Sprintf("%d / %d", viewPage, tp))
		if table != nil {
			table.Refresh()
		}
	}

	// ── Selection and Sort logic ──────────────────────────────────────
	sortReverse := false
	sortByCol := func(col int) {
		mu.Lock()
		sort.SliceStable(allResults, func(i, j int) bool {
			var a, b string
			switch col {
			case 0:
				a, b = allResults[i].Name, allResults[j].Name
			case 1:
				a, b = allResults[i].Style, allResults[j].Style
			case 2:
				a, b = allResults[i].Band, allResults[j].Band
			case 3:
				a, b = allResults[i].Author, allResults[j].Author
			case 4:
				a, b = allResults[i].Date, allResults[j].Date
			case 5:
				ai, _ := strconv.Atoi(strings.ReplaceAll(allResults[i].Downloads, ",", ""))
				bi, _ := strconv.Atoi(strings.ReplaceAll(allResults[j].Downloads, ",", ""))
				if sortReverse {
					return ai > bi
				}
				return ai < bi
			}
			if sortReverse {
				return strings.ToLower(a) > strings.ToLower(b)
			}
			return strings.ToLower(a) < strings.ToLower(b)
		})
		sortReverse = !sortReverse
		viewPage = 1
		mu.Unlock()
		
		localFilterEntry.OnChanged(localFilterEntry.Text)
	}

	selectionLabel := widget.NewLabel(i18n.GetF("selected_out_of", 0, 0))
	selectionLabel.TextStyle = fyne.TextStyle{Bold: true}

	// ── Widgets ───────────────────────────────────────────────────────
	searchEntry := widget.NewEntry()
	searchEntry.SetPlaceHolder(i18n.Get("term_placeholder"))
	
	// Define btnSearch later, but we need its logic for OnSubmitted
	// We'll wire up OnSubmitted after btnSearch is defined.

	sortSelect := widget.NewSelect(getSortOptions(), nil)
	sortSelect.PlaceHolder = i18n.Get("select_placeholder")
	folderLabel := widget.NewLabel(cfg.DownloadFolder)
	folderLabel.Wrapping = fyne.TextTruncate

	searchModeGlobal := widget.NewRadioGroup([]string{i18n.Get("global_search"), i18n.Get("since_last_search")}, nil)
	searchModeGlobal.SetSelected(i18n.Get("global_search"))
	searchModeGlobal.Horizontal = true
 
	// Wire up localFilterEntry
	localFilterEntry.OnChanged = func(s string) {
		mu.Lock()
		
		var terms []string
		for _, part := range strings.Split(s, ",") {
			t := strings.ToLower(strings.TrimSpace(part))
			if t != "" {
				terms = append(terms, t)
			}
		}

		if len(terms) == 0 {
			*visibleResults = allResults
		} else {
			var filtered []Tone
			isAND := checkAndSearch.Checked
			for _, t := range allResults {
				haystack := strings.ToLower(t.Name + " " + t.Author + " " + t.Style + " " + t.Band)
				matches := 0
				for _, term := range terms {
					if strings.Contains(haystack, term) {
						matches++
					}
				}
				if isAND {
					if matches == len(terms) {
						filtered = append(filtered, t)
					}
				} else {
					if matches > 0 {
						filtered = append(filtered, t)
					}
				}
			}
			*visibleResults = filtered
		}
		viewPage = 1
		mu.Unlock()

		refreshPage()
		mu.Lock()
		selectionLabel.SetText(i18n.GetF("selected_out_of", len(selectedIDs), len(*visibleResults)))
		mu.Unlock()
	}
	localFilterEntry.OnSubmitted = func(s string) {
		localFilterEntry.OnChanged(s)
	}

	lastSearchLabel := widget.NewLabel("")
	dateFromEntry := widget.NewEntry()
	dateToEntry := widget.NewEntry()
	btnDateFrom := widget.NewButton("📅", func() { showDatePicker(i18n.Get("dp_title_from"), w, dateFromEntry) })
	btnDateTo := widget.NewButton("📅", func() { showDatePicker(i18n.Get("dp_title_to"), w, dateToEntry) })

	statusLabel := widget.NewLabel(i18n.Get("status_ready"))
	statusLabel.TextStyle = fyne.TextStyle{Bold: true}
	progressBar := widget.NewProgressBar()
	resultCountLabel := widget.NewLabel(i18n.Get("no_results_title"))

	// Results table
	table = widget.NewTable(
		func() (int, int) {
			mu.Lock()
			defer mu.Unlock()
			start := (viewPage - 1) * viewPageSize
			end := start + viewPageSize
			if end > len(*visibleResults) {
				end = len(*visibleResults)
			}
			count := end - start
			if count < 0 {
				count = 0
			}
			return count + 1, 6 // +1 for header row
		},
		func() fyne.CanvasObject {
			bg := canvas.NewRectangle(color.Transparent)
			icon := widget.NewIcon(theme.CheckButtonIcon())
			lbl := widget.NewLabel("Template")
			lbl.Wrapping = fyne.TextTruncate
			
			headerTxt := canvas.NewText("", color.Black)
			headerTxt.TextStyle = fyne.TextStyle{Bold: true}
			headerTxt.Alignment = fyne.TextAlignCenter
			headerTxt.Hide()

			content := container.NewMax(
				container.NewBorder(nil, nil, icon, nil, lbl),
				headerTxt,
			)
			
			return newTappableContainer(bg, content, nil)
		},
		func(id widget.TableCellID, o fyne.CanvasObject) {
			tc := o.(*tappableContainer)
			bg := tc.bg
			content := tc.content
			
			innerBorder := content.Objects[0].(*fyne.Container)
			var icon *widget.Icon
			var l *widget.Label
			for _, obj := range innerBorder.Objects {
				if ic, ok := obj.(*widget.Icon); ok {
					icon = ic
				} else if lb, ok := obj.(*widget.Label); ok {
					l = lb
				}
			}
			h := content.Objects[1].(*canvas.Text)
			
			mu.Lock()
			start := (viewPage - 1) * viewPageSize
			absIdx := start + id.Row - 1
			
			var isSelected bool
			if id.Row > 0 && absIdx < len(*visibleResults) {
				isSelected = selectedIDs[(*visibleResults)[absIdx].ID]
			}

			// Background colors
			if id.Row == 0 {
				bg.FillColor = color.NRGBA{R: 0xFF, G: 0xD0, B: 0x40, A: 0xFF} // Bright Yellow
				bg.Refresh()
				
				innerBorder.Hide()
				h.Show()
				h.TextSize = theme.TextSize()
				
				switch id.Col {
				case 0: h.Text = i18n.Get("col_name")
				case 1: h.Text = i18n.Get("col_style")
				case 2: h.Text = i18n.Get("col_band")
				case 3: h.Text = i18n.Get("col_author")
				case 4: h.Text = i18n.Get("col_date")
				case 5: h.Text = i18n.Get("col_downloads")
				}
				h.Refresh()

				// Sort on tap
				tc.onMouse = func(me *desktop.MouseEvent) {
					if me.Button == desktop.MouseButtonPrimary {
						sortByCol(id.Col)
					}
				}
				mu.Unlock()
				return
			}
			
			h.Hide()
			innerBorder.Show()
			l.TextStyle = fyne.TextStyle{}

			if isSelected {
				bg.FillColor = color.NRGBA{R: 0x00, G: 0xBF, B: 0xFF, A: 0x40}
			} else if id.Row%2 == 0 {
				bg.FillColor = color.NRGBA{R: 0x2A, G: 0x26, B: 0x43, A: 0xFF}
			} else {
				bg.FillColor = color.Transparent
			}
			bg.Refresh()

			if absIdx >= len(*visibleResults) {
				l.SetText("")
				icon.Hide()
				tc.onMouse = nil
				mu.Unlock()
				return
			}
			
			t := (*visibleResults)[absIdx]
			switch id.Col {
			case 0:
				icon.Show()
				if selectedIDs[t.ID] {
					icon.SetResource(theme.CheckButtonCheckedIcon())
				} else {
					icon.SetResource(theme.CheckButtonIcon())
				}
				l.SetText(t.Name)
			case 1:
				icon.Hide()
				l.SetText(t.Style)
			case 2:
				icon.Hide()
				l.SetText(t.Band)
			case 3:
				icon.Hide()
				l.SetText(t.Author)
			case 4:
				icon.Hide()
				l.SetText(formatDate(t.Date))
			case 5:
				icon.Hide()
				l.SetText(t.Downloads)
			}

			// Selection logic on mouse down
			tc.onMouse = func(me *desktop.MouseEvent) {
				if me.Button == desktop.MouseButtonSecondary {
					// Context Menu
					pop := widget.NewPopUpMenu(fyne.NewMenu("",
						fyne.NewMenuItem(i18n.Get("download_menu"), func() {
							mu.Lock()
							if !selectedIDs[t.ID] {
								// If not selected, clear others and select this one
								selectedIDs = map[string]bool{t.ID: true}
							}
							ids := make([]string, 0, len(selectedIDs))
							for sid := range selectedIDs {
								ids = append(ids, sid)
							}
							mu.Unlock()
							doDownload(ids, false)
						}),
						fyne.NewMenuItem(i18n.Get("view_ct_menu"), func() {
							u := fmt.Sprintf("https://line6.com/customtone/tone/%s/", t.ID)
							parsedURL, _ := url.Parse(u)
							fyne.CurrentApp().OpenURL(parsedURL)
						}),
					), w.Canvas())
					pop.ShowAtPosition(me.AbsolutePosition)
					return
				}

				mu.Lock()
				// Re-check shift/ctrl state from the event itself!
				shiftPressed := me.Modifier&fyne.KeyModifierShift != 0
				ctrlPressed := me.Modifier&fyne.KeyModifierControl != 0

				if shiftPressed && lastSelectedIdx != -1 {
					minIdx, maxIdx := lastSelectedIdx, absIdx
					if minIdx > maxIdx {
						minIdx, maxIdx = maxIdx, minIdx
					}
					for i := minIdx; i <= maxIdx; i++ {
						if i < len(*visibleResults) {
							selectedIDs[(*visibleResults)[i].ID] = true
						}
					}
				} else if ctrlPressed {
					if selectedIDs[t.ID] {
						delete(selectedIDs, t.ID)
					} else {
						selectedIDs[t.ID] = true
					}
					lastSelectedIdx = absIdx
				} else {
					// Toggle just this one
					if selectedIDs[t.ID] {
						delete(selectedIDs, t.ID)
					} else {
						selectedIDs[t.ID] = true
					}
					lastSelectedIdx = absIdx
				}
				mu.Unlock()
				
				selectionLabel.SetText(i18n.GetF("selected_out_of", len(selectedIDs), len(*visibleResults)))
				table.Refresh()
			}
			mu.Unlock()
		},
	)
	table.StickyRowCount = 1
	table.OnSelected = func(id widget.TableCellID) {
		table.UnselectAll()
	}

	wrappedTable := &tableWrapper{table: table}
	wrappedTable.ExtendBaseWidget(wrappedTable)

	// ── Track selection on table tap (including Shift & Ctrl logic) ───
	// lastSelectedIdx is defined above for scoping reasons
	_ = lastSelectedIdx
	_ = sortByCol

	// ── Pagination ────────────────────────────────────────────────────
	totalViewPages = func() int {
		mu.Lock()
		defer mu.Unlock()
		n := len(*visibleResults)
		if n == 0 {
			return 0
		}
		return (n + viewPageSize - 1) / viewPageSize
	}

	pageLabel.Alignment = fyne.TextAlignCenter

	refreshPage = func() {
		tp := totalViewPages()
		if viewPage > tp && tp > 0 {
			viewPage = tp
		}
		if viewPage < 1 {
			viewPage = 1
		}
		pageLabel.SetText(fmt.Sprintf("%d / %d", viewPage, tp))
		if table != nil {
			table.Refresh()
		}
		selectionLabel.SetText(i18n.GetF("selected_out_of", len(selectedIDs), len(*visibleResults)))
		mu.Lock()
		count := len(*visibleResults)
		overall := len(allResults)
		mu.Unlock()
		resultCountLabel.SetText(i18n.GetF("results_found", count))
		statusLabel.SetText(i18n.GetF("results_overall", count, overall))
	}

	pageSizeSelect := widget.NewSelect([]string{"25", "50", "100", "200"}, func(s string) {
		n, _ := strconv.Atoi(s)
		if n > 0 {
			viewPageSize = n
			viewPage = 1
			refreshPage()
		}
	})
	pageSizeSelect.SetSelected(strconv.Itoa(DefaultPageSz))

	btnFirst := widget.NewButton(i18n.Get("nav_first"), func() { viewPage = 1; refreshPage() })
	btnPrev := widget.NewButton(i18n.Get("nav_prev"), func() {
		if viewPage > 1 {
			viewPage--
		}
		refreshPage()
	})
	btnNext := widget.NewButton(i18n.Get("nav_next"), func() {
		if viewPage < totalViewPages() {
			viewPage++
		}
		refreshPage()
	})
	btnLast := widget.NewButton(i18n.Get("nav_last"), func() {
		tp := totalViewPages()
		if tp > 0 {
			viewPage = tp
		}
		refreshPage()
	})

	// ── Selection buttons ─────────────────────────────────────────────
	btnSelectAll := widget.NewButton(i18n.Get("sel_all"), func() {
		mu.Lock()
		for _, t := range *visibleResults {
			selectedIDs[t.ID] = true
		}
		mu.Unlock()
		selectionLabel.SetText(i18n.GetF("selected_out_of", len(selectedIDs), len(*visibleResults)))
		table.Refresh()
	})

	btnDeselectAll := widget.NewButton(i18n.Get("desel_all"), func() {
		mu.Lock()
		selectedIDs = make(map[string]bool)
		mu.Unlock()
		selectionLabel.SetText(i18n.GetF("selected_out_of", 0, len(*visibleResults)))
		table.Refresh()
	})

	btnInvert := widget.NewButton(i18n.Get("invert_sel"), func() {
		mu.Lock()
		newSel := make(map[string]bool)
		for _, t := range *visibleResults {
			if !selectedIDs[t.ID] {
				newSel[t.ID] = true
			}
		}
		selectedIDs = newSel
		mu.Unlock()
		selectionLabel.SetText(i18n.GetF("selected_out_of", len(selectedIDs), len(*visibleResults)))
		table.Refresh()
	})

	// ── Folder selection ──────────────────────────────────────────────
	btnFolder := widget.NewButton(i18n.Get("folder_btn"), func() {
		dialog.ShowFolderOpen(func(lu fyne.ListableURI, err error) {
			if lu != nil {
				path := lu.Path()
				cfg.DownloadFolder = path
				folderLabel.SetText(path)
				saveConfig(cfg)
				localDB = loadLocalDB(path)
			}
		}, w)
	})

	btnOpenFolder := widget.NewButton(i18n.Get("open_folder_btn"), func() {
		os.MkdirAll(cfg.DownloadFolder, 0755)
		openFolder(cfg.DownloadFolder)
	})

	// ── Search ────────────────────────────────────────────────────────
	var btnSearch *widget.Button
	var doSearch func()
	btnSearch = widget.NewButton(i18n.Get("search_btn"), func() { doSearch() })
	
	searchEntry.OnSubmitted = func(s string) { doSearch() }

	doSearch = func() {
		if cfg.DownloadFolder == "" {
			dialog.ShowInformation(i18n.Get("folder_req_title"), i18n.Get("folder_req_msg"), w)
			dialog.ShowFolderOpen(func(lu fyne.ListableURI, err error) {
				if lu != nil {
					path := lu.Path()
					cfg.DownloadFolder = path
					folderLabel.SetText(path)
					saveConfig(cfg)
					localDB = loadLocalDB(path)
					// Trigger search after folder selection
					doSearch()
				}
			}, w)
			return
		}

		// Save config
		sk := getSortKey(sortSelect.Selected)
		if sk == "" {
			sk = "rating"
		}
		cfg.Sort = sk
		saveConfig(cfg)

		cancelFlag.Store(false)
		btnSearch.Disable()

		// Show progress dialog
		ctx, cancelFunc := context.WithCancel(context.Background())
		
		modalStatus := widget.NewLabel(i18n.Get("start_search"))
		modalStatus.Wrapping = fyne.TextWrapWord
		modalPages := widget.NewLabel("")
		modalProgress := widget.NewProgressBar()
		btnCancel := widget.NewButton(i18n.Get("cancel_btn"), func() {
			cancelFlag.Store(true)
			cancelFunc()
			modalStatus.SetText(i18n.Get("canceling"))
		})

		// Logo in modal
		logoImage := canvas.NewImageFromResource(logoResource)
		logoImage.SetMinSize(fyne.NewSize(280, 100))
		logoImage.FillMode = canvas.ImageFillContain

		modalContent := container.NewVBox(
			logoImage,
			modalStatus,
			modalProgress,
			modalPages,
			container.NewCenter(btnCancel),
		)
		modalDialog := dialog.NewCustomWithoutButtons(i18n.Get("searching_title"), applyGradient(modalContent), w)
		modalDialog.Resize(fyne.NewSize(480, 300))
		modalDialog.Show()

		go func() {
			defer func() {
				fyne.Do(func() { btnSearch.Enable() })
			}()

			term := searchEntry.Text
			client := newClient()

			// Probe page 1
			fyne.Do(func() { modalStatus.SetText(i18n.Get("probing")) })
			probeURL := buildPageURL(1, term, sk)
			resp, err := doGet(ctx, client, probeURL)
			if err != nil {
				fyne.Do(func() { 
					modalDialog.Hide()
					if ctx.Err() == nil {
						dialog.ShowError(fmt.Errorf(i18n.GetF("conn_error", err)), w)
					}
				})
				return
			}
			defer resp.Body.Close()
			doc, err := goquery.NewDocumentFromReader(resp.Body)
			if err != nil {
				fyne.Do(func() {
					modalDialog.Hide()
					dialog.ShowError(fmt.Errorf(i18n.GetF("html_error", err)), w)
				})
				return
			}

			totalTones := 0
			doc.Find("span#tone-count").Each(func(i int, s *goquery.Selection) {
				re := regexp.MustCompile(`[\d,]+`)
				m := re.FindString(s.Text())
				m = strings.ReplaceAll(m, ",", "")
				totalTones, _ = strconv.Atoi(m)
			})

			totalPages := 0
			if totalTones > 0 {
				totalPages = (totalTones + TonesPerPage - 1) / TonesPerPage
			}
			if totalPages == 0 {
				fyne.Do(func() {
					modalDialog.Hide()
					dialog.ShowInformation(i18n.Get("no_results_title"), i18n.Get("no_results_msg"), w)
				})
				return
			}

			log.Printf("Probe: %d tones, %d pages", totalTones, totalPages)

			// Parse page 1
			firstTones := parseTones(doc)
			seen := make(map[string]bool)
			var collected []Tone
			for _, t := range firstTones {
				if !seen[t.ID] {
					seen[t.ID] = true
					collected = append(collected, t)
				}
			}

			pagesDone := int32(1)
			fyne.Do(func() {
				modalStatus.SetText(i18n.GetF("first_page_probe", totalPages, len(collected)))
				modalPages.SetText(i18n.GetF("pages_threads", 1, totalPages, Workers))
				modalProgress.SetValue(float64(1) / float64(totalPages))
			})

			// Concurrent fetch
			if totalPages > 1 && !cancelFlag.Load() {
				var wg sync.WaitGroup
				pageCh := make(chan int, totalPages)
				resultCh := make(chan []Tone, totalPages)

				for i := 0; i < Workers; i++ {
					wg.Add(1)
					go func() {
						defer wg.Done()
						c := newClient()
						for pg := range pageCh {
							if cancelFlag.Load() {
								return
							}
							tones := fetchPage(ctx, c, pg, term, sk)
							if cancelFlag.Load() {
								return
							}
							resultCh <- tones
							done := atomic.AddInt32(&pagesDone, 1)
							if !cancelFlag.Load() {
								pct := float64(done) / float64(totalPages)
								fyne.Do(func() {
									modalProgress.SetValue(pct)
									modalStatus.SetText(i18n.GetF("fetching_pages", done, totalPages))
									modalPages.SetText(i18n.GetF("pages_threads", done, totalPages, Workers))
								})
							}
						}
					}()
				}

				go func() {
					for pg := 2; pg <= totalPages; pg++ {
						if cancelFlag.Load() {
							break
						}
						pageCh <- pg
					}
					close(pageCh)
				}()

				go func() {
					wg.Wait()
					close(resultCh)
				}()

				Loop:
				for {
					select {
					case tones, ok := <-resultCh:
						if !ok {
							break Loop
						}
						// There's no need to lock global `mu` because `collected` and `seen` 
						// are localized to this goroutine! 
						for _, t := range tones {
							if !seen[t.ID] {
								seen[t.ID] = true
								collected = append(collected, t)
							}
						}
					case <-ctx.Done():
						// Break instantly if canceled, returning whatever we collected so far
						break Loop
					}
				}
			}

			// Apply date filters
			filtered := collected
			dateFrom := strings.TrimSpace(dateFromEntry.Text)
			dateTo := strings.TrimSpace(dateToEntry.Text)
			sinceLast := ""
			if searchModeGlobal.Selected == i18n.Get("since_last_search") || searchModeGlobal.Selected == "Desde última búsqueda" || searchModeGlobal.Selected == "Since last search" {
				sinceLast = cfg.LastSearchDate
			}

			if dateFrom != "" || dateTo != "" || sinceLast != "" {
				effectiveFrom := dateFrom
				if sinceLast != "" {
					effectiveFrom = sinceLast
				}
				var filt []Tone
				for _, t := range collected {
					td, ok := parseDate(t.Date)
					if !ok {
						filt = append(filt, t)
						continue
					}
					if effectiveFrom != "" {
						ef, efOK := parseDate(effectiveFrom)
						if efOK && td.Before(ef) {
							continue
						}
					}
					if dateTo != "" {
						et, etOK := parseDate(dateTo)
						if etOK && td.After(et) {
							continue
						}
					}
					filt = append(filt, t)
				}
				filtered = filt
			}

			// Update last search date
			cfg.LastSearchDate = time.Now().Format("02/01/2006")
			saveConfig(cfg)
			fyne.Do(func() { lastSearchLabel.SetText("(" + cfg.LastSearchDate + ")") })

			// Update results
			mu.Lock()
			allResults = filtered
			*visibleResults = allResults
			selectedIDs = make(map[string]bool)
			viewPage = 1
			mu.Unlock()

			fyne.Do(func() {
				modalProgress.SetValue(1.0)
				modalDialog.Hide()
				resultCountLabel.SetText(i18n.GetF("results_found", len(filtered)))
				statusLabel.SetText(i18n.GetF("results_overall", len(filtered), len(collected)))
				
				if len(filtered) > 0 {
					actionGroup.Show()
				} else {
					actionGroup.Hide()
				}
				
				refreshPage()
			})
		}()
	}

	// ── Download functions ────────────────────────────────────────────
	// Define doDownload ahead of time for recursion/scoping
	doDownload = func(ids []string, skipExisting bool) {
		if len(ids) == 0 {
			return
		}
		
		// Folder validation: if empty, prompt user to select one
		if cfg.DownloadFolder == "" {
			dialog.ShowInformation(i18n.Get("folder_req_title"), i18n.Get("folder_req_msg"), w)
			dialog.ShowFolderOpen(func(lu fyne.ListableURI, err error) {
				if lu != nil {
					path := lu.Path()
					cfg.DownloadFolder = path
					folderLabel.SetText(path)
					saveConfig(cfg)
					localDB = loadLocalDB(path)
					// Proceed after selection
					doDownload(ids, skipExisting)
				}
			}, w)
			return
		}

		cancelFlag.Store(false)

		ctx, cancelFunc := context.WithCancel(context.Background())
		
		dlStatus := widget.NewLabel(i18n.Get("starting"))
		dlProgress := widget.NewProgressBar()
		dlCancel := widget.NewButton(i18n.Get("cancel_btn"), func() {
			cancelFlag.Store(true)
			cancelFunc()
			dlStatus.SetText(i18n.Get("canceling"))
		})
		dlContent := container.NewVBox(dlStatus, dlProgress, container.NewCenter(dlCancel))
		dlDialog := dialog.NewCustomWithoutButtons(i18n.Get("download_status"), applyGradient(dlContent), w)
		dlDialog.Resize(fyne.NewSize(440, 180))
		dlDialog.Show()

		go func() {
			baseFolder := cfg.DownloadFolder
			presetsFolder := filepath.Join(baseFolder, PresetsSubdir)
			// os.MkdirAll is already called on startup, but ensuring it here too doesn't hurt
			os.MkdirAll(presetsFolder, 0755)
			
			db, err := initDB(baseFolder)
			if err != nil {
				fyne.Do(func() {
					dlDialog.Hide()
					dialog.ShowError(fmt.Errorf(i18n.GetF("error_db", err)), w)
				})
				return
			}
			defer db.Close()

			total := len(ids)
			ok, skipped := 0, 0
			var failedIDs []DownloadResult
			client := &http.Client{Timeout: 30 * time.Second}

			for i, tid := range ids {
				if cancelFlag.Load() {
					break
				}

				// Find tone info
				var currentTone Tone
				mu.Lock()
				for _, t := range allResults {
					if t.ID == tid {
						currentTone = t
						break
					}
				}
				mu.Unlock()

				if currentTone.ID == "" {
					currentTone.ID = tid
					currentTone.Name = "Preset " + tid
				}

				if skipExisting {
					var exists int
					db.QueryRow("SELECT COUNT(*) FROM presets WHERE id = ?", tid).Scan(&exists)
					if exists > 0 {
						skipped++
						fyne.Do(func() { dlProgress.SetValue(float64(i+1) / float64(total)) })
						continue
					}
				}

				fyne.Do(func() { dlStatus.SetText(i18n.GetF("dl_progress", i+1, total, currentTone.Name)) })
				dlURL := fmt.Sprintf("%s/%s/", DeliverURL, tid)

				resp, err := doGet(ctx, client, dlURL)
				if err != nil {
					failedIDs = append(failedIDs, DownloadResult{ID: tid, Name: currentTone.Name, Error: err})
					fyne.Do(func() { dlProgress.SetValue(float64(i+1) / float64(total)) })
					continue
				}

				ct := resp.Header.Get("Content-Type")
				if strings.Contains(ct, "text/html") {
					resp.Body.Close()
					failedIDs = append(failedIDs, DownloadResult{ID: tid, Name: currentTone.Name, Error: fmt.Errorf(i18n.Get("error_html"))})
					fyne.Do(func() { dlProgress.SetValue(float64(i+1) / float64(total)) })
					continue
				}

				body, err := io.ReadAll(resp.Body)
				resp.Body.Close()
				if err != nil {
					failedIDs = append(failedIDs, DownloadResult{ID: tid, Name: currentTone.Name, Error: err})
					continue
				}

				fn := fmt.Sprintf("tone_%s.hlx", tid)
				cd := resp.Header.Get("Content-Disposition")
				if cd != "" {
					re := regexp.MustCompile(`filename="?([^"]+)"?`)
					m := re.FindStringSubmatch(cd)
					if len(m) > 1 {
						fn = m[1]
					}
				}
				fn = safeFilename(fn)
				if fn == "" {
					fn = fmt.Sprintf("tone_%s.hlx", tid)
				}

				localPath := filepath.Join(presetsFolder, fn)
				err = os.WriteFile(localPath, body, 0644)
				if err != nil {
					failedIDs = append(failedIDs, DownloadResult{ID: tid, Name: currentTone.Name, Error: err})
					continue
				}

				savePresetToDB(db, currentTone, fn, localPath)
				ok++
				fyne.Do(func() { dlProgress.SetValue(float64(i+1) / float64(total)) })
				time.Sleep(200 * time.Millisecond)
			}
			
			fyne.Do(func() {
				dlDialog.Hide()
				
				msg := i18n.GetF("report_title", ok, skipped, len(failedIDs))
				
				if len(failedIDs) > 0 {
					var errList []string
					var retryIDs []string
					for _, f := range failedIDs {
						errList = append(errList, fmt.Sprintf("- %s: %v", f.Name, f.Error))
						retryIDs = append(retryIDs, f.ID)
					}
					
					errEntry := widget.NewEntry()
					errEntry.MultiLine = true
					errEntry.SetText(strings.Join(errList, "\n"))
					content := container.NewVBox(
						widget.NewLabel(msg),
						widget.NewLabel(i18n.Get("report_detected")),
						errEntry,
					)
					
					retryDlg := dialog.NewCustomConfirm(i18n.Get("dl_error_title"), i18n.Get("dl_retry"), i18n.Get("close_btn"), applyGradient(content), func(yes bool) {
						if yes {
							doDownload(retryIDs, false)
						}
					}, w)
					retryDlg.Resize(fyne.NewSize(500, 400))
					retryDlg.Show()
				} else {
					dialog.ShowInformation(i18n.Get("completed_short"), msg, w)
				}
				statusLabel.SetText(i18n.GetF("dl_done_status", ok))
			})
		}()
	}


	btnDlSel := widget.NewButton(i18n.Get("btn_dl_sel"), func() {
		mu.Lock()
		ids := make([]string, 0, len(selectedIDs))
		for id := range selectedIDs {
			ids = append(ids, id)
		}
		mu.Unlock()
		if len(ids) == 0 {
			dialog.ShowInformation(i18n.Get("no_sel_title"), i18n.Get("no_sel_msg"), w)
			return
		}
		doDownload(ids, false)
	})

	var btnDlAll *widget.Button
	btnDlAll = widget.NewButton(i18n.Get("btn_dl_all"), func() {
		mu.Lock()
		n := len(allResults)
		mu.Unlock()
		if n == 0 {
			return
		}

		newCount := 0
		mu.Lock()
		for _, t := range allResults {
			if _, exists := localDB[t.ID]; !exists {
				newCount++
			}
		}
		already := n - newCount
		mu.Unlock()

		content := widget.NewLabel(i18n.GetF("dl_all_msg", n, already, newCount))
		content.Wrapping = fyne.TextWrapWord

		dlg := dialog.NewCustomConfirm(i18n.Get("dl_title"), i18n.Get("dl_all_btn"), i18n.Get("cancel"), content, func(ok bool) {
			if ok {
				mu.Lock()
				ids := make([]string, 0, len(allResults))
				for _, t := range allResults {
					ids = append(ids, t.ID)
				}
				mu.Unlock()
				doDownload(ids, false)
			}
		}, w)
		dlg.Show()
	})

	btnSearch.Importance = widget.HighImportance // Vibrant primary color

	// ── Layout ────────────────────────────────────────────────────────
	// Logo header with black background
	headerLogo := canvas.NewImageFromResource(logoResource)
	headerLogo.SetMinSize(fyne.NewSize(420, 110))
	headerLogo.FillMode = canvas.ImageFillContain
	
	headerBg := canvas.NewRectangle(color.Black)
	headerContainer := container.NewMax(headerBg, container.NewCenter(headerLogo))

	btnManual := widget.NewButtonWithIcon(i18n.Get("btn_help"), theme.HelpIcon(), func() {
		tmpFile := filepath.Join(os.TempDir(), "CTHD_Manual.html")
		err := os.WriteFile(tmpFile, manualHtmlBytes, 0644)
		if err != nil {
			dialog.ShowError(fmt.Errorf(i18n.GetF("error_manual", err)), w)
			return
		}
		absPath, _ := filepath.Abs(tmpFile)
		u, _ := url.Parse("file:///" + filepath.ToSlash(absPath))
		descargador.OpenURL(u)
	})

	var updateTexts func()
	var updateMainContent func()

	// Language Selector
	langSelector := i18n.NewLangSelector(w, cfg.Language, func(newLang string) {
		cfg.Language = newLang
		saveConfig(cfg)
		i18n.SetLang(newLang) // Added!
		// We will define updateTexts shortly below
		if updateTexts != nil {
			updateTexts()
		}
	})

	// Named labels for hot swap
	lblSearch := widget.NewLabel("🔍 " + i18n.Get("label_search"))
	lblSort := widget.NewLabel(i18n.Get("label_sort"))
	lblFrom := widget.NewLabel(i18n.Get("label_from"))
	lblTo := widget.NewLabel(i18n.Get("label_to"))
	lblFolder := widget.NewLabel(i18n.Get("label_choose_folder"))
	lblPerPage := widget.NewLabel(i18n.Get("label_per_page"))
	
	lblLocalFilter := widget.NewLabel(i18n.Get("filter_label"))
	
	// Consolidated Search row with distinctive background
	searchRow := container.NewHBox(
		lblSearch, container.NewGridWrap(fyne.NewSize(200, 35), searchEntry),
		lblSort, sortSelect,
		lblFrom, container.NewGridWrap(fyne.NewSize(120, 35), dateFromEntry), btnDateFrom,
		lblTo, container.NewGridWrap(fyne.NewSize(120, 35), dateToEntry), btnDateTo,
		btnSearch,
		layout.NewSpacer(),
		langSelector,
		btnManual,
		lblFolder, btnFolder, folderLabel,
	)
	
	searchBg := canvas.NewLinearGradient(color.NRGBA{R: 0, G: 0, B: 0, A: 100}, color.NRGBA{R: 0x1a, G: 0x16, B: 0x33, A: 200}, 90)
	searchContainer := container.NewMax(searchBg, container.NewPadded(searchRow))

	btnDlAll.Importance = widget.HighImportance // Make download colorful

	// Action and Filter row
	actionRow := container.NewHBox(
		btnSelectAll, btnDeselectAll, btnInvert,
		layout.NewSpacer(),
		actionGroup,
	)

	// Page nav
	pageNav := container.NewHBox(
		resultCountLabel, selectionLabel,
		layout.NewSpacer(),
		btnFirst, btnPrev, pageLabel, btnNext, btnLast,
		lblPerPage, pageSizeSelect,
	)

	// Bottom bar
	bottomBar := container.NewVBox(
		container.NewGridWithColumns(3, btnDlAll, btnDlSel, btnOpenFolder),
		container.NewBorder(nil, nil, statusLabel, progressBar),
	)

	// Main layout
	topSection := container.NewVBox(
		headerContainer,
		searchContainer,
		container.NewPadded(container.NewCenter(container.NewHBox(searchModeGlobal, lastSearchLabel))),
	)

	resultsSection := container.NewPadded(wrappedTable)
	
	// App Logic: If no folder, show a big "Set Output Folder" button
	setupFolderBtn := newPulsingButton("Establecer carpeta de salida", "Set Output Folder", theme.FolderOpenIcon(), func() {
		btnFolder.OnTapped()
	})
	
	lblWelcomeTitle := widget.NewLabelWithStyle(i18n.Get("welcome_title"), fyne.TextAlignCenter, fyne.TextStyle{Bold: true})
	lblWelcomeMsg := widget.NewLabel(i18n.Get("welcome_msg"))
	
	setupFolderContainer := container.NewCenter(
		container.NewVBox(
			lblWelcomeTitle,
			lblWelcomeMsg,
			layout.NewSpacer(),
			container.NewCenter(container.NewPadded(setupFolderBtn)),
			layout.NewSpacer(),
		),
	)

	var mainContent *fyne.Container
	var btnOpenDB *widget.Button
	
	updateTexts = func() {
		w.SetTitle(i18n.Get("app_title"))
		searchEntry.SetPlaceHolder(i18n.Get("term_placeholder"))
		localFilterEntry.SetPlaceHolder(i18n.Get("filter_results"))
		sortSelect.Options = getSortOptions()
		if len(sortSelect.Options) > 0 {
			sortSelect.SetSelected(sortSelect.Options[0]) 
		}
		
		lblSearch.SetText("🔍 " + i18n.Get("label_search"))
		lblSort.SetText(i18n.Get("label_sort"))
		lblFrom.SetText(i18n.Get("label_from"))
		lblTo.SetText(i18n.Get("label_to"))
		lblFolder.SetText(i18n.Get("label_choose_folder"))
		lblPerPage.SetText(i18n.Get("label_per_page"))
		lblLocalFilter.SetText(i18n.Get("filter_label"))
		
		btnSelectAll.SetText(i18n.Get("sel_all"))
		btnDeselectAll.SetText(i18n.Get("desel_all"))
		btnInvert.SetText(i18n.Get("invert_sel"))
		
		checkAndSearch.SetText(i18n.Get("and_search"))
		oldSel := searchModeGlobal.Selected
		searchModeGlobal.Options = []string{i18n.Get("global_search"), i18n.Get("since_last_search")}
		if oldSel == "Búsqueda global" || oldSel == "Global search" {
			searchModeGlobal.SetSelected(i18n.Get("global_search"))
		} else {
			searchModeGlobal.SetSelected(i18n.Get("since_last_search"))
		}
		searchModeGlobal.Refresh()
		
		btnSearch.SetText(i18n.Get("search_btn"))
		btnFolder.SetText(i18n.Get("folder_btn"))
		btnOpenFolder.SetText(i18n.Get("open_folder_btn"))
		btnManual.SetText(i18n.Get("btn_help"))
		
		btnDlAll.SetText(i18n.Get("btn_dl_all"))
		btnDlSel.SetText(i18n.Get("btn_dl_sel"))
		btnLocalFilter.SetText(i18n.Get("filter_btn"))
		if btnOpenDB != nil {
			btnOpenDB.SetText(i18n.Get("manager_db_btn"))
		}
		
		btnFirst.SetText(i18n.Get("nav_first"))
		btnPrev.SetText(i18n.Get("nav_prev"))
		btnNext.SetText(i18n.Get("nav_next"))
		btnLast.SetText(i18n.Get("nav_last"))
		
		// setupFolderBtn is a pulsingButton (custom widget), update its text field directly
		setupFolderBtn.topText.Text = "Establecer carpeta de salida"
		setupFolderBtn.bottomText.Text = "Set Output Folder"
		setupFolderBtn.topText.Refresh()
		setupFolderBtn.bottomText.Refresh()
		lblWelcomeTitle.SetText(i18n.Get("welcome_title"))
		lblWelcomeMsg.SetText(i18n.Get("welcome_msg"))
		
		// Refresh dynamic components
		refreshPage()
		updateMainContent()
	}

	updateMainContent = func() {
		// The base underlying application view
		baseAppContent := container.NewBorder(
			container.NewVBox(topSection, container.NewPadded(actionRow), container.NewPadded(pageNav)),
			container.NewPadded(bottomBar),
			nil, nil,
			resultsSection,
		)

		var finalContent fyne.CanvasObject
		if cfg.DownloadFolder == "" {
			// If no folder, show base app covered by the intercepting overlay, and the pulsing button on top
			overlay := newInterceptingOverlay()
			finalContent = container.NewMax(
				baseAppContent,
				overlay,
				setupFolderContainer,
			)
		} else {
			// Normal usable app
			finalContent = baseAppContent
		}
		mainContent.Objects = []fyne.CanvasObject{finalContent}
		mainContent.Refresh()
	}

	mainContent = container.NewMax()
	
	// Override btnFolder behavior to also update UI
	btnFolder.OnTapped = func() {
		dialog.ShowFolderOpen(func(lu fyne.ListableURI, err error) {
			if lu != nil {
				path := lu.Path()
				cfg.DownloadFolder = path
				folderLabel.SetText(path)
				saveConfig(cfg)
				localDB = loadLocalDB(path)
				updateMainContent()
			}
		}, w)
	}
	setupFolderBtn.onTap = btnFolder.OnTapped

	updateMainContent()
	w.SetContent(mainContent)

	// --- Database Manager Logic ---
	showDBManager := func() {
		dbWin := descargador.NewWindow(i18n.Get("manager_db_title"))
		dbWin.Resize(fyne.NewSize(1000, 700))

		db, err := initDB(cfg.DownloadFolder)
		if err != nil {
			dialog.ShowError(err, dbWin)
			return
		}

		searchEntry := widget.NewEntry()
		searchEntry.SetPlaceHolder(i18n.Get("search_full_ph"))
		
		var dbResults []Tone
		var dbFilenames []string
		
		dbTable := widget.NewTable(
			func() (int, int) { return len(dbResults) + 1, 7 },
			func() fyne.CanvasObject { 
				bg := canvas.NewRectangle(color.Transparent)
				l := widget.NewLabel("Template")
				l.Wrapping = fyne.TextTruncate
				btn := widget.NewButtonWithIcon("", theme.FolderOpenIcon(), nil)
				btn.Importance = widget.LowImportance
				return container.NewMax(bg, l, btn)
			},
			func(id widget.TableCellID, o fyne.CanvasObject) {
				cont := o.(*fyne.Container)
				bg := cont.Objects[0].(*canvas.Rectangle)
				l := cont.Objects[1].(*widget.Label)
				btn := cont.Objects[2].(*widget.Button)

				if id.Row == 0 {
					btn.Hide()
					l.Show()
					bg.FillColor = color.Transparent
					l.TextStyle = fyne.TextStyle{Bold: true}
					headers := []string{i18n.Get("col_name"), i18n.Get("col_style"), i18n.Get("col_band"), i18n.Get("col_author"), i18n.Get("col_date"), i18n.Get("col_file"), i18n.Get("col_action")}
					if id.Col < len(headers) {
						l.SetText(headers[id.Col])
					}
					return
				}

				if id.Row%2 == 0 {
					bg.FillColor = color.NRGBA{R: 0x2A, G: 0x26, B: 0x43, A: 0xFF}
				} else {
					bg.FillColor = color.Transparent
				}
				bg.Refresh()

				if id.Col == 6 {
					l.Hide()
					btn.Show()
					btn.SetText(i18n.Get("view_in_explorer"))
					btn.OnTapped = func() {
						idx := id.Row - 1
						if idx >= 0 && idx < len(dbFilenames) {
							fn := dbFilenames[idx]
							full := filepath.Join(cfg.DownloadFolder, PresetsSubdir, fn)
							// Correct Windows explorer /select syntax by letting Go handle quoting of the whole argument
							exec.Command("explorer", "/select,", full).Start()
						}
					}
					return
				}

				btn.Hide()
				l.Show()
				l.TextStyle = fyne.TextStyle{}
				if id.Row-1 >= len(dbResults) { return }
				t := dbResults[id.Row-1]
				switch id.Col {
				case 0: l.SetText(t.Name)
				case 1: l.SetText(t.Style)
				case 2: l.SetText(t.Band)
				case 3: l.SetText(t.Author)
				case 4: l.SetText(formatDate(t.Date))
				case 5: l.SetText(dbFilenames[id.Row-1])
				}
			},
		)

		dbTable.OnSelected = func(id widget.TableCellID) {
			if id.Row > 0 {
				row := id.Row - 1
				localPath := filepath.Join(cfg.DownloadFolder, PresetsSubdir, dbFilenames[row])
				// Show path in status or allow opening
				log.Println("Selected:", localPath)
			}
			dbTable.UnselectAll()
		}

		// Fixed column widths
		dbTable.SetColumnWidth(0, 220)
		dbTable.SetColumnWidth(1, 120)
		dbTable.SetColumnWidth(2, 120)
		dbTable.SetColumnWidth(3, 120)
		dbTable.SetColumnWidth(4, 90)
		dbTable.SetColumnWidth(5, 180)
		dbTable.SetColumnWidth(6, 160)

		refreshDB := func() {
			term := "%" + searchEntry.Text + "%"
			rows, err := db.Query(`
				SELECT id, name, style, band, author, date, downloads, filename 
				FROM presets 
				WHERE name LIKE ? OR style LIKE ? OR author LIKE ? OR band LIKE ? 
				ORDER BY download_date DESC`, term, term, term, term)
			if err != nil {
				log.Println("DB Query Error:", err)
				return
			}
			defer rows.Close()

			dbResults = []Tone{}
			dbFilenames = []string{}
			for rows.Next() {
				var t Tone
				var fn string
				rows.Scan(&t.ID, &t.Name, &t.Style, &t.Band, &t.Author, &t.Date, &t.Downloads, &fn)
				dbResults = append(dbResults, t)
				dbFilenames = append(dbFilenames, fn)
			}
			dbTable.Refresh()
		}

		searchEntry.OnChanged = func(s string) { refreshDB() }
		
		btnOpenPresets := widget.NewButtonWithIcon(i18n.Get("open_presets_exp"), theme.FolderOpenIcon(), func() {
			p := filepath.Join(cfg.DownloadFolder, PresetsSubdir)
			os.MkdirAll(p, 0755)
			openFolder(p)
		})

		managerContent := container.NewBorder(
			container.NewVBox(
				container.NewPadded(container.NewBorder(nil, nil, widget.NewLabel(i18n.Get("filter_label")), nil, searchEntry)),
				container.NewCenter(btnOpenPresets),
			),
			nil, nil, nil,
			container.NewPadded(dbTable),
		)

		dbWin.SetContent(applyGradient(managerContent))
		dbWin.Show()
		refreshDB()
		
		dbWin.SetOnClosed(func() { db.Close() })
	}

	btnOpenDB = widget.NewButtonWithIcon(i18n.Get("manager_db_btn"), theme.StorageIcon(), func() {
		// Dialog to choose whether to open existing or change folder
		var dlg dialog.Dialog
		
		btnOpenCurrent := widget.NewButtonWithIcon(i18n.GetF("manager_db_open", cfg.DownloadFolder), theme.FolderIcon(), func() {
			dlg.Hide()
			managerPath := filepath.Join(cfg.DownloadFolder, "CTHD_Manager.exe")
			if _, err := os.Stat(managerPath); err == nil {
				exec.Command(managerPath, "--no-splash").Start()
			} else {
				showDBManager()
			}
		})
		btnOpenCurrent.Importance = widget.HighImportance

		btnChangeFolder := widget.NewButtonWithIcon(i18n.Get("manager_db_change"), theme.SearchIcon(), func() {
			dlg.Hide()
			dialog.ShowFolderOpen(func(lu fyne.ListableURI, err error) {
				if lu != nil {
					selectedFolder := lu.Path()
					cfg.DownloadFolder = selectedFolder
					folderLabel.SetText(selectedFolder)
					saveConfig(cfg)
					localDB = loadLocalDB(selectedFolder)
					deployManager(selectedFolder)
					
					managerPath := filepath.Join(selectedFolder, "CTHD_Manager.exe")
					if _, err := os.Stat(managerPath); err == nil {
						exec.Command(managerPath, "--no-splash").Start()
					} else {
						showDBManager()
					}
				}
			}, w)
		})
		
		content := container.NewVBox(
			widget.NewLabelWithStyle(i18n.Get("manager_where_open"), fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
			layout.NewSpacer(),
			container.NewGridWrap(fyne.NewSize(400, 60), btnOpenCurrent),
			layout.NewSpacer(),
			container.NewGridWrap(fyne.NewSize(400, 50), btnChangeFolder),
		)
		
		dlg = dialog.NewCustom(i18n.Get("manager_options"), i18n.Get("cancel"), container.NewPadded(content), w)
		dlg.Resize(fyne.NewSize(450, 250))
		dlg.Show()
	})
	btnOpenDB.Importance = widget.WarningImportance

	// Deploy manager on startup
	deployManager(cfg.DownloadFolder)

	// Update Bottom Bar Layout
	bottomBar.Objects[0] = container.NewGridWithColumns(4, btnDlAll, btnDlSel, btnOpenDB, btnOpenFolder)

	w.CenterOnScreen()
	w.Show()

	MaximizeWindow(w)
	})

	descargador.Run()
}

// openFolder opens the download folder in the OS file manager
func openFolder(path string) {
	OpenFolder(path)
}
