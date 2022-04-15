package query

import (
	"github.com/apache/arrow/go/v8/arrow"
	"github.com/apache/arrow/go/v8/arrow/memory"

	"github.com/polarsignals/arcticdb/dynparquet"
	"github.com/polarsignals/arcticdb/query/logicalplan"
	"github.com/polarsignals/arcticdb/query/physicalplan"
)

type Builder interface {
	Aggregate(aggExpr logicalplan.Expr, groupExprs ...logicalplan.ColumnExpr) Builder
	Filter(expr logicalplan.Expr) Builder
	Distinct(expr ...logicalplan.ColumnExpr) Builder
	Project(projections ...string) Builder
	Execute(callback func(r arrow.Record) error) error
}

type LocalEngine struct {
	pool          memory.Allocator
	tableProvider logicalplan.TableProvider
}

func NewEngine(
	pool memory.Allocator,
	tableProvider logicalplan.TableProvider,
) *LocalEngine {
	return &LocalEngine{
		pool:          pool,
		tableProvider: tableProvider,
	}
}

type LocalQueryBuilder struct {
	pool        memory.Allocator
	planBuilder logicalplan.Builder
}

func (e *LocalEngine) ScanTable(name string, options ...logicalplan.IterateOption) Builder {
	return LocalQueryBuilder{
		pool:        e.pool,
		planBuilder: (&logicalplan.Builder{}).Scan(e.tableProvider, name, options...),
	}
}

func (e *LocalEngine) ScanSchema(name string) Builder {
	return LocalQueryBuilder{
		pool:        e.pool,
		planBuilder: (&logicalplan.Builder{}).ScanSchema(e.tableProvider, name),
	}
}

func (b LocalQueryBuilder) Aggregate(
	aggExpr logicalplan.Expr,
	groupExprs ...logicalplan.ColumnExpr,
) Builder {
	return LocalQueryBuilder{
		pool:        b.pool,
		planBuilder: b.planBuilder.Aggregate(aggExpr, groupExprs...),
	}
}

func (b LocalQueryBuilder) Filter(
	expr logicalplan.Expr,
) Builder {
	return LocalQueryBuilder{
		pool:        b.pool,
		planBuilder: b.planBuilder.Filter(expr),
	}
}

func (b LocalQueryBuilder) Distinct(
	expr ...logicalplan.ColumnExpr,
) Builder {
	return LocalQueryBuilder{
		pool:        b.pool,
		planBuilder: b.planBuilder.Distinct(expr...),
	}
}

func (b LocalQueryBuilder) Project(
	projections ...string,
) Builder {
	return LocalQueryBuilder{
		pool:        b.pool,
		planBuilder: b.planBuilder.Project(projections...),
	}
}

func (b LocalQueryBuilder) Execute(callback func(r arrow.Record) error) error {
	logicalPlan := b.planBuilder.Build()

	optimizers := []logicalplan.Optimizer{
		&logicalplan.ProjectionPushDown{},
		&logicalplan.FilterPushDown{},
		&logicalplan.DistinctPushDown{},
	}

	for _, optimizer := range optimizers {
		optimizer.Optimize(logicalPlan)
	}

	phyPlan, err := physicalplan.Build(
		b.pool,
		dynparquet.NewSampleSchema(),
		logicalPlan,
	)
	if err != nil {
		return err
	}

	return phyPlan.Execute(b.pool, callback)
}
