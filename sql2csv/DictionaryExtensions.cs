using System.Collections.Generic;

namespace sql2csv
{
	public static class DictionaryExtensions
	{
		public static TValue GetOrDefault<TKey, TValue>(this Dictionary<TKey, TValue> dict, TKey key, TValue val) => dict.ContainsKey(key) && dict[key] != null ? dict[key] : val;
	}
}