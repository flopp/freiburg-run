# freiburg.run - Feature Overview

This document lists implemented features of the freiburg.run project (website + generator + operations tooling), based on the current codebase.

## 1. Product Scope

- Static website for the regional running ecosystem around Freiburg (~50 km radius).
- Main content types:
	- Laufveranstaltungen (events)
	- Lauftreffs (running groups)
	- Lauf-Shops (shops)
	- Tags/Kategorien
	- Serien
	- Optional: Dietenbach parkrun page
	- Optional: Community Run page

## 2. Main Website Features

### 2.1 Navigation and Information Architecture

- Fixed top navigation with desktop + mobile behavior.
- Dropdown navigation for event sub-areas (Kategorien, Serien, Archiv, Karte).
- Separate pages for events, groups, shops, info, legal pages, sitemap.
- Config-gated pages/menu entries:
	- `pages.club`
	- `pages.parkrun`
	- `pages.support`
	- `pages.watchlist`
- Breadcrumb navigation on non-map pages.
- Footer with:
	- optional custom footer links,
	- data source link to Google Sheets,
	- last update timestamp.

### 2.2 Event List Pages

- Card-based list rendering for:
	- current events,
	- past events archive,
	- groups,
	- shops,
	- tag pages,
	- series pages.
- Month separator cards to structure long lists.
- Archive split by year with year-switch buttons.
- Event card data shown:
	- name,
	- date,
	- location,
	- status,
	- primary URL,
	- details,
	- registration + extra links,
	- series,
	- tags.
- Distinct visual handling for cancelled events.
- Country flag indicators for FR/CH locations.
- Contextual location distance/direction from configured city center.

### 2.3 Filtering and Category Hiding

- Client-side filter input (name/location search).
- Numeric distance filter support in list search (e.g. `10`, `10.5`, `10,5`) with +/-10% tolerance.
- Filter summary text with shown/hidden counters.
- Per-tag hide controls on the categories page.
- Persisted hidden tags via Local Storage (`hiddenTags`).

### 2.4 Maps

- Full map page showing all geocoded entities:
	- current events,
	- old events,
	- groups,
	- shops.
- Map legend and marker color coding by item type.
- Radius circles around configured city center (25 km and 50 km).
- Fit-to-markers behavior.
- Embedded/toggleable map on list pages.
- Event detail map (single marker + optional route polyline).
- Parkrun map with course polyline + special POI markers.
- Gesture handling support for better mobile map usability.

### 2.5 Event Detail Page Features

- Dedicated detail page per entity (event/group/shop).
- Action buttons:
	- Share (Web Share API when supported)
	- Calendar (for upcoming events)
	- Fehler melden (report form)
	- Watchlist toggle (if enabled)
- Data table with key details and all outbound links.
- Historical context:
	- sibling editions of same base event,
	- previous/next edition links.
- Nearby upcoming events recommendation block (`Meta.UpcomingNear`, radius-based).

### 2.6 Calendar Features

- Per-event calendar integration:
	- Google Calendar deep link,
	- downloadable `.ics` data URL.
- Global `events.ics` feed for upcoming events.
- Calendar modal for user choice and explanation.
- All-day event handling using date ranges (ICS `DTSTART/DTEND` with end + 1 day).

### 2.7 Watchlist (Merkliste)

- Config-switchable (`pages.watchlist`), disabled by default in example config.
- Available in navbar (desktop + mobile), event cards, and detail pages.
- Browser-local persistence via Local Storage (`watchlist.v1`).
- Watchlist item schema includes id, slug/url, category, date fields, location, metadata.
- Stable identity uses `WatchlistID` based on `SlugNoBase`.
- Auto-prune of past items using `timeTo < today` (date-based).
- Sorting by earliest date first (fallback by name for undated entries).
- Deduplication by id with best-metadata retention.
- Hard cap of 100 items.
- Grouped display by category (Veranstaltungen/Lauftreffs/Lauf-Shops) in modal.
- Count badges in navbar.
- Local Storage unavailability fallback:
	- controls disabled,
	- warning shown in modal.
- Watchlist analytics events:
	- `watchlist-open`
	- `watchlist-add`
	- `watchlist-remove`
	- `watchlist-open-item`

### 2.8 Tags and Series

- Central tags page with:
	- link to each tag page,
	- current/archive counters,
	- hide toggle checkbox per tag.
- Tag detail page includes current events plus matching groups/shops.
- Tag archive page for old events.
- Cross-link between current and archive views.
- Series index page with current + old series tables.
- Series detail page with description, custom links, and associated events/groups/shops.

### 2.9 Parkrun Page

- Optional dedicated Dietenbach parkrun page.
- Parkrun-specific table with:
	- event index,
	- date,
	- runners,
	- temperature,
	- cafe,
	- special notes,
	- links (results/report/photos).
- Current-week highlighting.
- Responsive mobile/desktop table formatting.
- Additional static links and explanatory content.
- Separate WordPress-compatible export template for parkrun content.

### 2.10 Community/Support/Info/Legal

- Optional `community-run` page with schedule, route details, map, and terms.
- Optional support page with sponsorship/support options.
- Info page with mission, contact options, regional links, changelog, and technical notes.
- Impressum and Datenschutzerklaerung pages.
- 404 page and sitemap page.

### 2.11 Notifications

- Configurable top notification system.
- Time-window model (`start`/`end`) with active filtering.
- One-time display behavior per message id via Local Storage.
- Embedded-list guard (notifications suppressed in embed contexts).

### 2.12 Embeds and Sharing

- Generated embed pages for trail-related tags, split by country:
	- DE (`embed/trailrun-de.html`)
	- FR (`embed/trailrun-fr.html`)
	- CH (`embed/trailrun-ch.html`)
- Embed list includes event cards + attribution box.
- Share tracking and outbound link tracking via Umami events/attributes.

### 2.13 SEO, Discoverability, and Metadata

- Canonical URLs and OpenGraph/Twitter metadata.
- Dynamic page descriptions and titles.
- XML sitemap generation + human-readable sitemap page.
- `robots.txt` generation with sitemap reference.
- `llms.txt` generation with key page links and technical endpoints.
- `manifest.json` generation for app-like metadata.
- IndexNow key-file generation (optional).

### 2.14 Redirect and URL Compatibility Features

- Generated `.htaccess` includes:
	- canonical host redirect (www -> non-www),
	- historical/manual redirects,
	- redirects for old names (`NAME|oldname`),
	- redirects for slug/base-name transitions,
	- redirects for obsolete entities.

## 3. Data and Domain Logic Features

### 3.1 Data Source and Validation

- Google Sheets as authoritative source.
- Required sheets:
	- `EventsYYYY` (multiple, consecutive years)
	- `Groups`
	- `Shops`
	- `Tags`
	- `Series`
	- optional `Parkrun` (if enabled)
- Validation and normalization include:
	- date ordering checks,
	- name ordering checks,
	- event sheet naming/year validation,
	- required-column checks,
	- sheet discovery and unknown-sheet warnings.

### 3.2 Event Processing

- Split into current/old/obsolete lists.
- Automatic old/new detection from event date.
- Previous/next and sibling relation discovery.
- Nearby upcoming events discovery based on geodistance.
- Distance extraction/detection from text.
- Automatic country/location tags from FR/CH markers.
- Cancellation and obsolete handling from status semantics.
- Registration link relabeling for known result portals.

### 3.3 Tag and Series Aggregation

- Aggregates associations across all entity types.
- Automatic archive split for tags and series.
- Sorting and month separators for associated event lists.

### 3.4 Link Checking

- Optional link checker mode (`-checklinks`).
- Checks main + external links.
- Per-domain grouping with request concurrency limiting.

## 4. Generator and Build Features

- Custom Go static site generator.
- HTML templates with shared partials and helper functions.
- Build-time minification of generated HTML.
- Content-hashed asset output for cache busting (`*-HASH.*`).
- Static + vendor asset copying pipeline.
- Runtime base path handling for local file outputs and production URLs.
- Per-page canonical metadata and structured breadcrumbs.

## 5. Asset and Frontend Infrastructure

- Bulma-based design with custom CSS.
- Leaflet + Leaflet.Legend + Leaflet.GestureHandling integration.
- Gesture handling German text patch during asset preparation.
- Umami analytics script vendoring and optional activation via config.
- JS infrastructure for:
	- filters,
	- maps,
	- modals,
	- navbar burger,
	- dropdowns,
	- share,
	- watchlist,
	- notifications,
	- analytics hooks.

## 6. CLI, Operations, and Deployment Features

### 6.1 `cmd/generate`

- Configurable flags:
	- `-config`
	- `-out`
	- `-hashfile`
	- `-checklinks`
	- `-backup`
	- `-basepath`
- Retry-based data fetch for transient API issues.
- Backup mode exports Google Sheet as ODS.

### 6.2 `cmd/vendor-update`

- Downloads/updates external frontend dependencies:
	- Bulma
	- Leaflet
	- Leaflet.GestureHandling
	- Leaflet.Legend
	- Umami script
- Version pins documented in code (Renovate-aware comments).

### 6.3 Makefile Workflows

- Local build and run targets.
- Link-check target.
- Backup target.
- Vendor update target.
- Lint/test/full-test targets.
- Remote sync + remote execution targets for server deployment workflow.

### 6.4 Cron/Server Script

- Production script generates output and copies to web root.
- Uses strict bash settings (`set -euo pipefail`).

## 7. Configuration-Driven Features

- Central JSON config controls:
	- website identity + domain,
	- city center coordinates,
	- optional pages,
	- contact/social links,
	- footer links,
	- Google API/sheet ids,
	- analytics id,
	- IndexNow key,
	- notification messages.

## 8. Non-Functional Characteristics

- Static output (no application server runtime needed for the website).
- Mobile-focused behavior and map interaction safeguards.
- Privacy-oriented user state storage in Local Storage only for client-side preferences/features.
- SEO and crawler-friendly output (sitemap, robots, canonical, metadata).
- Architecture designed for periodic unattended regeneration/deployment.
