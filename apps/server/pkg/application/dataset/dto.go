package dataset

// CreateDatasetRequest is the body of POST .../datasets.
// Name is required. Description is optional but stored verbatim when present.
type CreateDatasetRequest struct {
	Name        string  `json:"name"`
	Description *string `json:"description,omitempty"`
}

// UpdateDatasetRequest is the body of PATCH .../datasets/{id}.
// Nil pointers mean "leave unchanged" (matches entity.DatasetUpdate semantics).
type UpdateDatasetRequest struct {
	Name        *string `json:"name,omitempty"`
	Description *string `json:"description,omitempty"`
}

// CreateDatasetItemRequest is the body of POST .../items (manual authoring).
// `input` is required; `expected` and `metadata` are optional and stored as-is.
type CreateDatasetItemRequest struct {
	Input    any            `json:"input"`
	Expected any            `json:"expected,omitempty"`
	Metadata map[string]any `json:"metadata,omitempty"`
}

// AddDatasetItemFromTraceRequest is the body of POST .../items/from-trace.
// The service verifies the span belongs to the trace, the trace belongs to the
// project, and seeds `input` from the span. `expected` is NOT auto-pulled from
// span.output — what *should* have happened ≠ what *did* happen, and seeding
// it would bake every buggy output into a "gold" expectation. The caller is
// the one who knows the right answer (or leaves it blank for an LLM-judge).
type AddDatasetItemFromTraceRequest struct {
	TraceID  string         `json:"traceId"`
	SpanID   string         `json:"spanId"`
	Expected any            `json:"expected,omitempty"`
	Metadata map[string]any `json:"metadata,omitempty"`
}

// ImportDatasetItemsRequest is the body of POST .../items/import.
// Used for CSV/JSON bulk seeding. Each entry follows the same shape as
// CreateDatasetItemRequest — kept inline to avoid a partial-success ambiguity.
type ImportDatasetItemsRequest struct {
	Items []CreateDatasetItemRequest `json:"items"`
}
