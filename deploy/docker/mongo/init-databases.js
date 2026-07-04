// Local development MongoDB databases and collections.
// Applications authenticate through the admin database with the local root user.

const databases = {
  zhicore_content: ["post_bodies", "content_body_cleanup_tasks", "content_body_repair_tasks"],
  zhicore_ranking: ["ranking_archives"],
};

for (const [databaseName, collections] of Object.entries(databases)) {
  const database = db.getSiblingDB(databaseName);
  for (const collection of collections) {
    if (!database.getCollectionNames().includes(collection)) {
      database.createCollection(collection);
    }
  }
}
