# SQL2CSV

sql2csv is a command line tool to export query results to csv file

You gonna need it only if you are going to export big amounts of data. Otherwise consider use PowerShell.

This tool is used in conjunction with `bq load` to import data into BigQuery, e.g.:

```bash
sql2csv --query="SELECT NotebookID, Name, Surname, convert(varchar, AddDate, 120), HeadquarterCityID AS CityID FROM NotebookEmployee with (nolock)" --output=notebookemployee.csv
bq load --replace=true db.notebookemployee notebookemployee.csv "NotebookID:INTEGER,Name:STRING,Surname:STRING,AddDate:DATETIME,CityID:INTEGER"
```

## CSV

- All fields are surrounded with double quotes
- Quotes inside fields are escaped following to RFC with two doublequotes
- Output file is UTF-8 without BOM
- Newlines are `\n`

## Usage example

```bash
sql2csv --query="SELECT * FROM NotebookEmployee with (nolock)" --output=notebookemployee.csv --password=secret123
```

## Required options

`--query="select 1"` or `--input=query.sql` - provide query you wish to export

`--output=data.csv` - file to save results to

## Optional

`--server=localhost` - servername to connect to
`--database=RabotaUA2` - database to run query against
`--username=sa` - username
`--password=secret123` - password

## Build

```bash
dotnet publish -c Release -p:PublishReadyToRun=true -p:PublishSingleFile=true -p:PublishTrimmed=true --self-contained true -r osx-x64

dotnet publish -c Release -p:PublishSingleFile=true -p:PublishTrimmed=true --self-contained true -r win-x64

dotnet publish -c Release -p:PublishSingleFile=true -p:PublishTrimmed=true --self-contained true -r linux-x64
```

Note: `PublishReadyToRun` is available only when you building project on a target platform

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
