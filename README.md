# CustomTone Helix Downloader (CTHD)

[English](#english) | [Español](#español)

---

## English

### Introduction
**CTHD** is a powerful tool designed to search and download presets from the Line 6 CustomTone repository automatically. Coupled with **CTHD Manager**, you can organize, filter, and audit your collection effortlessly.

### Key Features
- **Smart Search**: Find presets by name, style, or band/artist.
- **Bulk Download**: Download multiple presets simultaneously.
- **Integrated Manager**: Audit your downloaded presets, view them in Explorer, and organize them into folders.
- **Bilingual Interface**: Full support for English and Spanish.
- **Embedded Resources**: No external image files needed; everything is self-contained in the `.exe`.

### How to use
1. Run `CTHD.exe` (Windows) or open `CustomTone Helix Downloader.app` (macOS).
2. Configure your download folder (it will prompt you if not set).
3. Use the search bar to find presets.
4. Click "Download" to fetch them.
5. Use `CTHD_Manager.exe` (Windows) or `Helix Manager.app` (macOS) to manage your local collection.

### macOS Troubleshooting
If you see an error saying the app is **"damaged"** (because it is unsigned), run this command in your Terminal:
```bash
xattr -cr "CustomTone Helix Downloader.app"
xattr -cr "Helix Manager.app"
```
Then you can open them normally.

**App Translocation (Infinite Permissions)**: If macOS keeps asking for permission at every folder level, it's because the app is running in a temporary sandbox (App Translocation). To fix this, **move the `.app` bundle from your Downloads folder to the `/Applications` folder** before running it.

### Downloads
You can find the latest version for Windows and macOS in the [Releases](https://github.com/blackchorimaGZ/Customtone-Helix-Downloader/releases) section.

---

## Español

### Introducción
**CTHD** es una potente herramienta diseñada para buscar y descargar presets del repositorio CustomTone de Line 6 de forma automática. Junto con **CTHD Manager**, puedes organizar, filtrar y auditar tu colección sin esfuerzo.

### Características Principales
- **Búsqueda Inteligente**: Encuentra presets por nombre, estilo o banda/artista.
- **Descarga Masiva**: Descarga múltiples presets simultáneamente.
- **Mánager Integrado**: Audita tus presets descargados, míralos en el Explorador y organízalos en carpetas.
- **Interfaz Bilingüe**: Soporte completo para inglés y español.
- **Recursos Integrados**: No se necesitan archivos de imagen externos; todo está contenido en el `.exe`.

### Cómo usar
1. Ejecuta `CTHD.exe` (Windows) o abre `CustomTone Helix Downloader.app` (macOS).
2. Configura tu carpeta de descarga (se te preguntará si no está configurada).
3. Usa la barra de búsqueda para encontrar presets.
4. Haz clic en "Descargar" para obtenerlos.
5. Usa `CTHD_Manager.exe` (Windows) o `Helix Manager.app` (macOS) para gestionar tu colección local.

### Solución de problemas en macOS
Si ves un error diciendo que la aplicación está **"dañada"** (debido a que no está firmada), ejecuta este comando en tu Terminal:
```bash
xattr -cr "CustomTone Helix Downloader.app"
xattr -cr "Helix Manager.app"
```
Después podrás abrirlas normalmente.

**App Translocation (Permisos infinitos)**: Si macOS te pide permisos para cada subcarpeta una y otra vez, es porque la app está en "App Translocation". Para solucionarlo, **mueve el archivo `.app` desde Descargas a la carpeta `/Aplicaciones`** antes de abrirlo.

### Descargas
Puedes encontrar la última versión para Windows y macOS en la sección de [Releases](https://github.com/blackchorimaGZ/Customtone-Helix-Downloader/releases).

---

### License / Licencia
This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.
Este proyecto está bajo la Licencia MIT - ver el archivo [LICENSE](LICENSE) para más detalles.
