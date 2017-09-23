using System;
using System.Data.SqlClient;
using System.IO;
using System.Linq;

namespace sql2csv
{
	public class Program
	{
		public static int Main(string[] args)
		{
			var options = args.ToDictionary(arg => arg.TrimStart('-').Split('=').FirstOrDefault(), arg => arg.Split('=').LastOrDefault());

			var query = options.GetOrDefault("query", "");
			var input = options.GetOrDefault("input", "");
			var output = options.GetOrDefault("output", "");
			var server = options.GetOrDefault("server", "localhost");
			var database = options.GetOrDefault("database", "RabotaUA2");
			var username = options.GetOrDefault("username", "");
			var password = options.GetOrDefault("password", "");

			var hasValidInput = !string.IsNullOrEmpty(query) || !string.IsNullOrEmpty(input) && File.Exists(input);
			var hasValidOutput = !string.IsNullOrEmpty(output);

			if (!(hasValidInput && hasValidOutput))
			{
				Console.WriteLine("Required arguments:");
				Console.WriteLine();
				Console.WriteLine("  --query=\"select top 10 * from city\" - required if there is no input argument");
				Console.WriteLine("  --input=query.sql - required if there is no query argument");
				Console.WriteLine("  --output=city.csv");
				Console.WriteLine();
				Console.WriteLine("Optional arguments:");
				Console.WriteLine("  --server=localhost");
				Console.WriteLine("  --database=RabotaUA2");
				Console.WriteLine("  --username=sa");
				Console.WriteLine("  --password=password");
				Console.WriteLine();
				Console.WriteLine("Usage examples:");
				Console.WriteLine();
				Console.WriteLine("sql2csv --query=\"select top 10 * from city\" --output=city.csv");
				Console.WriteLine();
				return 1;
			}

			if (string.IsNullOrEmpty(query))
			{
				query = File.ReadAllText(input);
			}

			var builder = new SqlConnectionStringBuilder
			{
				DataSource = server,
				InitialCatalog = database,
				PersistSecurityInfo = true
			};

			if (string.IsNullOrEmpty(username) && string.IsNullOrEmpty(password))
			{
				builder.IntegratedSecurity = true;
			}
			else
			{
				builder.UserID = username;
				builder.Password = password;
			}



			using (var connection = new SqlConnection(builder.ConnectionString))
			{
				using (var command = new SqlCommand(query, connection) { CommandTimeout = 0 })
				{
					connection.Open();
					using (var reader = command.ExecuteReader())
					{
						if (reader.HasRows)
						{
							while (reader.Read())
							{
								//stats[i] += 1;
								//Console.Write("{0}\r", string.Join(", ", stats.Select(num => string.Format("{0:N0}", num))));
							}
						}
						reader.Close();
					}
				}
			}

			return 0;
		}
	}
}
