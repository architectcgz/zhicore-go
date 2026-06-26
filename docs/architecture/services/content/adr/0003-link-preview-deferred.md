# Content defers link preview to a backend asynchronous feature

Content does not implement link preview in the first phase; external links are rendered as safe links only. We chose to defer it because link preview requires the backend to fetch user-provided URLs, which introduces SSRF protection, redirect validation, response-size limits, caching, retry, and image proxy concerns that are not needed for the first publishing slice. If link preview is implemented later, it must be generated asynchronously by the backend with an SSRF-safe fetcher, and the frontend must only consume cached preview data.

**Status:** accepted

**Consequences:** saving drafts and publishing posts must not block on link preview generation, and frontend code must not independently scrape external URLs for previews.
