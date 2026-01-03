import { neon, NeonQueryFunction } from '@neondatabase/serverless';
import { drizzle, NeonHttpDatabase } from 'drizzle-orm/neon-http';
import * as schema from './schema';

let sql: NeonQueryFunction<false, false> | null = null;
let database: NeonHttpDatabase<typeof schema> | null = null;

function getDatabase(): NeonHttpDatabase<typeof schema> {
  if (!database) {
    const connectionString = process.env.DATABASE_URL;
    if (!connectionString) {
      throw new Error('DATABASE_URL environment variable is not set');
    }
    sql = neon(connectionString);
    database = drizzle(sql, { schema });
  }
  return database;
}

// Lazy-loaded db proxy
export const db = new Proxy({} as NeonHttpDatabase<typeof schema>, {
  get(_, prop) {
    const database = getDatabase();
    const value = database[prop as keyof typeof database];
    if (typeof value === 'function') {
      return value.bind(database);
    }
    return value;
  },
});

export type Database = typeof db;
