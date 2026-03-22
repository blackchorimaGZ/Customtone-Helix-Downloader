package main

import (
	"database/sql"
	_ "embed"
	"fmt"
	"image/color"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	_ "modernc.org/sqlite"
	"encoding/json"

	"descargadorhelix/i18n"
)

//go:embed Icon.png
var logoBytes []byte

// --- Copy of Theme and Constants from main.go ---
const (
	AppTitle      = "CTHD Manager"
	AppVersion    = ""
	DBFilename    = "presets.db"
	PresetsSubdir = "PRESETS"
)

var (
	colBG      = color.NRGBA{R: 0x0D, G: 0x0B, B: 0x1E, A: 0xFF}
	colSurface = color.NRGBA{R: 0x1A, G: 0x16, B: 0x33, A: 0xFF}
	colAccent  = color.NRGBA{R: 0x00, G: 0xBF, B: 0xFF, A: 0xFF}
	colFG      = color.NRGBA{R: 0xFF, G: 0xFF, B: 0xFF, A: 0xFF} // Pure White
	colDimFG   = color.NRGBA{R: 0xC0, G: 0xC0, B: 0xE0, A: 0xFF} // Brighter Dim
	colRed     = color.NRGBA{R: 0xFF, G: 0x30, B: 0x30, A: 0xFF}
	colGreen   = color.NRGBA{R: 0x30, G: 0xE0, B: 0x80, A: 0xFF}
	colYellow  = color.NRGBA{R: 0xFF, G: 0xD0, B: 0x40, A: 0xFF}
)

type neonTheme struct{}

func (n *neonTheme) Color(name fyne.ThemeColorName, v fyne.ThemeVariant) color.Color {
	switch name {
	case theme.ColorNameBackground: return colBG
	case theme.ColorNameForeground: return colFG
	case theme.ColorNameButton: return colSurface
	case theme.ColorNamePrimary: return colAccent
	case theme.ColorNameFocus: return colAccent
	case theme.ColorNameSelection: return color.NRGBA{R: 0, G: 0xBF, B: 0xFF, A: 0x40}
	case theme.ColorNameInputBackground: return colSurface
	case theme.ColorNameInputBorder: return colDimFG
	case theme.ColorNamePlaceHolder: return colDimFG
	case theme.ColorNameDisabled: return colDimFG
	case theme.ColorNameScrollBar: return colDimFG
	case theme.ColorNameHeaderBackground: return color.NRGBA{R: 0x15, G: 0x12, B: 0x2B, A: 0xFF}
	case theme.ColorNameShadow: return color.Transparent
	case theme.ColorNameSuccess: return colGreen
	case theme.ColorNameError: return colRed
	case theme.ColorNameWarning: return colYellow
	case theme.ColorNameOverlayBackground: return color.Black
	default: return theme.DefaultTheme().Color(name, theme.VariantDark)
	}
}

func (n *neonTheme) Font(s fyne.TextStyle) fyne.Resource { return theme.DefaultTheme().Font(s) }
func (n *neonTheme) Icon(name fyne.ThemeIconName) fyne.Resource { return theme.DefaultTheme().Icon(name) }
func (n *neonTheme) Size(name fyne.ThemeSizeName) float32 {
	if name == theme.SizeNamePadding { return 6 }
	if name == theme.SizeNameInputRadius || name == theme.SizeNameSelectionRadius { return 8 }
	return theme.DefaultTheme().Size(name)
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

type Tone struct {
	ID        string
	Name      string
	Style     string
	Band      string
	Author    string
	Date      string
	Downloads string
}

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
	
	tw.table.SetColumnWidth(0, avail*0.25)
	tw.table.SetColumnWidth(1, avail*0.10)
	tw.table.SetColumnWidth(2, avail*0.15)
	tw.table.SetColumnWidth(3, avail*0.12)
	tw.table.SetColumnWidth(4, avail*0.08)
	tw.table.SetColumnWidth(5, avail*0.15)
	tw.table.SetColumnWidth(6, avail*0.15)
}

func (tw *tableWrapper) MinSize() fyne.Size {
	return tw.table.MinSize()
}

func parseDate(s string) (time.Time, bool) {
	s = strings.TrimSpace(s)
	for _, fmt := range []string{
		"1/2/06", "01/02/06", "1/02/06", "01/2/06", "1/2/2006", "01/02/2006",
		"2006-01-02", "02/01/2006", "Jan 2, 2006",
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

func main() {
	exe, _ := os.Executable()
	appDir := filepath.Dir(exe)
	
	type TempConfig struct {
		Language string `json:"language"`
	}
	cfg := TempConfig{Language: "es"}
	cfgFile := filepath.Join(appDir, "descargador_config.json")
	if data, err := os.ReadFile(cfgFile); err == nil {
		json.Unmarshal(data, &cfg)
	}
	if cfg.Language == "" { cfg.Language = "es" }
	i18n.SetLang(cfg.Language)

	a := app.NewWithID("com.descargadorhelix.manager")

	runManager := func() {
		w := a.NewWindow(i18n.Get("manager_title"))
		w.Resize(fyne.NewSize(1000, 700))

		// Diálogo inicial de selección de ruta
		var showSelectionDialog func()

		
		startWithFolder := func(targetDir string) {
			dbPath := filepath.Join(targetDir, DBFilename)
			if _, err := os.Stat(dbPath); os.IsNotExist(err) {
				dialog.ShowConfirm(i18n.Get("db_not_found"), 
					fmt.Sprintf(i18n.Get("db_not_found_msg"), targetDir), 
					func(yes bool) {
						if yes { showSelectionDialog() } else { a.Quit() }
					}, w)
				return
			}

			db, err := sql.Open("sqlite", dbPath)
			if err != nil {
				dialog.ShowError(err, w)
				return
			}
			
			currentAppDir := targetDir

	searchEntry := widget.NewEntry()
	searchEntry.SetPlaceHolder(i18n.Get("manager_search_ph"))

	var dbResults []Tone
	var dbFilenames []string

	dbTable := widget.NewTable(
		func() (int, int) { return len(dbResults) + 1, 7 },
		func() fyne.CanvasObject {
			bg := canvas.NewRectangle(color.Transparent)
			l := canvas.NewText("Template", colFG)
			l.Alignment = fyne.TextAlignLeading
			btn := widget.NewButtonWithIcon("", theme.FolderOpenIcon(), nil)
			btn.Importance = widget.LowImportance

			h := canvas.NewText("", color.Black)
			h.TextStyle = fyne.TextStyle{Bold: true}
			h.Alignment = fyne.TextAlignCenter
			h.Hide()

			return container.NewMax(bg, l, btn, h)
		},
		func(id widget.TableCellID, o fyne.CanvasObject) {
			cont := o.(*fyne.Container)
			bg := cont.Objects[0].(*canvas.Rectangle)
			var l *canvas.Text
			var btn *widget.Button
			var h *canvas.Text
			for _, obj := range cont.Objects {
				if tx, ok := obj.(*canvas.Text); ok {
					if tx.Alignment == fyne.TextAlignLeading {
						l = tx
					} else {
						h = tx
					}
				} else if bt, ok := obj.(*widget.Button); ok {
					btn = bt
				}
			}

			if id.Row == 0 {
				btn.Hide()
				l.Hide()
				h.Show()
				bg.FillColor = color.NRGBA{R: 0xFF, G: 0xD0, B: 0x40, A: 0xFF} // Bright Yellow
				bg.Refresh()
				
				h.Color = color.Black
				h.TextSize = theme.TextSize()
				headers := []string{i18n.Get("col_name"), i18n.Get("col_style"), i18n.Get("col_band"), i18n.Get("col_author"), i18n.Get("col_date"), i18n.Get("col_file"), i18n.Get("col_action")}
				if id.Col < len(headers) {
					h.Text = headers[id.Col]
				}
				h.Refresh()
				return
			}

			h.Hide()
			// Background colors
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
						OpenExplorer(filepath.Join(currentAppDir, PresetsSubdir, dbFilenames[idx]))
					}
				}
				return
			}

			btn.Hide()
			l.Show()
			l.TextStyle = fyne.TextStyle{}
			l.Color = color.White // Explicitly white for contrast
			idx := id.Row - 1
			if idx >= len(dbResults) { return }
			
			t := dbResults[idx]
			switch id.Col {
			case 0: l.Text = t.Name
			case 1: l.Text = t.Style
			case 2: l.Text = t.Band
			case 3: l.Text = t.Author
			case 4: l.Text = formatDate(t.Date)
			case 5: l.Text = dbFilenames[idx]
			case 6: l.Text = i18n.Get("select_action")
			}
			l.Refresh()
		},
	)

	// Add context menu to table rows
	// Note: In this standalone manager, we'll try to use a more "explorer" feel.
	// Since Fyne Table doesn't allow easy custom widgets per cell (with buttons), 
	// we use a Context Menu or a separate action bar.
	
	copyFile := func(filename string) {
		dialog.ShowFolderOpen(func(lu fyne.ListableURI, err error) {
			if lu == nil || err != nil { return }
			dest := filepath.Join(lu.Path(), filename)
			src := filepath.Join(appDir, PresetsSubdir, filename)
			
			in, err := os.Open(src)
			if err != nil { dialog.ShowError(err, w); return }
			defer in.Close()
			
			out, err := os.Create(dest)
			if err != nil { dialog.ShowError(err, w); return }
			defer out.Close()
			
			_, err = io.Copy(out, in)
			if err != nil { dialog.ShowError(err, w); return }
			
			dialog.ShowInformation(i18n.Get("success_title"), fmt.Sprintf(i18n.Get("copied_msg"), dest), w)
		}, w)
	}

	dbTable.OnSelected = func(id widget.TableCellID) {
		if id.Row > 0 {
			row := id.Row - 1
			fn := dbFilenames[row]
			
			// Selection logic on row tap
			// We can't easily get the absolute position here without MouseDown,
			// so we'll show it as a modal or just use a helper.
			// Let's use a simple dialog choice for now to be robust.
			dialog.ShowCustom(fmt.Sprintf(i18n.Get("actions_for"), fn), i18n.Get("close_btn"), container.NewVBox(
				widget.NewButton(i18n.Get("open_explorer_menu"), func() {
					OpenExplorer(filepath.Join(currentAppDir, PresetsSubdir, fn))
				}),
				widget.NewButton(i18n.Get("copy_to_menu"), func() {
					copyFile(fn)
				}),
			), w)
		}
		dbTable.UnselectAll()
	}

	dbTable.SetColumnWidth(0, 220)
	dbTable.SetColumnWidth(1, 120)
	dbTable.SetColumnWidth(2, 120)
	dbTable.SetColumnWidth(3, 120)
	dbTable.SetColumnWidth(4, 90)
	dbTable.SetColumnWidth(5, 180)
	dbTable.SetColumnWidth(6, 160)

	refreshDB := func() {
		txt := strings.TrimSpace(searchEntry.Text)
		
		var query string
		var args []interface{}
		// Load all if empty or dot
		if txt == "" || txt == "." {
			query = "SELECT id, name, style, band, author, date, downloads, filename FROM presets ORDER BY download_date DESC"
		} else {
			term := "%" + txt + "%"
			query = `SELECT id, name, style, band, author, date, downloads, filename 
					 FROM presets 
					 WHERE name LIKE ? OR style LIKE ? OR author LIKE ? OR band LIKE ? 
					 ORDER BY download_date DESC`
			args = []interface{}{term, term, term, term}
		}

		rows, err := db.Query(query, args...)
		if err != nil {
			log.Println("Query error:", err)
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
	searchEntry.OnSubmitted = func(s string) { refreshDB() }

	btnOpenPresets := widget.NewButtonWithIcon(i18n.Get("open_presets_btn"), theme.FolderOpenIcon(), func() {
		p := filepath.Join(appDir, PresetsSubdir)
		OpenFolder(p)
	})

	lblSearch := widget.NewLabel(i18n.Get("search_label"))

	langSelector := i18n.NewLangSelector(w, cfg.Language, func(newLang string) {
		cfg.Language = newLang
		data, _ := os.ReadFile(cfgFile)
		var fullCfg map[string]interface{}
		json.Unmarshal(data, &fullCfg)
		if fullCfg == nil { fullCfg = make(map[string]interface{}) }
		fullCfg["language"] = newLang
		newData, _ := json.MarshalIndent(fullCfg, "", "  ")
		os.WriteFile(cfgFile, newData, 0644)
		
		// Hot Reload Text updates
		i18n.SetLang(newLang)
		w.SetTitle(i18n.Get("manager_title"))
		searchEntry.SetPlaceHolder(i18n.Get("manager_search_ph"))
		btnOpenPresets.SetText(i18n.Get("open_presets_btn"))
		lblSearch.SetText(i18n.Get("search_label"))
		dbTable.Refresh() // refreshes headers + row texts natively
	})

	content := container.NewBorder(
		container.NewVBox(
			container.NewHBox(layout.NewSpacer(), langSelector),
			container.NewPadded(container.NewBorder(nil, nil, lblSearch, nil, searchEntry)),
			container.NewCenter(btnOpenPresets),
		),
		nil, nil, nil,
		container.NewPadded(&tableWrapper{table: dbTable}),
	)
		w.SetContent(applyGradient(content))
		w.CenterOnScreen()
		w.Show()
		MaximizeWindow(w)
		refreshDB()
		
		w.SetOnClosed(func() {
			db.Close()
		})
	}

	showSelectionDialog = func() {
		btnCurrent := widget.NewButtonWithIcon(i18n.GetF("open_current_path", appDir), theme.FolderIcon(), func() {
			startWithFolder(appDir)
		})
		btnCurrent.Importance = widget.HighImportance

		btnChoose := widget.NewButtonWithIcon(i18n.Get("choose_other_folder"), theme.SearchIcon(), func() {
			dialog.ShowFolderOpen(func(lu fyne.ListableURI, err error) {
				if lu != nil {
					startWithFolder(lu.Path())
				}
			}, w)
		})

		content := container.NewVBox(
			widget.NewLabelWithStyle(i18n.Get("which_db_msg"), fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
			layout.NewSpacer(),
			container.NewGridWrap(fyne.NewSize(400, 70), btnCurrent),
			layout.NewSpacer(),
			container.NewGridWrap(fyne.NewSize(400, 40), btnChoose),
		)
		w.SetContent(applyGradient(container.NewCenter(content)))
		w.CenterOnScreen()
		w.Show()
		MaximizeWindow(w)
	}

	showSelectionDialog()
}

skipSplash := false
for _, arg := range os.Args {
	if arg == "--no-splash" {
		skipSplash = true
		break
	}
}

if skipSplash {
	runManager()
} else {
	showSplash(a, false, runManager)
}

a.Run()
}
