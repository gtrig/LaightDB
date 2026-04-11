package stress

// StandardQueries is a fixed workload of search strings used for repeatable benchmarks.
// They are chosen to exercise typical hybrid (BM25 + vector) retrieval over mixed vocabulary.
var StandardQueries = []string{
	"error handling and recovery",
	"database storage engine",
	"concurrent access patterns",
	"implementation details",
	"configuration and environment",
	"authentication middleware",
	"vector similarity search",
	"full text indexing",
	"write ahead log",
	"memory table flush",
	"compaction strategy",
	"metadata filters",
	"REST API endpoints",
	"context embedding",
	"chunking and summarization",
}
