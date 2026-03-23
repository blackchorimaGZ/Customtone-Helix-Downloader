package i18n

import (
	"fmt"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

var FlagESResource = fyne.NewStaticResource("es.svg", []byte(`
<svg width="60" height="40" xmlns="http://www.w3.org/2000/svg">
  <rect width="60" height="10" fill="#AA151B"/>
  <rect y="10" width="60" height="20" fill="#F1BF00"/>
  <rect y="30" width="60" height="10" fill="#AA151B"/>
</svg>`))

var FlagENResource = fyne.NewStaticResource("en.svg", []byte(`
<svg width="60" height="40" viewBox="0 0 60 40" xmlns="http://www.w3.org/2000/svg">
  <rect width="60" height="40" fill="#012169"/>
  <path d="M0,0 L60,40 M60,0 L0,40" stroke="#fff" stroke-width="6"/>
  <path d="M0,0 L60,40 M60,0 L0,40" stroke="#C8102E" stroke-width="4"/>
  <path d="M30,0 v40 M0,20 h60" stroke="#fff" stroke-width="10"/>
  <path d="M30,0 v40 M0,20 h60" stroke="#C8102E" stroke-width="6"/>
</svg>`))

var dict = map[string]map[string]string{
	"es": {
		"app_title":            "CustomTone Helix Downloader",
		"manager_title":        "CTHD Manager",
		"starting":             "Iniciando...",
		"closing":              "Cerrando...",
		"cancel":               "Cancelar",
		"sort_rating":          "⭐ Mejor valorados",
		"sort_recent":          "🕐 Más recientes",
		"sort_downloads":       "📥 Más descargados",
		"sort_name":            "🔤 Nombre",
		"months":               "Enero,Febrero,Marzo,Abril,Mayo,Junio,Julio,Agosto,Septiembre,Octubre,Noviembre,Diciembre",
		"days":                 "L,M,X,J,V,S,D",
		"col_name":             "Nombre del Preset",
		"col_style":            "Estilo",
		"col_band":             "Banda / Artista",
		"col_author":           "Autor",
		"col_date":             "Fecha",
		"col_downloads":        "Descargas",
		"col_file":             "Archivo",
		"col_action":           "Acción",
		"selected_out_of":      "%d seleccionados de %d",
		"term_placeholder":     "Término (vacío = todos)",
		"select_placeholder":   "Seleccionar",
		"and_search":           "Búsqueda AND",
		"global_search":        "Búsqueda global",
		"since_last_search":    "Desde última búsqueda",
		"filter_results":       "🔎 Filtrar dentro de los resultados...",
		"filter_btn":           "Filtrar",
		"nav_first":            "« Primero",
		"nav_prev":             "‹ Ant.",
		"nav_next":             "Sig. ›",
		"nav_last":             "Último »",
		"sel_all":              "✅ Seleccionar todos",
		"desel_all":            "❌ Deseleccionar todos",
		"invert_sel":           "🔄 Invertir",
		"folder_btn":           "📂 Carpeta",
		"open_folder_btn":      "📁 Abrir carpeta",
		"search_btn":           "🔎 BUSCAR",
		"folder_req_title":     "Carpeta requerida",
		"folder_req_msg":       "Por favor, selecciona una carpeta de destino para las descargas.",
		"cancel_btn":           "⛔ Cancelar",
		"canceling":            "Cancelando…",
		"searching_title":      "Buscando en CustomTone…",
		"start_search":         "🔍 Iniciando búsqueda…",
		"probing":              "🔍 Sondeando CustomTone…",
		"conn_error":           "error de conexión: %v",
		"html_error":           "error parseando HTML: %v",
		"no_results_title":     "Sin resultados",
		"no_results_msg":       "No se encontraron tones.",
		"pages_threads":        "Páginas: %d/%d · Hilos: %d",
		"fetching_pages":       "🔍 %d/%d páginas — obteniendo…",
		"first_page_probe":     "🔍 1/%d páginas — %d presets",
		"download_menu":        "⬇ Descargar",
		"view_ct_menu":         "🔗 Ver en CustomTone",
		"restart_req":          "Idioma cambiado",
		"restart_msg":          "Textos cambiados al nuevo idioma.",
		"manager_search_ph":    "Buscar... (usa '.' para ver todos)",
		"view_explorer_btn":    "Ver en explorador",
		"select_action":        "➡ Seleccionar",
		"actions_for":          "Acciones para: %s",
		"close_btn":            "Cerrar",
		"open_explorer_menu":   "📁 Abrir en Explorador",
		"copy_to_menu":         "💾 Copiar a...",
		"success_title":        "Éxito",
		"copied_msg":           "Archivo copiado a: %s",
		"open_presets_btn":     "Abrir Carpeta PRESETS",
		"search_label":         "Buscar: ",
		"db_not_found":         "Base de datos no encontrada",
		"db_not_found_msg":     "No se encontró 'presets.db' en:\n%s\n\n¿Deseas elegir otra carpeta?",
		"which_db_msg":         "¿Qué base de datos deseas gestionar?",
		"open_current_path":    "Abrir en ruta actual:\n%s",
		"choose_other_folder":  "Elegir otra carpeta...",
		"downloading_presets":  "Descargando Presets",
		"downloading_status":   "Descargando %d/%d...",
		"completed_title":      "Descarga Completada",
		"completed_msg":        "Se han procesado %d presets (%d exitosos, %d errores, %d saltados).",
		"download_failed":      "Fallo",
		"download_skipped":     "Saltado",
		"download_success":     "OK",
		"status_ready":         "Listo",
		"date_from":            "Desde",
		"date_to":              "Hasta",
		"app_version":          " v",
		"language":             "Idioma",

		// More missing translations
		"set_folder":           "Establecer carpeta de salida / Set Output Folder",
		"welcome_title":        "Bienvenido a CustomTone Downloader / Welcome to CustomTone Downloader",
		"welcome_msg":          "Para comenzar, selecciona una carpeta donde se guardarán los presets.\nTo begin, select a folder where your presets will be saved.",
		"manager_db_title":     "Gestor de Base de Datos - CustomTone",
		"search_full_ph":       "Buscar por nombre, estilo, autor, banda...",
		"view_in_explorer":     "Ver en explorador",
		"open_presets_exp":     "Abrir Explorador PRESETS",
		"filter_label":         "Filtrar: ",
		"manager_db_btn":       "🗄️ Gestor de presets descargados",
		"manager_db_open":      "Abrir gestor en:\n%s",
		"manager_db_change":    "Cambiar ruta y abrir...",
		"manager_where_open":   "¿Dónde quieres abrir el gestor de la base de datos?",
		"manager_options":      "Opciones del Gestor",
		"download_status":      "Descargando…",
		"dl_progress":          "Descargando %d/%d: %s…",
		"error_db":             "error base de datos: %v",
		"error_html":           "tone no disponible (HTML)",
		"report_title":         "📊 Informe:\n\n✅ Descargados: %d\n⏭ Omitidos: %d\n❌ Errores: %d",
		"report_detected":      "\nErrores detectados:",
		"dl_error_title":       "Descarga con errores",
		"dl_retry":             "Reintentar fallidos",
		"dl_done_status":       "Descarga finalizada: %d archivos",
		"btn_dl_sel":           "✅ Descargar SELECCIONADOS",
		"no_sel_title":         "Sin selección",
		"no_sel_msg":           "Selecciona al menos un preset.",
		"btn_dl_all":           "⬇ Descargar TODOS",
		"dl_all_msg":           "Se encontraron %d presets.\nYa descargados: %d\nNuevos: %d\n\n¿Qué deseas descargar?",
		"dl_title":             "Descargar",
		"dl_all_btn":           "Descargar TODOS",
		"btn_help":             "Ayuda",
		"error_manual":         "error al extraer el manual: %v",
		"label_search":         "🔍 Buscar:",
		"label_sort":           "Ordenar:",
		"label_from":           " Desde:",
		"label_to":             " Hasta:",
		"label_choose_folder":  "Elegir carpeta de descarga:",
		"label_per_page":       "    Por página:",
		"results_found":        "%d presets encontrados",
		"results_overall":      "✅ %d presets (de %d totales)",
		"dp_title_from":        "Desde",
		"dp_title_to":          "Hasta",
		"dl_new_only_btn":      "Solo nuevos",
		"completed_short":      "Descarga completada",
		"error_init_db":        "Error inicializando DB en el arranque",
		"btn_donate":           "☕ Donar",
		"btn_theme_dark":       "🌙 Oscuro",
		"btn_theme_light":      "☀️ Claro",
	},
	"en": {
		"app_title":            "CustomTone Helix Downloader",
		"manager_title":        "CTHD Manager",
		"starting":             "Starting...",
		"closing":              "Closing...",
		"cancel":               "Cancel",
		"sort_rating":          "⭐ Top Rated",
		"sort_recent":          "🕐 Most Recent",
		"sort_downloads":       "📥 Most Downloaded",
		"sort_name":            "🔤 Name",
		"months":               "January,February,March,April,May,June,July,August,September,October,November,December",
		"days":                 "M,T,W,T,F,S,S",
		"col_name":             "Preset Name",
		"col_style":            "Style",
		"col_band":             "Band / Artist",
		"col_author":           "Author",
		"col_date":             "Date",
		"col_downloads":        "Downloads",
		"col_file":             "File",
		"col_action":           "Action",
		"selected_out_of":      "%d selected out of %d",
		"term_placeholder":     "Term (empty = all)",
		"select_placeholder":   "Select",
		"and_search":           "AND Search",
		"global_search":        "Global search",
		"since_last_search":    "Since last search",
		"filter_results":       "🔎 Filter within results...",
		"filter_btn":           "Filter",
		"nav_first":            "« First",
		"nav_prev":             "‹ Prev",
		"nav_next":             "Next ›",
		"nav_last":             "Last »",
		"sel_all":              "✅ Select All",
		"desel_all":            "❌ Deselect All",
		"invert_sel":           "🔄 Invert Selection",
		"folder_btn":           "📂 Folder",
		"open_folder_btn":      "📁 Open Folder",
		"search_btn":           "🔎 SEARCH",
		"folder_req_title":     "Folder required",
		"folder_req_msg":       "Please select a destination folder for your downloads.",
		"cancel_btn":           "⛔ Cancel",
		"canceling":            "Canceling…",
		"searching_title":      "Searching CustomTone…",
		"start_search":         "🔍 Starting search…",
		"probing":              "🔍 Probing CustomTone…",
		"conn_error":           "connection error: %v",
		"html_error":           "error parsing HTML: %v",
		"no_results_title":     "No results",
		"no_results_msg":       "No tones were found.",
		"pages_threads":        "Pages: %d/%d · Threads: %d",
		"fetching_pages":       "🔍 %d/%d pages — fetching…",
		"first_page_probe":     "🔍 1/%d pages — %d presets",
		"download_menu":        "⬇ Download",
		"view_ct_menu":         "🔗 View on CustomTone",
		"restart_req":          "Language Changed",
		"restart_msg":          "Texts changed to the new language.",
		"manager_search_ph":    "Search... (use '.' to view all)",
		"view_explorer_btn":    "View in Explorer",
		"select_action":        "➡ Select",
		"actions_for":          "Actions for: %s",
		"close_btn":            "Close",
		"open_explorer_menu":   "📁 Open in Explorer",
		"copy_to_menu":         "💾 Copy to...",
		"success_title":        "Success",
		"copied_msg":           "File copied to: %s",
		"open_presets_btn":     "Open PRESETS Folder",
		"search_label":         "Search: ",
		"db_not_found":         "Database not found",
		"db_not_found_msg":     "'presets.db' not found in:\n%s\n\nChoose another folder?",
		"which_db_msg":         "Which database do you want to manage?",
		"open_current_path":    "Open in current path:\n%s",
		"choose_other_folder":  "Choose another folder...",
		"downloading_presets":  "Downloading Presets",
		"downloading_status":   "Downloading %d/%d...",
		"completed_title":      "Download Completed",
		"completed_msg":        "Processed %d presets (%d successful, %d failed, %d skipped).",
		"download_failed":      "Failed",
		"download_skipped":     "Skipped",
		"download_success":     "OK",
		"status_ready":         "Ready",
		"date_from":            "From",
		"date_to":              "To",
		"app_version":          " v",
		"language":             "Language",

		"set_folder":           "Establecer carpeta de salida / Set Output Folder",
		"welcome_title":        "Bienvenido a CustomTone Downloader / Welcome to CustomTone Downloader",
		"welcome_msg":          "Para comenzar, selecciona una carpeta donde se guardarán los presets.\nTo begin, select a folder where your presets will be saved.",
		"manager_db_title":     "Database Manager - CustomTone",
		"search_full_ph":       "Search by name, style, author, band...",
		"view_in_explorer":     "View in explorer",
		"open_presets_exp":     "Open PRESETS Explorer",
		"filter_label":         "Filter: ",
		"manager_db_btn":       "🗄️ Downloaded presets manager",
		"manager_db_open":      "Open manager in:\n%s",
		"manager_db_change":    "Change path and open...",
		"manager_where_open":   "Where do you want to open the database manager?",
		"manager_options":      "Manager Options",
		"download_status":      "Downloading…",
		"dl_progress":          "Downloading %d/%d: %s…",
		"error_db":             "database error: %v",
		"error_html":           "tone unavailable (HTML)",
		"report_title":         "📊 Report:\n\n✅ Downloaded: %d\n⏭ Skipped: %d\n❌ Errors: %d",
		"report_detected":      "\nErrors detected:",
		"dl_error_title":       "Download finished with errors",
		"dl_retry":             "Retry failed",
		"dl_done_status":       "Download finished: %d files",
		"btn_dl_sel":           "✅ Download SELECTED",
		"no_sel_title":         "No selection",
		"no_sel_msg":           "Select at least one preset.",
		"btn_dl_all":           "⬇ Download ALL",
		"dl_all_msg":           "%d presets found.\nAlready downloaded: %d\nNew: %d\n\nWhat do you want to download?",
		"dl_title":             "Download",
		"dl_all_btn":           "Download ALL",
		"btn_help":             "Help",
		"error_manual":         "error extracting manual: %v",
		"label_search":         "🔍 Search:",
		"label_sort":           "Sort:",
		"label_from":           " From:",
		"label_to":             " To:",
		"label_choose_folder":  "Output folder:",
		"label_per_page":       "    Per page:",
		"results_found":        "%d presets found",
		"results_overall":      "✅ %d presets (%d total)",
		"dp_title_from":        "From",
		"dp_title_to":          "To",
		"dl_new_only_btn":      "New only",
		"completed_short":      "Download completed",
		"error_init_db":        "Error initializing DB on startup",
		"btn_donate":           "☕ Donate",
		"btn_theme_dark":       "🌙 Dark",
		"btn_theme_light":      "☀️ Light",
	},
}

var CurrentLang = "es"

func Get(key string) string {
	l := strings.ToLower(CurrentLang)
	if l != "en" {
		l = "es" // Default to Spanish
	}
	if v, ok := dict[l][key]; ok {
		return v
	}
	return key
}

func GetF(key string, args ...interface{}) string {
	return fmt.Sprintf(Get(key), args...)
}

func SetLang(l string) {
	if strings.ToLower(l) == "en" {
		CurrentLang = "en"
	} else {
		CurrentLang = "es"
	}
}

func GetMonths() []string {
	return strings.Split(Get("months"), ",")
}

func GetDays() []string {
	return strings.Split(Get("days"), ",")
}

// Helper block to inject language selector into UI
func NewLangSelector(w fyne.Window, cfgLang string, onLangChange func(string)) *fyne.Container {
	var btnES, btnEN *widget.Button

	updateImportance := func() {
		if CurrentLang == "en" {
			btnEN.Importance = widget.HighImportance
			btnES.Importance = widget.MediumImportance
		} else {
			btnES.Importance = widget.HighImportance
			btnEN.Importance = widget.MediumImportance
		}
		btnES.Refresh()
		btnEN.Refresh()
	}

	btnES = widget.NewButtonWithIcon("ES", FlagESResource, func() {
		if CurrentLang != "es" {
			SetLang("es")
			updateImportance()
			onLangChange("es")
		}
	})
	
	btnEN = widget.NewButtonWithIcon("EN", FlagENResource, func() {
		if CurrentLang != "en" {
			SetLang("en")
			updateImportance()
			onLangChange("en")
		}
	})

	updateImportance()

	return container.NewHBox(btnES, btnEN)
}
