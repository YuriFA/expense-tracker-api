package sqlite

import (
	"strings"
	"time"
)

type sqlClauseBuilder struct {
	clauses []string
	args    []any
}

func newUpdateBuilder() *sqlClauseBuilder {
	return &sqlClauseBuilder{
		clauses: []string{"updated_at = CURRENT_TIMESTAMP"},
	}
}

func newWhereBuilder() *sqlClauseBuilder {
	return &sqlClauseBuilder{
		clauses: []string{},
	}
}

func (b *sqlClauseBuilder) addString(
	col string,
	arg *string,
) *sqlClauseBuilder {
	if arg != nil {
		b.clauses = append(b.clauses, col+" = ?")
		b.args = append(b.args, *arg)
	}
	return b
}

func (b *sqlClauseBuilder) addTimeOp(
	col string,
	arg *time.Time,
	equal string,
) *sqlClauseBuilder {
	if arg != nil {
		b.clauses = append(b.clauses, "datetime("+col+") "+equal+" datetime(?)")
		b.args = append(b.args, *arg)
	}
	return b
}

func (b *sqlClauseBuilder) addFloat(
	col string,
	arg *float64,
) *sqlClauseBuilder {
	if arg != nil {
		b.clauses = append(b.clauses, col+" = ?")
		b.args = append(b.args, *arg)
	}
	return b
}

func (b *sqlClauseBuilder) build(delimiter string) (string, []any) {
	return strings.Join(b.clauses, delimiter), b.args
}
