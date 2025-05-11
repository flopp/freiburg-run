# freiburg-run
A static website listing local running events in the Freiburg region.

https://freiburg.run/

Data is pulled every hour from a [Google Spreadsheet](https://docs.google.com/spreadsheets/d/1VqYCMrkaD-mEDYWRfXPB9lRzMfKmxkS93l1eOUMThkE), a custom Golang program produces static HTML pages from that, and finally everything is deployed to https://freiburg.run
