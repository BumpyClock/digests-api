// ABOUTME: C# wrapper for the Digests DLL for use in WinUI 3 applications
// ABOUTME: Provides a managed interface to the native Go library

using System;
using System.Runtime.InteropServices;
using System.Text.Json;
using System.Threading.Tasks;

namespace Digests
{
    /// <summary>
    /// Client for interacting with the Digests RSS/Atom feed parser library
    /// </summary>
    public class DigestsClient : IDisposable
    {
        private bool _disposed;
        private bool _initialized;

        #region P/Invoke Declarations

        [DllImport("digests.dll", CallingConvention = CallingConvention.Cdecl)]
        private static extern int DigestsInit();

        [DllImport("digests.dll", CallingConvention = CallingConvention.Cdecl)]
        private static extern int DigestsInitWithCache(string cacheType, string cachePath);

        [DllImport("digests.dll", CallingConvention = CallingConvention.Cdecl)]
        private static extern void DigestsClose();

        [DllImport("digests.dll", CallingConvention = CallingConvention.Cdecl)]
        private static extern IntPtr DigestsParseFeed(string url);

        [DllImport("digests.dll", CallingConvention = CallingConvention.Cdecl)]
        private static extern IntPtr DigestsParseFeeds(string urlsJson);

        [DllImport("digests.dll", CallingConvention = CallingConvention.Cdecl)]
        private static extern IntPtr DigestsParseFeedWithoutEnrichment(string url);

        [DllImport("digests.dll", CallingConvention = CallingConvention.Cdecl)]
        private static extern IntPtr DigestsSearch(string query);

        [DllImport("digests.dll", CallingConvention = CallingConvention.Cdecl)]
        private static extern void DigestsFreeString(IntPtr str);

        #endregion

        /// <summary>
        /// Initialize the Digests client with default settings
        /// </summary>
        public DigestsClient()
        {
            var result = DigestsInit();
            if (result != 0)
            {
                throw new Exception("Failed to initialize Digests client");
            }
            _initialized = true;
        }

        /// <summary>
        /// Initialize the Digests client with cache configuration
        /// </summary>
        /// <param name="cacheType">Cache type: "memory" or "sqlite"</param>
        /// <param name="cachePath">Path to SQLite database (optional for memory cache)</param>
        public DigestsClient(string cacheType, string cachePath = null)
        {
            var result = DigestsInitWithCache(cacheType, cachePath ?? "");
            if (result != 0)
            {
                throw new Exception("Failed to initialize Digests client with cache");
            }
            _initialized = true;
        }

        /// <summary>
        /// Parse a single RSS/Atom feed
        /// </summary>
        /// <param name="url">Feed URL</param>
        /// <returns>Feed data</returns>
        public async Task<Feed> ParseFeedAsync(string url)
        {
            return await Task.Run(() => ParseFeed(url));
        }

        /// <summary>
        /// Parse a single RSS/Atom feed (synchronous)
        /// </summary>
        public Feed ParseFeed(string url)
        {
            EnsureNotDisposed();
            
            var resultPtr = DigestsParseFeed(url);
            try
            {
                var json = Marshal.PtrToStringAnsi(resultPtr);
                var result = JsonSerializer.Deserialize<FeedResult>(json);
                
                if (result?.Error != null)
                {
                    throw new Exception(result.Error);
                }
                
                return JsonSerializer.Deserialize<Feed>(json);
            }
            finally
            {
                DigestsFreeString(resultPtr);
            }
        }

        /// <summary>
        /// Parse multiple RSS/Atom feeds
        /// </summary>
        /// <param name="urls">Array of feed URLs</param>
        /// <returns>Array of feeds</returns>
        public async Task<Feed[]> ParseFeedsAsync(string[] urls)
        {
            return await Task.Run(() => ParseFeeds(urls));
        }

        /// <summary>
        /// Parse multiple RSS/Atom feeds (synchronous)
        /// </summary>
        public Feed[] ParseFeeds(string[] urls)
        {
            EnsureNotDisposed();
            
            var urlsJson = JsonSerializer.Serialize(urls);
            var resultPtr = DigestsParseFeeds(urlsJson);
            try
            {
                var json = Marshal.PtrToStringAnsi(resultPtr);
                var result = JsonSerializer.Deserialize<FeedResult>(json);
                
                if (result?.Error != null)
                {
                    throw new Exception(result.Error);
                }
                
                return JsonSerializer.Deserialize<Feed[]>(json);
            }
            finally
            {
                DigestsFreeString(resultPtr);
            }
        }

        /// <summary>
        /// Parse a feed without enrichment (faster)
        /// </summary>
        public async Task<Feed> ParseFeedWithoutEnrichmentAsync(string url)
        {
            return await Task.Run(() => ParseFeedWithoutEnrichment(url));
        }

        /// <summary>
        /// Parse a feed without enrichment (synchronous)
        /// </summary>
        public Feed ParseFeedWithoutEnrichment(string url)
        {
            EnsureNotDisposed();
            
            var resultPtr = DigestsParseFeedWithoutEnrichment(url);
            try
            {
                var json = Marshal.PtrToStringAnsi(resultPtr);
                var result = JsonSerializer.Deserialize<FeedResult>(json);
                
                if (result?.Error != null)
                {
                    throw new Exception(result.Error);
                }
                
                return JsonSerializer.Deserialize<Feed>(json);
            }
            finally
            {
                DigestsFreeString(resultPtr);
            }
        }

        /// <summary>
        /// Search for RSS feeds
        /// </summary>
        /// <param name="query">Search query</param>
        /// <returns>Search results</returns>
        public async Task<SearchResult[]> SearchAsync(string query)
        {
            return await Task.Run(() => Search(query));
        }

        /// <summary>
        /// Search for RSS feeds (synchronous)
        /// </summary>
        public SearchResult[] Search(string query)
        {
            EnsureNotDisposed();
            
            var resultPtr = DigestsSearch(query);
            try
            {
                var json = Marshal.PtrToStringAnsi(resultPtr);
                var result = JsonSerializer.Deserialize<FeedResult>(json);
                
                if (result?.Error != null)
                {
                    throw new Exception(result.Error);
                }
                
                return JsonSerializer.Deserialize<SearchResult[]>(json);
            }
            finally
            {
                DigestsFreeString(resultPtr);
            }
        }

        private void EnsureNotDisposed()
        {
            if (_disposed)
            {
                throw new ObjectDisposedException(nameof(DigestsClient));
            }
        }

        public void Dispose()
        {
            if (!_disposed && _initialized)
            {
                DigestsClose();
                _disposed = true;
            }
        }
    }

    #region Data Models

    public class Feed
    {
        public string Id { get; set; }
        public string Title { get; set; }
        public string Description { get; set; }
        public string Url { get; set; }
        public string Link { get; set; }
        public string Language { get; set; }
        public DateTime LastUpdated { get; set; }
        public string FeedType { get; set; }
        public FeedItem[] Items { get; set; }
    }

    public class FeedItem
    {
        public string Id { get; set; }
        public string Title { get; set; }
        public string Description { get; set; }
        public string Content { get; set; }
        public string Link { get; set; }
        public DateTime Published { get; set; }
        public string Author { get; set; }
        public string Thumbnail { get; set; }
        public RgbColor ThumbnailColor { get; set; }
        public string[] Categories { get; set; }
        
        // Podcast fields
        public string Duration { get; set; }
        public int Episode { get; set; }
        public int Season { get; set; }
        public string Image { get; set; }
        public string AudioUrl { get; set; }
        public string VideoUrl { get; set; }
    }

    public class RgbColor
    {
        public int R { get; set; }
        public int G { get; set; }
        public int B { get; set; }
    }

    public class SearchResult
    {
        public string Title { get; set; }
        public string Description { get; set; }
        public string Url { get; set; }
        public string FeedUrl { get; set; }
    }

    internal class FeedResult
    {
        public string Error { get; set; }
    }

    #endregion
}