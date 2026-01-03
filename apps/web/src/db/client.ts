import postgres from 'postgres';
import { drizzle, PostgresJsDatabase } from 'drizzle-orm/postgres-js';
import * as schema from './schema';

let connection: ReturnType<typeof postgres> | null = null;
let database: PostgresJsDatabase<typeof schema> | null = null;

function getDatabase(): PostgresJsDatabase<typeof schema> {
  if (!database) {
    const connectionString = process.env.DATABASE_URL;
    if (!connectionString) {
      throw new Error('DATABASE_URL environment variable is not set');
    }
    connection = postgres(connectionString, {
      prepare: false, // Required for Supabase connection pooling
    });
    database = drizzle(connection, { schema });
  }
  return database;
}

// Lazy-loaded db proxy
export const db = new Proxy({} as PostgresJsDatabase<typeof schema>, {
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
