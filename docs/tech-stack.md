# Tech Stack Spec — Go + Templ + HTMX + Alpine.js

> Principes directeurs : KISS, fichier unique par responsabilité, zero over-engineering.
> Ce document est la référence pour vibecoder des outils internes légers (dashboards, small apps).

***

## Stack Summary

| Layer | Technology | Role |
|---|---|---|
| Backend | Go (stdlib + chi/echo) | HTTP server, routing, business logic |
| Templating | `templ` | HTML type-safe côté serveur |
| Interactivité | HTMX | Échanges HTML partiaux, no JS pour les calls serveur |
| State client léger | Alpine.js | Dropdowns, toggles, state local UI only |
| Style | Tailwind CSS (CDN Play) | Utilitaires CSS, pas de build step obligatoire |
| Charts | Chart.js (CDN) | Graphes via HTMX swap ou Alpine init |
| DB | SQLITE | No ORM, SQL natif |

***

## Structure de projet

```
my-tool/
├── main.go               # Entry point, wiring des routes
├── handlers/
│   ├── pages.go          # Handlers full-page (GET /)
│   └── partials.go       # Handlers HTMX partials (GET /partials/*)
├── templates/
│   ├── layout.templ      # Layout base (head, nav, body wrapper)
│   ├── pages/
│   │   ├── index.templ
│   │   └── dashboard.templ
│   └── partials/
│       ├── table.templ
│       └── chart.templ
├── static/               # Assets statiques si besoin (images, icons)
├── go.mod
└── go.sum
```

**Règles de structure :**

- Un handler = un fichier, groupés par responsabilité (pages vs partials)
- Les templates mirrorent la structure des handlers
- `main.go` ne contient que le wiring : server config, routes, inject deps
- Pas de dossier `pkg/`, `internal/`, `cmd/` pour des outils légers — flat is fine

***

## Conventions Go

### main.go — wiring only

```go
func main() {
    r := chi.NewRouter()
    r.Use(middleware.Logger)

    r.Get("/", handlers.IndexPage)
    r.Get("/partials/table", handlers.TablePartial)
    r.Post("/actions/delete/{id}", handlers.DeleteItem)

    log.Fatal(http.ListenAndServe(":8080", r))
}
```

### Handlers — simples et directs

```go
// handlers/pages.go
func IndexPage(w http.ResponseWriter, r *http.Request) {
    items := db.GetItems() // appel direct, pas d'abstraction inutile
    templates.Index(items).Render(r.Context(), w)
}

// handlers/partials.go — retourne du HTML partiel pour HTMX
func TablePartial(w http.ResponseWriter, r *http.Request) {
    items := db.GetItems()
    templates.Table(items).Render(r.Context(), w)
}
```

**Règles Go :**

- Pas de service layer pour les outils simples — handler → data → template, c'est tout
- Erreurs gérées inline, pas de middleware d'erreur complexe
- Pas de generics sauf si vraiment nécessaire
- Une fonction = une responsabilité, max ~40 lignes
- Nommage Go standard : `camelCase` privé, `PascalCase` exporté

***

## Conventions Templ

```templ
// templates/partials/table.templ
package templates

templ Table(items []Item) {
    <div id="table-wrapper">
        <table class="w-full text-sm">
            <thead>
                <tr class="border-b border-gray-200">
                    <th class="text-left py-2 px-4">Name</th>
                    <th class="text-left py-2 px-4">Status</th>
                </tr>
            </thead>
            <tbody>
                for _, item := range items {
                    @TableRow(item)
                }
            </tbody>
        </table>
    </div>
}

templ TableRow(item Item) {
    <tr class="border-b border-gray-100 hover:bg-gray-50">
        <td class="py-2 px-4">{ item.Name }</td>
        <td class="py-2 px-4">@StatusBadge(item.Status)</td>
    </tr>
}
```

**Règles Templ :**

- Découper en sous-composants (`TableRow`, `StatusBadge`) dès qu'un élément se répète
- Layout de base dans `layout.templ`, toutes les pages l'utilisent via `@Layout(title) { ... }`
- IDs HTML stables pour les swaps HTMX (ex: `id="table-wrapper"`)
- Pas de logique métier dans les templates — pré-calculer dans le handler

***

## Conventions HTMX

```html
<!-- Refresh un partial toutes les 5s -->
<div hx-get="/partials/stats"
     hx-trigger="load, every 5s"
     hx-target="#stats-container"
     hx-swap="innerHTML">
    <!-- contenu initial rendu côté serveur -->
</div>

<!-- Form submit sans reload -->
<form hx-post="/actions/create"
      hx-target="#table-wrapper"
      hx-swap="outerHTML">
    <input type="text" name="name" class="border rounded px-2 py-1" />
    <button type="submit" class="bg-blue-600 text-white px-4 py-1 rounded">
        Add
    </button>
</form>

<!-- Delete avec confirmation -->
<button hx-delete="/actions/delete/{ item.ID }"
        hx-target="closest tr"
        hx-swap="outerHTML swap:300ms"
        hx-confirm="Supprimer cet élément ?">
    Delete
</button>
```

**Règles HTMX :**

- `hx-target` pointe toujours vers un ID stable défini dans le template
- Utiliser `hx-swap="outerHTML"` pour remplacer un élément, `innerHTML` pour son contenu
- Les partials retournent **uniquement** le fragment HTML concerné, pas le layout complet
- Préférer `hx-boost` sur les liens de navigation simple (upgrade progressif)
- Pas de JS custom pour ce que HTMX peut faire nativement

***

## Conventions Alpine.js

Alpine uniquement pour le **state purement client** qu'HTMX ne gère pas :
dropdowns, modals, toggles, tabs, validation inline.

```html
<!-- Dropdown -->
<div x-data="{ open: false }" class="relative">
    <button @click="open = !open" class="border rounded px-3 py-1">
        Options ▾
    </button>
    <ul x-show="open"
        @click.outside="open = false"
        class="absolute bg-white border rounded shadow mt-1 z-10">
        <li class="px-4 py-2 hover:bg-gray-50 cursor-pointer">Export CSV</li>
        <li class="px-4 py-2 hover:bg-gray-50 cursor-pointer">Refresh</li>
    </ul>
</div>

<!-- Tab switcher local (pas de serveur nécessaire) -->
<div x-data="{ tab: 'overview' }">
    <div class="flex gap-2 border-b mb-4">
        <button :class="tab === 'overview' ? 'border-b-2 border-blue-600' : ''"
                @click="tab = 'overview'">Overview</button>
        <button :class="tab === 'details' ? 'border-b-2 border-blue-600' : ''"
                @click="tab = 'details'">Details</button>
    </div>
    <div x-show="tab === 'overview'">...</div>
    <div x-show="tab === 'details'">...</div>
</div>
```

**Règles Alpine :**

- `x-data` le plus proche de l'élément qui en a besoin (pas de global store sauf nécessité absolue)
- Pas de logique métier dans Alpine — uniquement `open/close`, `tab`, `loading`, `error`
- Si une action a besoin d'aller au serveur → HTMX, pas `fetch()` dans Alpine
- Garder les expressions inline courtes ; dès que c'est complexe, extraire dans `x-data="myComponent()"`

***

## Données & Persistence

Pour des outils légers, ne pas sur-ingéniérer la couche data :

| Besoin | Solution |
|---|---|
| Données simples, lecture | SQLite via `modernc.org/sqlite` (no CGO) |
| Config / état persisté | JSON file ou SQLite |
| Cache en mémoire | `sync.Map` ou simple `map` avec mutex |
| DB existante | `database/sql` + driver adapté |
| Pas de données persistantes | Struct en mémoire dans le handler |

```go
// db.go — accès direct, pas de repository pattern pour les outils simples
var db *sql.DB

func GetItems() []Item {
    rows, _ := db.Query("SELECT id, name, status FROM items ORDER BY created_at DESC")
    defer rows.Close()
    var items []Item
    for rows.Next() {
        var i Item
        rows.Scan(&i.ID, &i.Name, &i.Status)
        items = append(items, i)
    }
    return items
}
```

***

## Layout de base (Tailwind CDN)

```templ
// templates/layout.templ
package templates

templ Layout(title string) {
    <!DOCTYPE html>
    <html lang="fr">
    <head>
        <meta charset="UTF-8"/>
        <meta name="viewport" content="width=device-width, initial-scale=1.0"/>
        <title>{ title }</title>
        <!-- Tailwind CDN (Play CDN pour dev/outils internes) -->
        <script src="https://cdn.tailwindcss.com"></script>
        <!-- HTMX -->
        <script src="https://unpkg.com/htmx.org@2.0.4" defer></script>
        <!-- Alpine.js -->
        <script src="https://unpkg.com/alpinejs@3.x.x/dist/cdn.min.js" defer></script>
    </head>
    <body class="bg-gray-50 text-gray-900 antialiased">
        <nav class="bg-white border-b px-6 py-3 flex items-center gap-6">
            <span class="font-semibold text-gray-800">{ title }</span>
        </nav>
        <main class="max-w-6xl mx-auto px-6 py-8">
            { children... }
        </main>
    </body>
    </html>
}
```

***

## KISS — Ce qu'on ne fait PAS

- ❌ Pas de SPA routing — chaque page est une URL serveur
- ❌ Pas de Redux/Pinia/state management global — Alpine `x-data` local suffit
- ❌ Pas de GraphQL — REST simple ou queries directes
- ❌ Pas de Docker pour dev local — `go run .` suffit
- ❌ Pas de microservices — un binary, un process
- ❌ Pas de service layer / repository pattern si l'outil a < 10 entités
- ❌ Pas de build step frontend obligatoire — Tailwind CDN pour outils internes
- ❌ Pas de tests unitaires sur les templates — tester la logique Go, pas le HTML

***

## Dev workflow

```bash
# Install
go install github.com/a-h/templ/cmd/templ@latest

# Génération des templates (à relancer à chaque modif .templ)
templ generate

# Hot reload avec Air
go install github.com/air-verse/air@latest
air  # surveille .go et .templ, relance le serveur auto

# Build final — binary unique, pas de dépendances externes
go build -o ./bin/my-tool .
./bin/my-tool
```

**`.air.toml` minimal :**

```toml
[build]
  cmd = "templ generate && go build -o ./tmp/main ."
  bin = "./tmp/main"
  include_ext = ["go", "templ", "html"]
  exclude_dir = ["tmp", "static"]
```

***

## Checklist avant de vibecoder

- [ ] `main.go` contient seulement les routes et le server init
- [ ] Chaque handler retourne soit une full page, soit un partial HTMX
- [ ] Les IDs HTMX targets sont définis dans les templates, pas dans le HTML inline
- [ ] Alpine uniquement pour le state UI local (dropdowns, tabs, modals)
- [ ] Pas de `fetch()` JS custom si HTMX peut le faire
- [ ] SQLite ou struct en mémoire selon besoin, pas de ORM
- [ ] `templ generate` lancé avant `go run`
- [ ] `air` pour le hot reload en dev
