package physicalplan

import (
	"testing"

	"github.com/apache/arrow/go/v7/arrow/memory"
	"github.com/stretchr/testify/require"

	"github.com/polarsignals/arcticdb/dynparquet"
	"github.com/polarsignals/arcticdb/query/logicalplan"
)

func TestBuildPhysicalPlan(t *testing.T) {
	p := (&logicalplan.Builder{}).
		Scan(nil, "table1").
		Filter(logicalplan.Col("labels.test").Eq(logicalplan.Literal("abc"))).
		Aggregate(
			logicalplan.Sum(logicalplan.Col("value")).Alias("value_sum"),
			logicalplan.Col("stacktrace"),
		).
		Project("stacktrace", "value_sum").
		Build()

	optimizers := []logicalplan.Optimizer{
		&logicalplan.ProjectionPushDown{},
		&logicalplan.FilterPushDown{},
	}

	for _, optimizer := range optimizers {
		optimizer.Optimize(p)
	}

	_, err := Build(memory.DefaultAllocator, dynparquet.NewSampleSchema(), p)
	require.NoError(t, err)
}
