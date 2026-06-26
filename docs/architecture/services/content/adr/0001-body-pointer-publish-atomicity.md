# Content uses PostgreSQL body pointers for publish atomicity

Content publishes article bodies by first writing an immutable MongoDB body snapshot, then committing a PostgreSQL transaction that switches `published_body_id`, published metadata, outbox events, and cleanup tasks together. We chose this instead of a PostgreSQL + MongoDB distributed transaction because 2PC/XA would add coordinator recovery, hanging transactions, and operational complexity, while the product only needs user-visible atomicity. We also rejected “PostgreSQL marks the post published and MongoDB body is compensated later” because article body is core read data; a successful publish must not produce a published post that readers cannot open.

**Status:** accepted

**Consequences:** PostgreSQL is the visibility source of truth, MongoDB stores body documents, and old draft/snapshot bodies are cleaned by body-id tasks after PostgreSQL no longer references them.
