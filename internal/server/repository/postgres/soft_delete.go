package postgres

import (
	sq "github.com/Masterminds/squirrel"
)

// applySoftDeleteFilter adds deleted_at IS NULL condition to a SelectBuilder
// This ensures that soft-deleted records are excluded from queries
func applySoftDeleteFilter(builder sq.SelectBuilder) sq.SelectBuilder {
	return builder.Where(sq.Expr("deleted_at IS NULL"))
}

// applySoftDeleteFilterToUpdate adds deleted_at IS NULL condition to an UpdateBuilder
// This ensures that soft-deleted records are excluded from update operations
func applySoftDeleteFilterToUpdate(builder sq.UpdateBuilder) sq.UpdateBuilder {
	return builder.Where(sq.Expr("deleted_at IS NULL"))
}
