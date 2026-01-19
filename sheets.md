Documentation for the used "Google Sheets" format.

Tabs / Sheets:

* (required) Running events lists, split by year: "Events$YEAR", e.g. "Events2025", "Events2026", ...
* (required) Running groups list: "Groups"
* (required) Running shops list: "Shops"
* (required) Tag definitions: "Tags"
* (required) Running series definitions: "Series"
* (required if config.pages.parkrun=true) Parkrun data: "Parkrun"
* (optional) Ignored tabs/sheets: name contains "(ignored)"

Columns in Events / Groups / Shops:

* NAME
** "name" or "name|oldname"
** Name of the event, also used to derive the URL
** if "oldname" is given, a redirect from "url(oldname)" to "url(name)" is added - useful for typos/renames 
* NAME2
** "basename"
** Used to group similar events
** Events with the same "basename" are linked together.
** The current event of the group is avaibale as "url(basename)"   
* STATUS
** "status"
** If non-empty, the status is displayed in the event card
** if status == "obsolete", the event is completely hidden
** if status contains "abgesagt" or "geschlossen", the event is marked as cancelled
