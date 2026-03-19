---
Acrylic — Feature Specification (v2)

Purpose: A desktop app for artists that manages painting projects. Drop an image,
extract its dominant colors, match them to real acrylic paints with mixing recipes,
track which paints you own, and generate shopping lists for what you need to buy.

Tech stack: Go backend (Wails v2), React 18 + TypeScript + Mantine 8 frontend,
SQLite database (modernc.org/sqlite), shared packages (packages/color for color
math, packages/appkit for app lifecycle).

---
Data Entities

Project
- id (int, primary key)
- name (text, user-editable, defaults to filename)
- image_path (text, relative to app data dir)
- thumbnail_path (text, 80x80 thumbnail)
- n_colors (int, 2–40, default 10)
- tile_size (int, 1/2/4/8/16, default 1)
- posterize (bool, default off)
- smoothing_passes (int, 0–5, default 0)
- aspect_ratio (text, original/16:9/9:16/1:1, default original)
- match_owned_only (bool, default off)
- notes (text)
- created_at, updated_at (timestamps)

Project Color (extracted dominant color, belongs to Project)
- id, project_id (FK, CASCADE delete)
- sort_order (dominant first by pixel count)
- r, g, b, hex, pixel_count

Color Match (paint match for a Project Color)
- id, color_id (FK, CASCADE delete)
- match_type ("single" or "recipe")
- rank (1st, 2nd, 3rd best)
- delta_e (float), match_rating (Perfect/Excellent/Good/Approximate)

Match Part (paint component of a Color Match)
- id, match_id (FK, CASCADE delete)
- paint_id (FK to paints table)
- parts (int, ratio portion)

Paint (seeded from embedded CSV on first launch)
- id (text, e.g. "GOLDEN-1120", primary key)
- brand, name, series (int), opacity, pigments
- r, g, b, hex
- lab_l, lab_a, lab_b (pre-computed for matching)
- owned (bool, default false — user toggles to track inventory)

Favorite (user-saved mixing recipe, not project-specific)
- id, name, notes
- r, g, b, hex (resulting color)
- created_at

Favorite Part (paint in a Favorite recipe)
- id, favorite_id (FK, CASCADE delete)
- paint_id (FK to paints table)
- parts (int, ratio portion)

All FK relationships use ON DELETE CASCADE so deleting a project or favorite
cleans up all child rows automatically.

---
Database Schema

CREATE TABLE paints (
    id          TEXT PRIMARY KEY,
    brand       TEXT NOT NULL,
    name        TEXT NOT NULL,
    series      INTEGER NOT NULL,
    opacity     TEXT NOT NULL,
    pigments    TEXT NOT NULL,
    r           INTEGER NOT NULL,
    g           INTEGER NOT NULL,
    b           INTEGER NOT NULL,
    hex         TEXT NOT NULL,
    lab_l       REAL NOT NULL,
    lab_a       REAL NOT NULL,
    lab_b       REAL NOT NULL,
    owned       INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE projects (
    id               INTEGER PRIMARY KEY AUTOINCREMENT,
    name             TEXT NOT NULL,
    image_path       TEXT NOT NULL,
    thumbnail_path   TEXT NOT NULL,
    n_colors         INTEGER NOT NULL DEFAULT 10,
    tile_size        INTEGER NOT NULL DEFAULT 1,
    posterize        INTEGER NOT NULL DEFAULT 0,
    smoothing_passes INTEGER NOT NULL DEFAULT 0,
    aspect_ratio     TEXT NOT NULL DEFAULT 'original',
    match_owned_only INTEGER NOT NULL DEFAULT 0,
    notes            TEXT NOT NULL DEFAULT '',
    created_at       TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at       TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE TABLE project_colors (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    project_id  INTEGER NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    sort_order  INTEGER NOT NULL,
    r           INTEGER NOT NULL,
    g           INTEGER NOT NULL,
    b           INTEGER NOT NULL,
    hex         TEXT NOT NULL,
    pixel_count INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE color_matches (
    id           INTEGER PRIMARY KEY AUTOINCREMENT,
    color_id     INTEGER NOT NULL REFERENCES project_colors(id) ON DELETE CASCADE,
    match_type   TEXT NOT NULL,
    rank         INTEGER NOT NULL,
    delta_e      REAL NOT NULL,
    match_rating TEXT NOT NULL
);

CREATE TABLE match_parts (
    id       INTEGER PRIMARY KEY AUTOINCREMENT,
    match_id INTEGER NOT NULL REFERENCES color_matches(id) ON DELETE CASCADE,
    paint_id TEXT NOT NULL REFERENCES paints(id),
    parts    INTEGER NOT NULL
);

CREATE TABLE favorites (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    name       TEXT NOT NULL,
    notes      TEXT NOT NULL DEFAULT '',
    r          INTEGER NOT NULL,
    g          INTEGER NOT NULL,
    b          INTEGER NOT NULL,
    hex        TEXT NOT NULL,
    created_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE TABLE favorite_parts (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    favorite_id INTEGER NOT NULL REFERENCES favorites(id) ON DELETE CASCADE,
    paint_id    TEXT NOT NULL REFERENCES paints(id),
    parts       INTEGER NOT NULL
);

---
Paint Inventory

Initial seed: 117 Golden Acrylics, embedded as CSV in packages/color.
On first launch, the app seeds the paints table from the CSV (using packages/color
to parse and compute Lab values). This is a one-time operation.

Future brands can be added as additional CSV files. The paint_id format uses a
brand prefix (e.g. "GOLDEN-1120", "LIQUITEX-1045") to avoid collisions.

The Paints list view doubles as an inventory manager: browse all paints, toggle
"owned" on/off. Filter by Owned / Need / All. PaintDetail shows ownership toggle,
full paint info, and "used in N projects" cross-reference.

---
Matching Modes (per-project toggle)

Best Match (match_owned_only = false):
- Match against ALL paints in the database
- Shopping list shows only paints in recipes that user does NOT own
- This is the default — gives ideal recipes

From My Collection (match_owned_only = true):
- Match only against paints where owned = 1
- Shopping list is empty (you have everything)
- Useful when you want to paint now with what you have

---
Project Lifecycle

Creating a Project:
1. User drops image (JPG, PNG, GIF) onto drop zone (or opens via file dialog)
2. "New Project" modal appears with:
   - Image thumbnail preview
   - Name field (pre-filled with filename sans extension)
   - nColors slider (default 10)
   - "Create" button
3. On "Create":
   - Copy image to ~/.local/share/trueblocks/acrylic/projects/<id>/original.<ext>
   - Generate 80x80 thumbnail at projects/<id>/thumb.<ext>
   - Insert project row in DB
   - Run processing pipeline (K-Means + matching)
   - Insert project_colors, color_matches, match_parts rows
   - Navigate to project workspace (detail view)

Reprocessing:
- When user changes nColors, tile_size, posterize, smoothing, aspect_ratio,
  or match_owned_only in the workspace, update the project row and reprocess
- Reprocessing discards old palette rows (CASCADE from project_colors) and
  regenerates from scratch
- Reprocessing is immediate on parameter change

Deleting a Project:
- Confirmation modal before proceeding
- DELETE project row (CASCADE cleans up all child rows)
- Delete project directory from filesystem (image + thumbnail)

---
Core Workflow

1. User creates a project (drop image → new project modal)
2. App extracts dominant colors via K-Means clustering (packages/color)
3. Each color is matched to paints using CIEDE2000 (packages/color)
4. User sees canvas + palette + shopping list in the project workspace
5. User adjusts parameters (nColors, posterize, etc.) — auto-reprocesses
6. User can toggle "owned" on paints to refine shopping list
7. User can save mixing recipes as favorites
8. User can export PDFs

---
Navigation & Screens

Nav items: Projects, Paints, Favorites

Projects Page (Page/List/Detail triad):
  ProjectsList — DataTable with thumbnail, name, color count, updated date
  ProjectDetail — the workspace (see below)

Paints Page (Page/List/Detail triad):
  PaintsList — DataTable with color swatch, name, brand, series, opacity,
    pigments, owned toggle. Filterable by brand, opacity, owned status.
  PaintDetail — large swatch, Lab values, ownership toggle, "used in N
    projects" cross-reference

Favorites Page (Page/List/Detail triad):
  FavoritesList — DataTable with color swatch, name, recipe summary
  FavoriteDetail — mixing bar, paint cards, notes, delete button

---
Project Workspace (ProjectDetail)

Canvas area:
- Displays original or posterized image
- Posterize toggle — flat color regions using palette
- Tile size — 1/2/4/8/16px blocks (paint-by-numbers effect)
- Smoothing passes — 0–5 (cleans boundaries between color regions)
- Aspect ratio — original / 16:9 / 9:16 / 1:1
- Click canvas to highlight pixels matching a color (red overlay)
- PDF export buttons (1–5) overlaid on canvas

Canvas controls:
- nColors slider (2–40)
- Posterize toggle
- Tile size selector
- Smoothing passes slider
- Aspect ratio selector
- Match mode toggle (Best Match / From My Collection)

Results panel (alongside canvas):
- Palette grid — color swatches with hex values; click to highlight in image
- Shopping list — each row shows:
  - Target color swatch
  - Expected mix swatch
  - Proportional mixing bar with paint names and ratios
  - Owned paints are greyed out with checkmark
  - Unowned paints are highlighted as "need to buy"
  - "Save as Favorite" action per recipe

Color Detail Modal (click a shopping list row):
- Zoomed image with highlighted pixels
- Target vs. result color comparison
- Mixing recipe bar (proportional)
- Paint cards: name, brand, series, pigments, opacity, hex, parts ratio
- Best single-paint alternative if closer than recipe
- "Save as Favorite" button
- Export PDF button for this color

All canvas rendering (posterize, tile, smoothing, highlight overlay) happens in
the frontend using HTML Canvas. The Go backend provides image file paths; the
frontend loads and renders them.

---
PDF Export (5 slots + color detail)

1. Comparison — Original + modified side-by-side, full shopping list,
   palette with isolation images, all mixing recipes
2. Paint-by-Numbers — Overview page, full-page numbered grid, shopping
   list, per-color pages with highlighted grid + recipe
3–5. Placeholder (not yet implemented)
Color Detail — From modal: zoomed highlight, target vs result, recipe
   bar, paint specs

PDF generation uses gofpdf on the Go backend. Export saves to a
user-chosen location via system file dialog.

---
Image Processing Parameters

nColors:      2–40, number of dominant colors extracted
Posterize:    on/off, map all pixels to nearest palette color
Tile size:    1/2/4/8/16, block pixels into NxN tiles
Smoothing:    0–5, mode filter + dilation to clean boundaries
Aspect ratio: original/16:9/9:16/1:1, crop ratio

---
Color Matching Engine (packages/color)

The acrylic app uses packages/color as a pure computation library. The app
loads paints from its SQLite database, passes them to packages/color functions,
and stores results back to the database. packages/color never knows about SQLite.

- CIEDE2000 (delta E) for perceptual accuracy
- Match quality ratings: Perfect (<1.0), Excellent (<2.0), Good (<5.0),
  Approximate (<10.0), No Match (>=10.0)
- Mixing: exhaustive search of 2-paint combos (ratios 1:9–9:1),
  then 3-paint if needed
- Ratio simplification (e.g. 4:6 → 2:3)
- Falls back to single best paint if it beats the recipe
- K-Means++ clustering with deterministic seed for repeatable results
- Image resized to max 256x256 for extraction performance

---
Keyboard Shortcuts

Cmd+1–5:              Export PDF (handlers 1–5)
Cmd+O:                Export PDF (context-dependent)
Cmd+N / Cmd+Shift+N:  Navigate views / cycle tabs (per CREATING_NEW_APPS.md)
Cmd+T / Cmd+Shift+T:  Cycle tile size forward / backward (in workspace)
Cmd+P:                Toggle posterize (in workspace)
Cmd+S / Cmd+Shift+S:  Increase / decrease smoothing (in workspace)
Cmd+A:                Cycle aspect ratio (in workspace)
Arrow keys:           Navigate colors in palette, prev/next in detail views
Enter:                Open color detail modal
Escape:               Close modal / unhighlight

---
State Persistence

App state saved via appkit.Store to
~/.local/share/trueblocks/acrylic/state.json:
- Window position and size
- Sidebar width
- Last route (restored on restart)
- Active tab per view (projects, paints, favorites)
- Full route per tab (so returning to a view restores the exact item)
- Table state per DataTable (sort, search, scroll)

Project-specific settings (nColors, tileSize, etc.) are stored in the
database, NOT in state.json.

Images stored at:
~/.local/share/trueblocks/acrylic/projects/<id>/original.<ext>
~/.local/share/trueblocks/acrylic/projects/<id>/thumb.<ext>

Database stored at:
~/.local/share/trueblocks/acrylic/acrylic.db

---
Go Backend Architecture

acrylic/
  main.go                     appkit.Run(), embed assets
  app/
    app.go                    App struct, Startup/Shutdown, DB init, paint seeding
    projects.go               Thin Wails bindings for project CRUD
    paints.go                 Thin Wails bindings for paint queries + owned toggle
    favorites.go              Thin Wails bindings for favorite CRUD
    processing.go             ProcessImage, ReprocessProject (orchestrates packages/color)
    pdf.go                    PDF export methods (gofpdf)
    state.go                  Thin Wails bindings for state.Manager
  internal/
    db/
      db.go                   DB struct, New(), schema init (embedded), paint seeding
      schema.sql              Embedded SQL schema (all tables above)
      projects.go             Project CRUD + project_colors + color_matches + match_parts
      paints.go               Paint queries, ownership toggle, cross-references
      favorites.go            Favorite CRUD + favorite_parts
    state/
      state.go                AppState + Manager (window, sidebar, routes, tabs, tables)

---
Frontend Architecture

frontend/src/
  main.tsx                    MantineProvider, Notifications, BrowserRouter
  App.tsx                     AppLayout, routes, hotkeys, state persistence
  utils/index.ts              Log, LogErr
  components/
    DataTable.tsx             createDataTable wrapper (table state persistence)
    NewProjectModal.tsx       Drop image → name → create project
    DeleteConfirmModal.tsx    Confirmation before deleting project
    Canvas.tsx                Image rendering, posterize, tile, highlight overlay
    CanvasControls.tsx        nColors, posterize, tile, smoothing, aspect, match mode
    PaletteGrid.tsx           Color swatches (click to highlight in canvas)
    ShoppingList.tsx          Target → recipe → proportional bars, owned indicators
    ColorDetailModal.tsx      Zoomed highlight, recipe, paint cards, save favorite
    MixingBar.tsx             Proportional paint ratio bar
    PdfButtons.tsx            5 export button overlays on canvas
  hooks/
    useCanvasRenderer.ts      Canvas drawing logic (posterize, tile, smooth, highlight)
    useKeyboardShortcuts.ts   All Cmd+X workspace hotkeys
  pages/
    ProjectsPage.tsx          TabView: list / detail (NavigationProvider)
    ProjectsList.tsx          DataTable of projects
    ProjectDetail.tsx         The workspace: Canvas + Controls + Results + PDF
    PaintsPage.tsx            TabView: list / detail (NavigationProvider)
    PaintsList.tsx            DataTable of paints with owned toggle
    PaintDetail.tsx           Paint info, ownership, cross-references
    FavoritesPage.tsx         TabView: list / detail (NavigationProvider)
    FavoritesList.tsx         DataTable of favorites
    FavoriteDetail.tsx        Mixing bar, paint cards, notes

---
Shared Packages

packages/color (Go):
- ExtractDominantColors(img) → []DominantColor
- FindPaintMatches(targetRGB, paints, topN) → []PaintMatch
- CalculateMixingRecipe(targetRGB, paints) → []MixingPart
- RGBToLab, LabToRGB, HexToRGB, RGBToHex
- DeltaE2000, DeltaE76
- LoadPaints (CSV parser, used for initial DB seeding)

packages/appkit (Go):
- Run(), AppConfig, Store[T], AppDirFor()
- GetFromMap, SetInMap, TableState, WindowState
- CopyFile, FileExists, FolderExists, LoadJSON[T]

@trueblocks/ui (TypeScript):
- AppLayout, TabView, DetailHeader, EditableField
- createDataTable, useWindowGeometry, NavItem

@trueblocks/scaffold (TypeScript):
- NavigationProvider, useNavigation
