# freiburg-run
A static website listing local running events in the Freiburg region.

https://freiburg.run/


Workflow:

* I manually organize all data in a [Google Spreadsheet](https://docs.google.com/spreadsheets/d/1VqYCMrkaD-mEDYWRfXPB9lRzMfKmxkS93l1eOUMThkE)
** There are pages for running events (by year), running groups, shops, the [local parkrun](https://parkrun.com.de/dietenbach),  and special pages listing tags & running series
** I use a color coding scheme to highlight complete & missing information: green = event is done & there is already a version for the next year, yellow = event is incomplete, e.g. registration link is missing, I need to revisit it later. This allows me to quickly identify events that need to be updated.
** Using Google Spreadsheets allows me to completely ignore the "admin interface" part, because I can just use Google's apps to manage the data. 
* On my server (hosted at [Uberspace](https://uberspace.de)), a cronjob is running every 30min, that calls a custom Golang to
** download the spreadsheet,
** extract all data,
** produce static HTML pages using Golang's HTML templates, and
** deploy everything to https://freiburg.run

Of couse you can fork the repository to create a version for your city, but be warned: 
Everything is tailored towards this very special workflow, the code is not built for customizability (e.g. the freiburg.run domain name is hardcoded).
Feel free to contact me, if you have such plans... 