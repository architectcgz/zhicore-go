# Async Interaction Code Examples

Use these TypeScript/React examples when implementing or reviewing frontend async flows: search, filters, dashboards, uploads, autosave, polling, or streaming.

## Independent I/O: Start Together, Join Later

Bad: this reads cleanly but makes independent requests run in series.

```ts
async function loadDashboard(userId: string) {
  const user = await api.getUser(userId);
  const orders = await api.getOrders(user.id);
  const recommendations = await api.getRecommendations(user.id);

  return { user, orders, recommendations };
}
```

Good: only await the dependency first, then start independent work together.

```ts
async function loadDashboard(userId: string, signal: AbortSignal) {
  const user = await api.getUser(userId, { signal });

  const ordersPromise = api.getOrders(user.id, { signal });
  const recommendationsPromise = api.getRecommendations(user.id, { signal });

  const [orders, recommendations] = await Promise.all([
    ordersPromise,
    recommendationsPromise,
  ]);

  return { user, orders, recommendations };
}
```

## Local Degradation: Encode Failure Semantics

Bad: one optional widget failure breaks the whole screen.

```ts
const [orders, recommendations, ads] = await Promise.all([
  api.getOrders(userId),
  api.getRecommendations(userId),
  api.getAds(userId),
]);
```

Good: decide which data is required and which data can degrade.

```ts
const [orders, recommendations, ads] = await Promise.all([
  api.getOrders(userId, { signal }), // Required: fail the workflow.
  api.getRecommendations(userId, { signal }).catch((error) => {
    reportWarning("recommendations failed", error);
    return [];
  }),
  api.getAds(userId, { signal }).catch((error) => {
    reportWarning("ads failed", error);
    return null;
  }),
]);
```

Use `Promise.allSettled` when every child result needs an explicit success/failure record.

```ts
const results = await Promise.allSettled([
  api.getPreview(file, { signal }),
  api.getExtractedText(file, { signal }),
  api.getSafetyScore(file, { signal }),
]);

const [preview, extractedText, safetyScore] = results.map((result) =>
  result.status === "fulfilled" ? result.value : null,
);
```

## Cancellation: Pass The Signal Through Every Layer

Bad: the component creates a signal, but the helper drops it.

```ts
function SearchBox() {
  useEffect(() => {
    const controller = new AbortController();
    searchProducts(query); // Signal is lost here.
    return () => controller.abort();
  }, [query]);
}

async function searchProducts(query: string) {
  const response = await fetch(`/api/search?q=${encodeURIComponent(query)}`);
  return response.json();
}
```

Good: every helper accepts and forwards the same signal.

```ts
function SearchBox({ query }: { query: string }) {
  useEffect(() => {
    const controller = new AbortController();

    void searchProducts(query, { signal: controller.signal });

    return () => controller.abort();
  }, [query]);
}

async function searchProducts(
  query: string,
  options: { signal: AbortSignal },
) {
  return fetchJson<Product[]>(
    `/api/search?q=${encodeURIComponent(query)}`,
    options,
  );
}

async function fetchJson<T>(url: string, options: { signal: AbortSignal }) {
  const response = await fetch(url, { signal: options.signal });
  if (!response.ok) throw new Error(`HTTP ${response.status}`);
  return response.json() as Promise<T>;
}
```

## Request Identity: Re-check State After Await

Bad: an older request can finish last and overwrite the newest UI state.

```ts
async function runSearch(query: string) {
  const result = await api.search(query);
  setResult(result);
}
```

Good: track request identity and ignore stale completions.

```ts
function useProductSearch(query: string) {
  const requestSeq = useRef(0);
  const [result, setResult] = useState<Product[]>([]);

  useEffect(() => {
    const requestId = ++requestSeq.current;
    const controller = new AbortController();

    async function run() {
      try {
        const nextResult = await api.search(query, {
          signal: controller.signal,
        });

        // Every await is a handoff point; only the newest request may update UI.
        if (controller.signal.aborted || requestId !== requestSeq.current) {
          return;
        }

        setResult(nextResult);
      } catch (error) {
        if (!controller.signal.aborted) {
          reportError(error);
        }
      }
    }

    void run();
    return () => controller.abort();
  }, [query]);

  return result;
}
```

## A Complete Dashboard Hook

This combines dependency ordering, parallel joins, local degradation, cancellation, and stale-response protection.

```ts
type DashboardState =
  | { status: "loading" }
  | { status: "ready"; data: DashboardData }
  | { status: "error"; error: Error };

function useDashboard(userId: string) {
  const requestSeq = useRef(0);
  const [state, setState] = useState<DashboardState>({ status: "loading" });

  useEffect(() => {
    const requestId = ++requestSeq.current;
    const controller = new AbortController();
    const { signal } = controller;

    async function load() {
      setState({ status: "loading" });

      try {
        const user = await api.getUser(userId, { signal });

        if (signal.aborted || requestId !== requestSeq.current) return;

        const [orders, recommendations, metrics] = await Promise.all([
          api.getOrders(user.id, { signal }),
          api.getRecommendations(user.id, { signal }).catch((error) => {
            reportWarning("recommendations failed", error);
            return [];
          }),
          api.getMetrics(user.id, { signal }).catch((error) => {
            reportWarning("metrics failed", error);
            return null;
          }),
        ]);

        if (signal.aborted || requestId !== requestSeq.current) return;

        setState({
          status: "ready",
          data: { user, orders, recommendations, metrics },
        });
      } catch (error) {
        if (signal.aborted || requestId !== requestSeq.current) return;
        setState({ status: "error", error: normalizeError(error) });
      }
    }

    void load();
    return () => controller.abort();
  }, [userId]);

  return state;
}
```

## Avoid Await Inside Locks

Bad: the lock is held while I/O is pending.

```ts
async function renameProject(projectId: string, name: string) {
  await projectLock.acquire(projectId);
  try {
    const project = projectStore.get(projectId);
    const saved = await api.saveProject({ ...project, name });
    projectStore.replace(projectId, saved);
  } finally {
    projectLock.release(projectId);
  }
}
```

Good: snapshot under normal state access, wait outside the critical section, then apply a guarded update.

```ts
async function renameProject(projectId: string, name: string) {
  const snapshot = projectStore.snapshot(projectId);
  const saved = await api.saveProject({ ...snapshot, name });

  await projectLock.runExclusive(projectId, () => {
    // Guard against state changes that happened while I/O was in flight.
    projectStore.replaceIfVersion(projectId, snapshot.version, saved);
  });
}
```

## CPU Work: Move It Off The Main Event Loop

Bad: `async` does not stop CPU-heavy work from blocking rendering.

```ts
async function exportCsv(rows: Row[]) {
  const csv = buildLargeCsv(rows); // Blocks the main thread.
  downloadBlob(csv, "report.csv");
}
```

Good: move the expensive transform to an execution context outside the UI thread. In the browser this usually means a Web Worker. In Node or backend code, use `worker_threads`, a process pool, a queue worker, or a server-side job. Application code chooses the execution context; the operating system schedules the actual CPU core.

`await` is still useful here, but only for waiting on the worker or job result. It does not make synchronous computation non-blocking by itself.

```ts
const worker = new Worker(new URL("./csv-export.worker.ts", import.meta.url), {
  type: "module",
});

function exportCsv(rows: Row[]) {
  worker.postMessage({ type: "export-csv", rows });
}

worker.addEventListener("message", (event: MessageEvent<WorkerResult>) => {
  if (event.data.type === "csv-ready") {
    downloadBlob(event.data.csv, "report.csv");
  }
});
```
