---
paths: "ee/**/*.go,apps/web/src/ee/**/*.ts,apps/web/src/ee/**/*.tsx"
---

# Enterprise Code Rules

## Code Separation

Enterprise code MUST stay in designated folders:
- Backend: `ee/server/`
- Frontend: `apps/web/src/ee/`

## Never Mix OSS and EE

```go
// WRONG - EE code in OSS path
// apps/server/pkg/application/billing/service.go

// CORRECT - EE code in EE path
// ee/server/application/billing/service.go
```

## EE Extends OSS via RouterExtension

```go
// ee/server/interfaces/http/extension.go
type Extension struct {
    billingHandler *handler.BillingHandler
    orgHandler     *handler.OrganizationHandler
}

func (e *Extension) RegisterRoutes(r chi.Router) {
    r.Route("/billing", func(r chi.Router) {
        r.Get("/subscription", e.billingHandler.GetSubscription)
    })
}
```

## EE Imports OSS, Not Vice Versa

```go
// CORRECT - EE imports OSS
// ee/server/cmd/server/main.go
import (
    "github.com/lelemondev/lelemon/apps/server/pkg/domain/entity"
)

// WRONG - OSS should never import EE
// apps/server/pkg/application/trace/service.go
import (
    "github.com/lelemondev/lelemon/ee/server/domain/entity"  // DON'T!
)
```

## Feature Gating in Frontend

```tsx
// apps/web/src/ee/components/FeatureGate.tsx
import { useEE } from '@/ee/lib/ee-context';

export function FeatureGate({ feature, children }: Props) {
  const { hasFeature } = useEE();
  if (!hasFeature(feature)) return null;
  return <>{children}</>;
}

// Usage
<FeatureGate feature="billing">
  <BillingDashboard />
</FeatureGate>
```

## Proprietary License

All code in `ee/` folder is under proprietary license.
See `ee/LICENSE` for terms.
