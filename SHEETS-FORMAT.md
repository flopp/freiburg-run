## Google Sheets Format Documentation

### Tabs / Sheets

- **(required)** Running events lists, split by year: `Events$YEAR`, e.g. `Events2025`, `Events2026`, ...
- **(required)** Running groups list: `Groups`
- **(required)** Running shops list: `Shops`
- **(required)** Tag definitions: `Tags`
- **(required)** Running series definitions: `Series`
- **(required if config.pages.parkrun=true)** Parkrun data: `Parkrun`
- **(optional)** Ignored tabs/sheets: name contains `(ignored)`

---

### Events / Groups / Shops Sheets

| Column        | Description |
|---------------|-------------|
| NAME          | Event name, or `name|oldname`. Used for URLs. If `oldname` is given, a redirect is created from the old to the new name. |
| NAME2         | Basename for grouping similar events. Events with the same basename are linked. The current event is available as `url(basename)`. |
| DATE          | Date or date range of the event (required for Events, optional for Groups/Shops). |
| ADDED         | Date when the event was added to the sheet. |
| STATUS        | Status string. If non-empty, displayed in the event card. If `obsolete`, the event is hidden. If contains `abgesagt` or `geschlossen`, the event is marked as cancelled. If `temp`, the row is ignored. |
| URL           | Main website or info URL for the event (required). |
| DESCRIPTION   | Description text, or `desc1|desc2` for two-part descriptions. |
| LOCATION      | Location name or address. |
| COORDINATES   | Latitude,Longitude (optional, for map display). |
| REGISTRATION  | Registration URL (optional, used as a special link). |
| TAGS          | Comma-separated list of tags. Tags starting with `serie:` are used for series assignment. |
| LINK1, LINK2, ... | Additional links in the format `Label|URL`. Any number of LINK columns can be added. |

---

### Parkrun Sheet

| Column    | Description |
|-----------|-------------|
| DATE      | Date of the parkrun event. |
| INDEX     | Event index/number. |
| RUNNERS   | Number of runners. |
| TEMP      | Temperature (°C, will be suffixed with `°C`). |
| SPECIAL   | Special notes (optional). |
| CAFE      | Cafe information (optional). |
| RESULTS   | Results page suffix (will be expanded to full URL). |
| REPORT    | Report link or info. |
| AUTHOR    | Author of the report. |
| PHOTOS    | Photos link. |

---

### Tags Sheet

| Column      | Description |
|-------------|-------------|
| TAG         | Tag identifier (used for referencing in events). |
| NAME        | Human-readable tag name. |
| DESCRIPTION | Description of the tag. |

---

### Series Sheet

| Column      | Description |
|-------------|-------------|
| NAME        | Series name. |
| DESCRIPTION | Description of the series. |
| LINK1, LINK2, ... | Additional links in the format `Label|URL`. Any number of LINK columns can be added. |
