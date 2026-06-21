package sqlite

import "strings"

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

func (b *sqlClauseBuilder) addStringOp(
	col string,
	arg *string,
	equal string,
) *sqlClauseBuilder {
	if arg != nil {
		b.clauses = append(b.clauses, col+" "+equal+" ?")
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
