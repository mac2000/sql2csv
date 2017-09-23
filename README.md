# SQL2CSV

sql2csv is a command line tool to export query results to csv file

You gonna need it only if you are going to export big amounts of data. Otherwise consider use PowerShell.

This tool is used in conjunction with `bq load` to import data into BigQuery, e.g.:

```
sql2csv.exe --query="SELECT NotebookID, Name, Surname, convert(varchar, AddDate, 120), HeadquarterCityID AS CityID FROM NotebookEmployee with (nolock)" --output=notebookemployee.csv
bq load --replace=true db.notebookemployee notebookemployee.csv "NotebookID:INTEGER,Name:STRING,Surname:STRING,AddDate:DATETIME,CityID:INTEGER"
```

## CSV

- All fields are surrounded with double quotes
- Quotes inside fields are escaped following to RFC with two doublequotes
- Output file is UTF-8 without BOM
- Newlines are `\n`

## Usage example:

```
sql2csv.exe --query="SELECT * FROM NotebookEmployee with (nolock)" --output=notebookemployee.csv --password=secret123
```

## Required options

`--query="select 1"` or `--input=query.sql` - provide query you wish to export

`--output=data.csv` - file to save results to

## Optional

`--server=localhost` - servername to connect to
`--database=RabotaUA2` - database to run query against
`--username=sa` - username
`--password=secret123` - password

# Build

Just clone repository, open solution in visual studio and build it