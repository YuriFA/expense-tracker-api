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

func (b *sqlClauseBuilder) addStringsForOr(
	cols []string,
	arg *string,
) *sqlClauseBuilder {
	if arg != nil {
		clauses := make([]string, len(cols))
		for i, col := range cols {
			clauses[i] = col + " = ?"
		}
		b.clauses = append(b.clauses, "("+strings.Join(clauses, " OR ")+")")
		for range cols {
			b.args = append(b.args, *arg)
		}
	}
	return b
}

func (b *sqlClauseBuilder) addTimeSet(
	col string,
	arg *time.Time,
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
