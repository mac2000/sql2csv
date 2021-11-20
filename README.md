# SQL2CSV

sql2csv is a command line tool to export query results to csv file

You gonna need it only if you are going to export big amounts of data. Otherwise consider use PowerShell.

This tool is used in conjunction with `bq load` to import data into BigQuery, e.g.:

```bash
sql2csv -s localhost -u sa -p P@ssword -q "SELECT NotebookID, Name, Surname, convert(varchar, AddDate, 120), HeadquarterCityID AS CityID FROM NotebookEmployee with (nolock)" -o notebookemployee.csv -v
bq load --replace=true db.notebookemployee notebookemployee.csv "NotebookID:INTEGER,Name:STRING,Surname:STRING,AddDate:DATETIME,CityID:INTEGER"
```

> If you have long query, save it to a file and pass as `--input` parameter

## CSV

CSV tries to be as close as possible to RFC 4180

https://en.wikipedia.org/wiki/Comma-separated_values#RFC_4180_standard

- Encoding is UTF-8
- MS-DOS-style lines that end with (CR/LF) characters.
- Optional header.
- Each record contain the same number of comma-separated fields.
- All field quoted with doublequote.
- Double quotes escaped by doublequotes.
- Non printable characters removed.
- New lines replaced with space.

## Usage example

```bash
sql2csv -s localhost -u sa -p 123 -d blog -q "select id, name from posts" -o posts.csv

sql2csv -s localhost -u sa -p 123 -d blog -i posts.sql -o posts.csv

sql2csv -s localhost -u sa -p 123 -d blog -i posts.sql -o posts.csv --header -lf --verbose
```

## Required options

- `--query="select 1"` or `--input=query.sql` - provide query you wish to export
- `--output=data.csv` - file to save results to
- `--server=localhost` - servername to connect to
- `--database=RabotaUA2` - database to run query against
- `--username=sa` - username
- `--password=secret123` - password

## Optional

- `--verbose` - verbose output
- `--headers` - add headers line
- `--lf` - use `\n` instead of `\r\n`

## Build

```bash
go build
```

## Performance

**bcp**

```bash
bcp "SELECT ID, VacancyApplyID, QUOTENAME(convert(varchar, AddDate, 120), '\"'), QUOTENAME(FileName, '\"'), CheckSum, FileSize FROM VacancyApplyCVs with (nolock)" queryout "vacancyapplycvs.csv" -c -C65001 -t"," -r"\n"
2382128 rows copied.
Network packet size (bytes): 4096
Clock Time (ms.) Total     : 5600   Average : (425380.00 rows per sec.)
```

**sql2csv**

```bash
sql2csv.exe --query="SELECT ID, VacancyApplyID, convert(varchar, AddDate, 120), FileName, CheckSum, FileSize FROM VacancyApplyCVs with (nolock)" --output=vacancyapplycvs.csv
2->382->135 read in 00:00:06.9164592
2->382->135 process in 00:00:11.5524886
2->382->135 write in 00:00:11.5530416
Done in 00:00:11.5554066
```

Unfortunately we do not have managements object installed on server so can not compare with PowerShells Invoke-SqlCmd but bet it will be way to slower
