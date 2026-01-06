import Database from 'better-sqlite3';

// Singleton database instance (in-memory)
let db: Database.Database | null = null;

export function getDatabase(): Database.Database {
  if (!db) {
    db = new Database(':memory:');
    seedDatabase(db);
  }
  return db;
}

function seedDatabase(db: Database.Database) {
  // Create products table
  db.exec(`
    CREATE TABLE products (
      id INTEGER PRIMARY KEY,
      name TEXT NOT NULL,
      price DECIMAL(10,2) NOT NULL,
      category TEXT NOT NULL,
      stock INTEGER NOT NULL,
      description TEXT
    )
  `);

  // Create orders table
  db.exec(`
    CREATE TABLE orders (
      id INTEGER PRIMARY KEY,
      product_id INTEGER NOT NULL,
      customer_name TEXT NOT NULL,
      quantity INTEGER NOT NULL,
      total_price DECIMAL(10,2) NOT NULL,
      status TEXT NOT NULL,
      created_at TEXT NOT NULL,
      FOREIGN KEY (product_id) REFERENCES products(id)
    )
  `);

  // Create customers table
  db.exec(`
    CREATE TABLE customers (
      id INTEGER PRIMARY KEY,
      name TEXT NOT NULL,
      email TEXT UNIQUE NOT NULL,
      tier TEXT NOT NULL,
      total_spent DECIMAL(10,2) DEFAULT 0,
      created_at TEXT NOT NULL
    )
  `);

  // Insert sample products
  const insertProduct = db.prepare(
    'INSERT INTO products (name, price, category, stock, description) VALUES (?, ?, ?, ?, ?)'
  );

  const products = [
    ['MacBook Pro 14"', 1999.00, 'laptops', 45, 'Apple M3 Pro chip, 18GB RAM, 512GB SSD'],
    ['MacBook Air 13"', 1099.00, 'laptops', 120, 'Apple M3 chip, 8GB RAM, 256GB SSD'],
    ['iPhone 15 Pro', 999.00, 'phones', 200, '6.1" display, A17 Pro chip, 256GB'],
    ['iPhone 15', 799.00, 'phones', 350, '6.1" display, A16 chip, 128GB'],
    ['AirPods Pro 2', 249.00, 'audio', 500, 'Active noise cancellation, spatial audio'],
    ['AirPods Max', 549.00, 'audio', 80, 'Over-ear headphones, computational audio'],
    ['iPad Pro 12.9"', 1099.00, 'tablets', 90, 'M2 chip, Liquid Retina XDR display'],
    ['iPad Air', 599.00, 'tablets', 150, 'M1 chip, 10.9" display'],
    ['Apple Watch Ultra 2', 799.00, 'wearables', 60, '49mm titanium, precision GPS'],
    ['Apple Watch Series 9', 399.00, 'wearables', 200, '41mm aluminum, always-on display'],
  ];

  for (const product of products) {
    insertProduct.run(...product);
  }

  // Insert sample customers
  const insertCustomer = db.prepare(
    'INSERT INTO customers (name, email, tier, total_spent, created_at) VALUES (?, ?, ?, ?, ?)'
  );

  const customers = [
    ['Alice Johnson', 'alice@example.com', 'gold', 5420.00, '2023-01-15'],
    ['Bob Smith', 'bob@example.com', 'silver', 2100.00, '2023-03-22'],
    ['Carol Williams', 'carol@example.com', 'bronze', 899.00, '2023-06-10'],
    ['David Brown', 'david@example.com', 'gold', 8750.00, '2022-11-05'],
    ['Eve Davis', 'eve@example.com', 'platinum', 15200.00, '2022-05-18'],
  ];

  for (const customer of customers) {
    insertCustomer.run(...customer);
  }

  // Insert sample orders
  const insertOrder = db.prepare(
    'INSERT INTO orders (product_id, customer_name, quantity, total_price, status, created_at) VALUES (?, ?, ?, ?, ?, ?)'
  );

  const orders = [
    [1, 'Alice Johnson', 1, 1999.00, 'delivered', '2024-01-15'],
    [3, 'Alice Johnson', 2, 1998.00, 'shipped', '2024-01-18'],
    [5, 'Bob Smith', 1, 249.00, 'delivered', '2024-01-10'],
    [7, 'Carol Williams', 1, 1099.00, 'processing', '2024-01-20'],
    [9, 'David Brown', 1, 799.00, 'shipped', '2024-01-19'],
    [2, 'Eve Davis', 2, 2198.00, 'delivered', '2024-01-12'],
    [4, 'Eve Davis', 1, 799.00, 'delivered', '2024-01-14'],
    [6, 'Alice Johnson', 1, 549.00, 'pending', '2024-01-21'],
  ];

  for (const order of orders) {
    insertOrder.run(...order);
  }

  console.log('[Playground] Database seeded with sample data');
}

export function executeQuery(sql: string): { rows: unknown[]; columns: string[] } {
  const db = getDatabase();

  // Security: Only allow SELECT queries
  const trimmedSql = sql.trim().toLowerCase();
  if (!trimmedSql.startsWith('select')) {
    throw new Error('Only SELECT queries are allowed for security reasons');
  }

  try {
    const stmt = db.prepare(sql);
    const rows = stmt.all();
    const columns = stmt.columns().map(c => c.name);
    return { rows, columns };
  } catch (error) {
    throw new Error(`SQL Error: ${error instanceof Error ? error.message : 'Unknown error'}`);
  }
}
