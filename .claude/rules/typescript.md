---
paths: "**/*.ts,**/*.tsx"
---

# TypeScript Rules

## Strict Mode
- No `any` types - always be explicit
- Enable all strict compiler options
- No implicit returns on async functions

## Type Annotations
```typescript
// DO: Explicit return types on exports
export async function getTraces(): Promise<Trace[]> { ... }

// DON'T: Implicit any or missing types
export async function getTraces() { ... }
```

## Imports
```typescript
// DO: Use path aliases
import { db } from '@/db/client';

// DON'T: Relative paths for deep imports
import { db } from '../../../db/client';
```

## Null Handling
```typescript
// DO: Use optional chaining and nullish coalescing
const name = user?.name ?? 'Anonymous';

// DON'T: Unsafe access
const name = user.name || 'Anonymous';
```
