Feature: Personal Watchlist

Requirements:

- data storage on device only (no server-side storage), e.g. using "local storage" API
- Events can be added removed from watchlist from event cards and from individual event sub-pages
- Menu item to show a "watchlist dialog/popup"
- "watchlist dialog/popup" should show all items in watchlist 
  - click on item in watchlist should show the event sub-page
  - each item should have a button to remove it from the watchlist
- Watchlist feature should be toggleable from the config
- German phrasing: "Merkliste", "Zur Merkliste hinzufügen", ... 
- Watchlist is disabled by default (opt-in via config)
- Maximum 100 items in watchlist
- Auto-prune past events from watchlist
- Definition of "past event": event end date < today (timezone ignored)
- Ordering in watchlist: by event date, earliest date first
- Unique watchlist key must use SlugNoBase, not Slug

Potentially Missing Requirements:

- Unique key definition: watchlist identity should be based on SlugNoBase (slug without current-series aliasing), not Slug.
- Persistence behavior: watchlist should survive page reloads and browser restarts, but is device/browser-local only.
- Disabled localStorage fallback: if storage is unavailable (privacy mode, blocked storage), watchlist controls are visible but disabled with a short hint.
- Deleted/changed events: if a stored slug no longer exists on the current page, it should still appear in popup as a link to the slug and can be removed.
- Capacity and safety: hard limit of 100 items to avoid unbounded storage growth and avoid large DOM rendering costs.
- Accessibility: all watchlist actions must be keyboard reachable, include clear button labels, and expose pressed/toggled state via aria attributes.
- Tracking: define Umami events (for example watchlist-add, watchlist-remove, watchlist-open) for adoption measurement.
- Config contract: config switch name and default value should be explicitly defined (pages.watchlist default false).
- UX on empty state: popup should show a useful empty message and direct link back to event list.
- Ordering in popup: deterministic order by event date ascending (earliest date first).
- Pruning strategy: remove past events automatically from stored watchlist data.
- Pruning rule detail: an event is pruned when end date < today, comparing date values only (ignore timezones).

Implementation Plan:

1. Extend Config Model

- Add a boolean switch to the config struct in internal/utils/config.go under Pages, for example Watchlist bool with JSON key watchlist.
- Set explicit values in config.json, local.json, and example.json so behavior is deterministic in all environments.
- Keep backward compatibility by treating missing key as false. The feature is opt-in and disabled by default.

2. Add Data Hook for Templates

- Ensure all pages that render the navbar and watchlist controls already pass Config into templates (they do via helper function Config in template funcs).
- Expose SlugNoBase in watchlist data attributes in event-card and event-detail templates, because Slug can change for "current" series events.
- Keep regular navigation URLs based on Slug/SlugFile, but store watchlist identity via SlugNoBase.

3. Add Watchlist Entry Point in Navigation

- In templates/parts/header.html, add a new navbar item/button "Merkliste" guarded by Config.Pages.Watchlist.
- Make it a modal trigger (same pattern as existing modal code in static/main.js) with data-target pointing to new modal id.
- Add an icon only if consistent with current icon style; text label should remain visible for clarity.

4. Add Watchlist Modal Markup

- Create modal HTML in templates/parts/watchlist-modal.html and include it in templates/parts/tail.html (similar to calendar-modal include).
- Modal structure:
  - title: "Merkliste"
  - list container for entries
  - empty state text
  - optional small hint about local browser storage
- Add close controls compatible with existing modal close logic (.modal-background, .modal-close, etc.).

5. Add Add/Remove Controls on Event Cards

- In templates/parts/card.html, add a watchlist toggle button on each card, guarded by Config.Pages.Watchlist.
- Use data attributes required by JS, for example:
  - data-watchlist-toggle
  - data-watchlist-id (from SlugNoBase)
  - data-slug (navigation URL)
  - data-name
  - data-time
  - data-time-from
  - data-location
- Initial label should be "Zur Merkliste hinzufügen"; JS updates label/state after hydration.

6. Add Add/Remove Controls on Event Detail Pages

- In templates/event.html, add matching toggle button near existing action buttons (Teilen, Kalender, Fehler melden), guarded by Config.Pages.Watchlist.
- Reuse the same data attribute contract as cards so one JS handler can manage both contexts.

7. Implement Watchlist State Management in JavaScript

- In static/main.js, add watchlist module functions:
  - getWatchlist(): parse JSON from localStorage safely, return [] on failure
  - saveWatchlist(items): serialize safely, handle quota/storage exceptions
  - hasWatchlistItem(id)
  - addWatchlistItem(item)
  - removeWatchlistItem(id)
  - prunePastWatchlistItems(items, now)
  - sortWatchlist(items)
  - normalizeWatchlistItem(item): keep only required fields
- Storage key recommendation: "watchlist.v1" for future schema migration.
- Item schema recommendation:
  - id (required, value from SlugNoBase)
  - slug (required, navigation URL)
  - name
  - time
  - timeFrom (sortable date string, optional)
  - location
  - addedAt (timestamp for ordering)
- Enforce hard cap of 100 items after each add operation.
- Sort items by timeFrom ascending (earliest date first). For items without date, place them after dated items and sort by name.
- Auto-prune past events during initialization and before every render.
- Use end-date-based pruning: prune item when timeTo < today (date-only comparison, timezone-agnostic).

8. Wire UI Rendering and Synchronization

- Add initWatchlist() call in main() in static/main.js.
- On startup:
  - detect storage availability (reuse getLocalStorage helper)
  - set toggle button states/labels/aria-pressed
  - render modal list from storage
- On toggle click:
  - add/remove item
  - update all matching toggles on page (card + detail variants)
  - rerender modal content
- Modal list item behavior:
  - title link opens event URL (base path aware)
  - remove button deletes item and rerenders
  - list is always rendered in earliest-date-first order

9. Handle Edge Cases and Error UX

- If storage is unavailable:
  - disable watchlist buttons
  - show short hint text in modal like "Merkliste ist in diesem Browsermodus nicht verfügbar."
- If JSON in storage is corrupt:
  - fallback to empty list
  - optionally overwrite with clean empty list once user interacts
- If duplicates are added:
  - de-duplicate by id (SlugNoBase) and keep the most complete metadata.
- If an item is in the past:
  - remove it automatically from storage and UI in pruning pass.

10. Add Styling

- In static/style.css, add minimal classes for:
  - watchlist toggle active/inactive states
  - compact modal list layout
  - empty state styling
- Keep Bulma-first styling to avoid regressions and preserve existing visual language.

11. Add Analytics Events

- Use existing umami_track_event helper in static/main.js.
- Emit events:
  - watchlist-open
  - watchlist-add
  - watchlist-remove
  - optional watchlist-open-item

12. QA and Regression Checks

- Manual checks:
  - desktop + mobile navbar behavior
  - add/remove from card and detail page
  - state persists after reload
  - modal empty/non-empty states
  - config toggle hides all watchlist UI when disabled
  - identity stays stable when an event switches between Slug and "current series" base URL
  - auto-pruning removes past events
  - ordering is earliest date first
  - item cap is enforced at 100
- Accessibility checks:
  - keyboard activation (Enter/Space) on controls
  - focus stays usable when modal opens/closes
  - labels and aria-pressed states update correctly
- Performance checks:
  - rendering speed with 100 items
  - no repeated event listener leaks on rerender

13. Optional Enhancements (post-MVP)

- "Nur Merkliste" quick filter on events list page.
- Import/export watchlist JSON for browser migration.
- "Neu seit letztem Besuch" based on watchlist items and timestamp comparisons.

14. Manual QA Click-Path Checklist

- Preconditions:
  - Build output is fresh (run build once before testing).
  - Test with watchlist enabled and disabled in config.
  - Test once with normal browser storage and once with storage disabled/private mode.

- Desktop happy path:
  - Open home/events list page.
  - Click "Merkliste" in navbar and confirm empty-state text is shown.
  - On any event card, click "Zur Merkliste hinzufugen".
  - Confirm button state changes to "Von Merkliste entfernen".
  - Re-open modal and confirm item appears with title and metadata.
  - Click item link and confirm event detail page opens.
  - On detail page, confirm watchlist button is already in selected state.
  - Click remove on detail page and confirm modal/list updates accordingly.

- Mobile navbar + modal behavior:
  - Open page in mobile viewport.
  - Open burger menu, then open "Merkliste".
  - Confirm modal opens and closes via close button, backdrop click, and Escape (if keyboard available).
  - Confirm no stuck navbar/modal states after close.

- Identity stability (Slug vs SlugNoBase):
  - Add an event from card/list.
  - Open same logical event via a sibling/current URL variant.
  - Confirm toggle is still selected (same watchlist item, no duplicate).

- Ordering and pruning:
  - Add multiple events with different dates.
  - Confirm modal order is earliest date first.
  - Include one past event (timeTo < today) in localStorage test data.
  - Reload page and confirm past event is auto-pruned.

- Capacity limit:
  - Populate watchlist with >100 entries via test data.
  - Reload page.
  - Confirm list is capped at 100 and page remains responsive.

- Disabled config behavior:
  - Set pages.watchlist to false and rebuild.
  - Confirm navbar item, card buttons, detail button, and modal are absent.

- Storage unavailable behavior:
  - Block localStorage / use restrictive private mode.
  - Confirm watchlist buttons are disabled.
  - Confirm warning text appears in modal.
  - Confirm no uncaught console errors during open/add/remove attempts.

- Accessibility checks:
  - Tab to watchlist toggle button and activate with Enter/Space.
  - Confirm aria-pressed updates with state changes.
  - Confirm modal controls are keyboard reachable and close action restores usable focus flow.

- Analytics smoke checks (if Umami enabled):
  - Trigger open/add/remove/open-item actions.
  - Confirm corresponding events are emitted: watchlist-open, watchlist-add, watchlist-remove, watchlist-open-item.