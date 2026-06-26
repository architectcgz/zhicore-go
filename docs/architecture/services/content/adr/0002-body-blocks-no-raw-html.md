# Content stores body as structured blocks and rejects raw HTML

Content stores article body as versioned structured blocks, with parser strategies selected by `schemaVersion`, and rejects user-submitted raw HTML. We chose blocks because media references, minimum text length, review, search extraction, future AI summary, and schema migration all need a controlled document model. Raw HTML was rejected because it would make XSS, style pollution, arbitrary iframe/script injection, and canonical hashing much harder to govern.

**Status:** accepted

**Consequences:** embed-like content must use `external_embed` with a provider allowlist, inline links must use safe link marks, and parser implementations live behind `BodyParserRegistry`.
