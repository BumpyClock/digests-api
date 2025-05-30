// ABOUTME: Example ViewModel for WinUI 3 applications using the Digests library
// ABOUTME: Demonstrates MVVM pattern with async operations and data binding

using CommunityToolkit.Mvvm.ComponentModel;
using CommunityToolkit.Mvvm.Input;
using System;
using System.Collections.ObjectModel;
using System.Linq;
using System.Threading.Tasks;
using Microsoft.UI.Dispatching;

namespace Digests.ViewModels
{
    public partial class FeedViewModel : ObservableObject, IDisposable
    {
        private readonly DigestsClient _digestsClient;
        private readonly DispatcherQueue _dispatcherQueue;

        [ObservableProperty]
        private ObservableCollection<Feed> feeds = new();

        [ObservableProperty]
        private ObservableCollection<FeedItem> allItems = new();

        [ObservableProperty]
        private bool isLoading;

        [ObservableProperty]
        private string errorMessage;

        [ObservableProperty]
        private string feedUrl = "https://news.ycombinator.com/rss";

        [ObservableProperty]
        private bool enableEnrichment = true;

        public FeedViewModel()
        {
            // Initialize with SQLite cache for better performance
            _digestsClient = new DigestsClient("sqlite", "feeds_cache.db");
            _dispatcherQueue = DispatcherQueue.GetForCurrentThread();
        }

        [RelayCommand]
        private async Task LoadFeedAsync()
        {
            if (string.IsNullOrWhiteSpace(FeedUrl))
            {
                ErrorMessage = "Please enter a valid feed URL";
                return;
            }

            IsLoading = true;
            ErrorMessage = null;

            try
            {
                Feed feed;
                if (EnableEnrichment)
                {
                    feed = await _digestsClient.ParseFeedAsync(FeedUrl);
                }
                else
                {
                    feed = await _digestsClient.ParseFeedWithoutEnrichmentAsync(FeedUrl);
                }

                // Update UI on the dispatcher thread
                _dispatcherQueue.TryEnqueue(() =>
                {
                    Feeds.Clear();
                    Feeds.Add(feed);
                    
                    AllItems.Clear();
                    foreach (var item in feed.Items)
                    {
                        AllItems.Add(item);
                    }
                });
            }
            catch (Exception ex)
            {
                ErrorMessage = $"Error loading feed: {ex.Message}";
            }
            finally
            {
                IsLoading = false;
            }
        }

        [RelayCommand]
        private async Task LoadMultipleFeedsAsync()
        {
            var feedUrls = new[]
            {
                "https://news.ycombinator.com/rss",
                "https://feeds.arstechnica.com/arstechnica/index",
                "https://www.theverge.com/rss/index.xml"
            };

            IsLoading = true;
            ErrorMessage = null;

            try
            {
                var feeds = await _digestsClient.ParseFeedsAsync(feedUrls);

                _dispatcherQueue.TryEnqueue(() =>
                {
                    Feeds.Clear();
                    AllItems.Clear();

                    foreach (var feed in feeds)
                    {
                        Feeds.Add(feed);
                        foreach (var item in feed.Items)
                        {
                            AllItems.Add(item);
                        }
                    }

                    // Sort all items by published date
                    var sortedItems = AllItems.OrderByDescending(i => i.Published).ToList();
                    AllItems.Clear();
                    foreach (var item in sortedItems)
                    {
                        AllItems.Add(item);
                    }
                });
            }
            catch (Exception ex)
            {
                ErrorMessage = $"Error loading feeds: {ex.Message}";
            }
            finally
            {
                IsLoading = false;
            }
        }

        [RelayCommand]
        private async Task SearchFeedsAsync(string query)
        {
            if (string.IsNullOrWhiteSpace(query))
            {
                ErrorMessage = "Please enter a search query";
                return;
            }

            IsLoading = true;
            ErrorMessage = null;

            try
            {
                var results = await _digestsClient.SearchAsync(query);

                _dispatcherQueue.TryEnqueue(() =>
                {
                    Feeds.Clear();
                    AllItems.Clear();

                    // For search results, we'd typically show them differently
                    // This is just an example
                    foreach (var result in results)
                    {
                        // You might want to parse each discovered feed
                        // For now, just show the search results
                        var placeholderFeed = new Feed
                        {
                            Title = result.Title,
                            Description = result.Description,
                            Url = result.FeedUrl,
                            Items = Array.Empty<FeedItem>()
                        };
                        Feeds.Add(placeholderFeed);
                    }
                });
            }
            catch (Exception ex)
            {
                ErrorMessage = $"Error searching: {ex.Message}";
            }
            finally
            {
                IsLoading = false;
            }
        }

        public void Dispose()
        {
            _digestsClient?.Dispose();
        }
    }
}