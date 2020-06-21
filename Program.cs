using System;
using System.Collections.Concurrent;
using System.Diagnostics;
using System.IO;
using System.Linq;
using System.Reflection;
using System.Text;
using System.Text.RegularExpressions;
using System.Threading;
using System.Threading.Tasks;
using Microsoft.Data.SqlClient;

namespace sql2csv
{
    public static class Program
    {
        private static readonly ParallelOptions ParallelOptions = new ParallelOptions { MaxDegreeOfParallelism = Math.Min(Environment.ProcessorCount / 2, 4) };
        private static readonly BlockingCollection<object[]> InputQueue = new BlockingCollection<object[]>();
        private static readonly BlockingCollection<string> OutputQueue = new BlockingCollection<string>();
        private static long InputRows;
        private static long ProcessedRows;
        private static long OutputRows;
        private static TimeSpan InputTime;
        private static TimeSpan ProcessTime;
        private static TimeSpan OutputTime;

        public static int Main(string[] args)
        {
            var (query, output, connectionString) = ParseArgs(args);
#if DEBUG
            query = "SELECT top 1000 EMail, Domain, cast(IsConfirmed as smallint), convert(varchar, AddDate, 120), cast(IsSend_System_JobRecommendation as smallint), cast(IsNeedConfirm_UkrNet as smallint) FROM EMailSource with (nolock)";
            output = "demo.csv";
            connectionString = "Data Source=beta.rabota.ua;Initial Catalog=RabotaUA2;Integrated Security=False;User ID=sa;Password=rabota;";
#endif
            
            if (string.IsNullOrEmpty(query) || string.IsNullOrEmpty(output) || string.IsNullOrEmpty(connectionString))
            {
                PrintHelpMessage();
                return 1;
            }

            var global = Stopwatch.StartNew();

            try
            {
                using var connection = new SqlConnection(connectionString);
                connection.Open();
                using var noExecCommand = new SqlCommand("SET NOEXEC ON", connection);
                noExecCommand.ExecuteNonQuery();
                using var queryCommand = new SqlCommand(query, connection);
                queryCommand.ExecuteNonQuery();
            }
            catch (Exception ex)
            {
                Console.WriteLine("There was error while validating query:");
                Console.WriteLine(ex.Message);
                return 1;
            }

            var status = new CancellationTokenSource();
            if (Environment.UserInteractive)
            {
                Task.Run(() =>
                {
                    Console.WriteLine("Read -> Process -> Write");
                    while (!status.IsCancellationRequested)
                    {
                        Console.Write($"{InputRows:N0} -> {ProcessedRows:N0} -> {OutputRows:N0} in {global.Elapsed}\r");
                        Thread.Sleep(200);
                    }
                }, status.Token);
            }

            var read = Task.Run(() =>
            {
                var timer = Stopwatch.StartNew();
                using var connection = new SqlConnection(connectionString);
                using var command = new SqlCommand(query, connection) { CommandTimeout = 0 };
                connection.Open();
                using var reader = command.ExecuteReader();
                if (reader.HasRows)
                {
                    while (reader.Read())
                    {
                        var values = new object[reader.FieldCount];
                        reader.GetValues(values);
                        InputQueue.Add(values, status.Token);
                        Interlocked.Increment(ref InputRows);
                    }
                }

                InputQueue.CompleteAdding();
                InputTime = timer.Elapsed;
            }, status.Token);

            var process = Task.Run(() =>
            {
                var timer = Stopwatch.StartNew();
                Parallel.ForEach(InputQueue.GetConsumingEnumerable(), ParallelOptions, values =>
                {
                    var sb = new StringBuilder();
                    for (var i = 0; i < values.Length; i++)
                    {
                        var str = (values[i] ?? "").ToString()?.Trim();

                        if (values[i] is string)
                        {
                            //str = HttpUtility.UrlDecode(str);
                            str = Regex.Replace(str, @"[\u0000-\u001F]", string.Empty);
                            str = Regex.Replace(str, "\\s+", " ");
                            str = str.Replace("\"", "\"\"");
                        }

                        sb.AppendFormat("{0}\"{1}\"", i > 0 ? "," : "", str);
                    }

                    OutputQueue.Add(sb.ToString(), status.Token);
                    Interlocked.Increment(ref ProcessedRows);
                });
                OutputQueue.CompleteAdding();
                ProcessTime = timer.Elapsed;
            }, status.Token);

            var write = Task.Run(() =>
            {
                var timer = Stopwatch.StartNew();
                using var writer = new StreamWriter(output, false, new UTF8Encoding(false)) { NewLine = "\n" };
                foreach (var row in OutputQueue.GetConsumingEnumerable())
                {
                    writer.WriteLine(row);
                    Interlocked.Increment(ref OutputRows);
                }

                OutputTime = timer.Elapsed;
            }, status.Token);

            Task.WaitAll(read, process, write);
            status.Cancel();
            Console.WriteLine();
            Console.WriteLine($"{InputRows:N0} read in {InputTime}");
            Console.WriteLine($"{ProcessedRows:N0} process in {ProcessTime}");
            Console.WriteLine($"{OutputRows:N0} write in {OutputTime}");
            Console.WriteLine($"Done in {global.Elapsed}");

            return 0;
        }

        private static (string query, string output, string connectionString) ParseArgs(string[] args)
        {
            var options = args.ToDictionary(arg => arg.TrimStart('-').Split('=').FirstOrDefault(), arg => arg.Split('=').LastOrDefault());

            var query = options.GetOrDefault("query", "");
            var input = options.GetOrDefault("input", "");
            var output = options.GetOrDefault("output", "");
            var server = options.GetOrDefault("server", "localhost");
            var database = options.GetOrDefault("database", "RabotaUA2");
            var username = options.GetOrDefault("username", "sa");
            var password = options.GetOrDefault("password", "");

            var hasValidInput = !string.IsNullOrEmpty(query) || !string.IsNullOrEmpty(input) && File.Exists(input);
            var hasValidOutput = !string.IsNullOrEmpty(output);

            if (!(hasValidInput && hasValidOutput))
            {
                return ("", "", "");
            }

            if (string.IsNullOrEmpty(query))
            {
                query = File.ReadAllText(input);
            }

            var builder = new SqlConnectionStringBuilder
            {
                ApplicationName = "sql2csv",
                DataSource = server,
                InitialCatalog = database,
                PersistSecurityInfo = true,
                ConnectTimeout = 5
            };

            if (string.IsNullOrEmpty(password))
            {
                builder.IntegratedSecurity = true;
            }
            else
            {
                builder.UserID = username;
                builder.Password = password;
            }

            return (query, output, builder.ConnectionString);
        }

        private static void PrintHelpMessage() => Console.WriteLine($@"
SQL2CSV Version: {Assembly.GetExecutingAssembly().GetName().Version}

Required arguments:

  --query=""select top 10 * from city"" - required if there is no input argument
  --input=query.sql - required if there is no query argument
  --output=city.csv

Optional arguments:

  --server=localhost
  --database=RabotaUA2
  --password=password

Usage examples:

sql2csv --query=""select top 10 * from city"" --output=city.csv
");
    }
}
