---
paths: "apps/web/src/components/**/*.tsx,apps/web/src/app/**/*.tsx"
---

# React Component Rules

## Component Structure

```typescript
'use client'; // Only if needed (hooks, events, browser APIs)

import { useState } from 'react';
import { Card, CardContent } from '@/components/ui/card';

interface Props {
  title: string;
  onSubmit?: (data: FormData) => void;
}

export function MyComponent({ title, onSubmit }: Props) {
  const [state, setState] = useState('');

  return (
    <Card>
      <CardContent>{title}</CardContent>
    </Card>
  );
}
```

## Use shadcn/ui Components

Always prefer existing UI components:
- `Card`, `CardHeader`, `CardContent`, `CardTitle`
- `Button`, `Input`, `Label`
- `Table`, `TableHeader`, `TableRow`, `TableCell`
- `Badge`, `Skeleton`, `Separator`

## Tailwind CSS

Use Tailwind utilities, follow existing patterns:
```typescript
// Spacing
<div className="space-y-6">

// Typography
<h1 className="text-3xl font-bold tracking-tight">

// Colors (use semantic)
<p className="text-muted-foreground">
```

## Client vs Server Components

- Default to Server Components
- Use `'use client'` only when you need:
  - React hooks (useState, useEffect)
  - Event handlers (onClick, onChange)
  - Browser APIs (window, localStorage)
