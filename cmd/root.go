/*
Copyright © 2021 Alexandr Marchenko <marchenko.alexandr@gmail.com

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
*/
package cmd

import (
	"bufio"
	"bytes"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"reflect"
	"regexp"
	"strings"
	"time"
	"unicode"

	_ "github.com/denisenkom/go-mssqldb"
	"github.com/spf13/cobra"
)

var Bad = regexp.MustCompile(`[\x00\x08\x0B\x0C\x0E-\x1F]+`)
var Spaces = regexp.MustCompile(`\s+`)

var Verbose bool

var Server string
var Username string
var Password string
var Database string
var Port uint16

var Lf bool
var Delimiter string
var Headers bool
var Eol string

var Input string
var Query string
var Output string

var ConnectionString string
var Con *sql.DB
var Ctx context.Context
var Csv *bufio.Writer
var Start time.Time

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "sql2csv",
	Short: "Export SQL Server query results to CSV file",
	Long: `Export SQL Server query results to CSV file

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

If you have small query, you might pass with with "--query" parameter, if query is big save it to file and pass its path with "--input" parameter.

Usage examples:

sql2csv -s localhost -u sa -p 123 -d blog -q "select id, name from posts" -o posts.csv

sql2csv -s localhost -u sa -p 123 -d blog -i posts.sql -o posts.csv

sql2csv -s localhost -u sa -p 123 -d blog -i posts.sql -o posts.csv --header -lf --verbose
`,
	// Uncomment the following line if your bare application
	// has an action associated with it:
	PreRunE: func(cmd *cobra.Command, args []string) error {
		Start = time.Now()
		if Query != "" && Input != "" {
			return errors.New("only one \"query\" or \"input\" should be set")
		}

		if Query == "" && Input == "" {
			return errors.New("required flag(s) \"query\" or \"input\" not set")
		}

		if Input != "" {
			bytes, err := ioutil.ReadFile(Input)
			if err != nil {
				return err
			}
			Query = string(bytes)
		}

		if Lf {
			Eol = "\n"
		} else {
			Eol = "\r\n"
		}

		if Verbose {
			fmt.Println("Arguments:")
			fmt.Printf("  server:\t%v\n", Server)
			fmt.Printf("  port:\t\t%v\n", Port)
			fmt.Printf("  username:\t%v\n", Username)
			fmt.Printf("  password:\t%v\n", strings.Repeat("*", len(Password)))
			fmt.Printf("  database:\t%v\n", Database)
			fmt.Printf("  delimiter:\t%v\n", Delimiter)
			if Lf {
				fmt.Printf("  eol:\t\t%v\n", `\r\n`)
			} else {
				fmt.Printf("  eol:\t\t%v\n", `\n`)
			}
			fmt.Printf("  output:\t%v\n", Output)
			fmt.Printf("  query:\t%v\n", Query)
		}

		ConnectionString = fmt.Sprintf("server=%s;user id=%s;password=%s;port=%d;database=%s;encrypt=disable;ApplicationIntent=ReadOnly;app name=sql2csv", Server, Username, Password, Port, Database)
		con, err := sql.Open("sqlserver", ConnectionString)
		if err != nil {
			return err
		}
		Con = con
		Ctx = context.Background()
		if Verbose {
			fmt.Println("Created database coonection pool")
		}

		err = con.PingContext(Ctx)
		if err != nil {
			return err
		}
		if Verbose {
			fmt.Printf("Connected to database \"%s@%s:%d/%s\"\n", Username, Server, Port, Database)
		}

		f, err := os.OpenFile(Output, os.O_TRUNC|os.O_WRONLY|os.O_CREATE, 0600)
		if err != nil {
			return err
		}
		Csv = bufio.NewWriterSize(f, 10*1024*1024)
		if Verbose {
			fmt.Printf("Opened \"%v\" file\n", Output)
		}

		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		rows, err := Con.QueryContext(Ctx, Query)
		if err != nil {
			log.Fatal("Error executing query: ", err.Error())
		}
		defer rows.Close()
		if Verbose {
			fmt.Printf("Query executed\n")
		}

		types, err := rows.ColumnTypes()
		cells := make([]interface{}, len(types))
		isString := make([]bool, len(types))
		if err != nil {
			log.Fatal("Error retrieving column types: ", err.Error())
		}
		if Verbose {
			fmt.Printf("Retrieved %d column definitions:\n", len(types))
		}
		for i, t := range types {
			cells[i] = new([]byte)
			isString[i] = t.ScanType().Kind() == reflect.String
			name := t.Name()
			if name == "" {
				name = fmt.Sprintf("column_%v", i+1)
			}
			if Headers {
				if i != 0 {
					if _, err = Csv.WriteString(Delimiter); err != nil {
						log.Fatal("Error writing delimiter to file: ", err.Error())
					}
				}
				if _, err = Csv.WriteString(fmt.Sprintf("\"%s\"", name)); err != nil {
					log.Fatal("Error writing column name to file: ", err.Error())
				}
			}
			if t.DatabaseTypeName() == "BINARY" {
				log.Fatal(fmt.Sprintf("Error \"%v\" has binary type which is not supported", name))
			}
			if Verbose {
				fmt.Printf("  %-10v\t%-10v\t%v\n", strings.ToLower(t.DatabaseTypeName()), t.ScanType().Kind(), name)
			}
		}
		if Headers {
			if _, err = Csv.WriteString(Eol); err != nil {
				log.Fatal("Error writing eol symbol to file: ", err.Error())
			}
		}

		if Verbose {
			fmt.Println("Start reading rows")
		}
		counter := 0
		for rows.Next() {
			if err = rows.Scan(cells...); err != nil {
				log.Fatal(err.Error())
			}
			for i, cell := range cells {
				if i != 0 {
					if _, err = Csv.WriteString(Delimiter); err != nil {
						log.Fatal("Error writing delimiter to file: ", err.Error())
					}
				}
				if bytes, ok := cell.(*[]byte); ok {
					// Quote all cellss
					if _, err = Csv.WriteString(fmt.Sprintf("\"%s\"", san(*bytes))); err != nil {
						log.Fatal("Error writing string to file: ", err.Error())
					}
					// Quote only string cells
					// if isString[i] {
					// 	if _, err = f.WriteString(fmt.Sprintf("\"%s\"", san(*bytes))); err != nil {
					// 		log.Fatal("Error writing string to file: ", err.Error())
					// 	}
					// } else {
					// 	if _, err = f.Write(*bytes); err != nil {
					// 		log.Fatal("Error writing bytes to file: ", err.Error())
					// 	}
					// }
				}
			}
			if _, err = Csv.WriteString(Eol); err != nil {
				log.Fatal("Error writing eol symbol to file: ", err.Error())
			}
			counter++
			if fileInfo, _ := os.Stdout.Stat(); (fileInfo.Mode() & os.ModeCharDevice) != 0 {
				if Verbose && counter%50 == 0 {
					fmt.Printf("Read %v rows          \r", counter)
				}
			}

		}
		fmt.Printf("Done reading %v rows in %v\n", counter, time.Since(Start))
	},
	PostRun: func(cmd *cobra.Command, args []string) {
		err := Con.Close()
		if err != nil {
			log.Println(err.Error())
		}
		err = Csv.Flush()
		if err != nil {
			log.Println(err.Error())
		}
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	cobra.CheckErr(rootCmd.Execute())
}

func init() {
	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.

	// rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.sql2csv.yaml)")

	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	// rootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")

	rootCmd.Flags().BoolVarP(&Verbose, "verbose", "v", false, "Verbose output")

	rootCmd.Flags().StringVarP(&Server, "server", "s", "", "database server IP or FQDN")
	rootCmd.Flags().StringVarP(&Username, "username", "u", "", "database username")
	rootCmd.Flags().StringVarP(&Password, "password", "p", "", "database password")
	rootCmd.Flags().StringVarP(&Database, "database", "d", "", "database name")
	rootCmd.Flags().Uint16Var(&Port, "port", 1433, "database port")

	rootCmd.Flags().BoolVar(&Lf, "lf", false, "use `\\n` instead `\\r\\n` which is expected by RFC 4180")
	rootCmd.Flags().StringVar(&Delimiter, "delimiter", ",", "cells delimiter")
	rootCmd.Flags().BoolVar(&Headers, "headers", false, "print headers line")

	rootCmd.Flags().StringVarP(&Input, "input", "i", "", "path to file with sql to run, required if query not provided")
	rootCmd.Flags().StringVarP(&Query, "query", "q", "", "sql query to run, required if input file not provided")
	rootCmd.Flags().StringVarP(&Output, "output", "o", "", "path to csv file where results will be written")

	rootCmd.MarkFlagRequired("server")
	rootCmd.MarkFlagRequired("username")
	rootCmd.MarkFlagRequired("password")
	rootCmd.MarkFlagRequired("database")
	rootCmd.MarkFlagRequired("output")
}

func san(input []byte) []byte {
	return bytes.Trim(bytes.ReplaceAll(
		bytes.ReplaceAll(
			bytes.Map(func(r rune) rune {
				if unicode.IsGraphic(r) {
					return r
				}
				if unicode.IsSpace(r) {
					return rune(' ')
				}
				return rune(' ')
			}, bytes.ToValidUTF8(input, nil)),
			[]byte("\u00A0"), []byte(" ")),
		[]byte("\""), []byte("\"\"")), " ")
}

// func sanitize(input string) string {
// 	return strings.Replace(Spaces.ReplaceAllString(Bad.ReplaceAllString(input, ""), " "), "\"", "\"\"", -1)
// }
