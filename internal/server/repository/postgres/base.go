package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/BigSm0uk/GophKeeper/internal/server/app/db"
	pgerrors "github.com/BigSm0uk/GophKeeper/internal/server/app/db/pgerror"
	"github.com/BigSm0uk/GophKeeper/internal/server/domain/interfaces"
	sq "github.com/Masterminds/squirrel"
	"github.com/avast/retry-go"
	"github.com/jackc/pgx/v5"
	"go.uber.org/zap"
)

type Scanner[T interfaces.Entity] func(row pgx.Row) (T, error)

type BaseRepository[T interfaces.Entity] struct {
	db              *db.PostgresDb
	logger          *zap.Logger
	errorClassifier *pgerrors.PostgresErrorClassifier
	scanner         Scanner[T]
	tableName       string
}

var _ interfaces.BaseRepository[interfaces.Entity] = (*BaseRepository[interfaces.Entity])(nil)

func NewBaseRepository[T interfaces.Entity](
	db *db.PostgresDb,
	logger *zap.Logger,
	tableName string,
	scanner Scanner[T],
) *BaseRepository[T] {
	return &BaseRepository[T]{
		db:              db,
		logger:          logger,
		errorClassifier: pgerrors.NewPostgresErrorClassifier(),
		scanner:         scanner,
		tableName:       tableName,
	}
}

// Delete implements [interfaces.BaseRepository].
func (p *BaseRepository[T]) Delete(ctx context.Context, id string) error {
	if id == "" {
		return pgx.ErrNoRows
	}

	query, args, err := sq.
		Update(p.tableName).
		Set("deleted_at", time.Now()).
		Where(sq.Eq{"id": id}).
		PlaceholderFormat(sq.Dollar).
		ToSql()
	if err != nil {
		p.logger.Error("Failed to build delete query",
			zap.Error(err),
			zap.String("table", p.tableName),
			zap.String("id", id))
		return err
	}

	err = retry.Do(
		func() error {
			conn, err := p.db.GetPool().Acquire(ctx)
			if err != nil {
				p.logger.Error("Failed to acquire database connection",
					zap.Error(err),
					zap.String("table", p.tableName))
				return err
			}
			defer conn.Release()

			tag, err := conn.Exec(ctx, query, args...)
			if err != nil {
				return err
			}

			if tag.RowsAffected() == 0 {
				return pgx.ErrNoRows
			}

			return nil
		},
		retry.Attempts(3),
		retry.Delay(100*time.Millisecond),
		retry.MaxDelay(1*time.Second),
		retry.RetryIf(func(err error) bool {
			if errors.Is(err, pgx.ErrNoRows) {
				return false
			}
			return p.errorClassifier.Classify(err) == pgerrors.Retriable
		}),
		retry.Context(ctx),
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			p.logger.Info("Entity not found for delete",
				zap.String("table", p.tableName),
				zap.String("id", id))
			return err
		}

		p.logger.Error("Failed to delete entity",
			zap.Error(err),
			zap.String("table", p.tableName),
			zap.String("id", id))
		return err
	}

	return nil
}

// Exists implements [interfaces.BaseRepository].
func (p *BaseRepository[T]) Exists(ctx context.Context, id string) (bool, error) {
	if id == "" {
		return false, nil
	}

	query, args, err := applySoftDeleteFilter(
		sq.Select("COUNT(*) > 0").
			From(p.tableName).
			Where(sq.Eq{"id": id}).
			Limit(1),
	).
		PlaceholderFormat(sq.Dollar).
		ToSql()
	if err != nil {
		p.logger.Error("Failed to build exists query",
			zap.Error(err),
			zap.String("table", p.tableName),
			zap.String("id", id))
		return false, err
	}

	var exists int

	err = retry.Do(
		func() error {
			conn, err := p.db.GetPool().Acquire(ctx)
			if err != nil {
				return err
			}
			defer conn.Release()

			return conn.QueryRow(ctx, query, args...).Scan(&exists)
		},
		retry.Attempts(3),
		retry.Delay(100*time.Millisecond),
		retry.MaxDelay(1*time.Second),
		retry.RetryIf(func(err error) bool {
			if errors.Is(err, pgx.ErrNoRows) {
				return false
			}
			return p.errorClassifier.Classify(err) == pgerrors.Retriable
		}),
		retry.Context(ctx),
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, nil
		}
		return false, err
	}

	return true, nil
}

// ExistsByUserID implements [interfaces.BaseRepository].
func (p *BaseRepository[T]) ExistsByUserID(ctx context.Context, userID string) (bool, error) {
	if userID == "" {
		return false, nil
	}

	query, args, err := applySoftDeleteFilter(
		sq.Select("COUNT(*) > 0").
			From(p.tableName).
			Where(sq.Eq{"user_id": userID}).
			Limit(1),
	).
		PlaceholderFormat(sq.Dollar).
		ToSql()
	if err != nil {
		p.logger.Error("Failed to build exists by user GetID query",
			zap.Error(err),
			zap.String("table", p.tableName),
			zap.String("user_id", userID))
		return false, err
	}

	var exists int

	err = retry.Do(
		func() error {
			conn, err := p.db.GetPool().Acquire(ctx)
			if err != nil {
				return err
			}
			defer conn.Release()

			return conn.QueryRow(ctx, query, args...).Scan(&exists)
		},
		retry.Attempts(3),
		retry.Delay(100*time.Millisecond),
		retry.MaxDelay(1*time.Second),
		retry.RetryIf(func(err error) bool {
			if errors.Is(err, pgx.ErrNoRows) {
				return false
			}
			return p.errorClassifier.Classify(err) == pgerrors.Retriable
		}),
		retry.Context(ctx),
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, nil
		}
		return false, err
	}

	return true, nil
}

// FindByID implements [interfaces.BaseRepository].
func (p *BaseRepository[T]) FindByID(ctx context.Context, id string) (T, error) {
	var zero T
	if id == "" {
		return zero, pgx.ErrNoRows
	}

	query, args, err := applySoftDeleteFilter(
		sq.Select("*").
			From(p.tableName).
			Where(sq.Eq{"id": id}),
	).
		PlaceholderFormat(sq.Dollar).
		ToSql()
	if err != nil {
		p.logger.Error("Failed to build find entity by GetID query",
			zap.Error(err),
			zap.String("table", p.tableName),
			zap.String("id", id))
		return zero, err
	}

	var result T

	err = retry.Do(
		func() error {
			conn, err := p.db.GetPool().Acquire(ctx)
			if err != nil {
				p.logger.Error("Failed to acquire database connection",
					zap.Error(err),
					zap.String("table", p.tableName))
				return err
			}
			defer conn.Release()

			row := conn.QueryRow(ctx, query, args...)
			entity, err := p.scanner(row)
			if err != nil {
				return err
			}
			result = entity
			return nil
		},
		retry.Attempts(3),
		retry.Delay(100*time.Millisecond),
		retry.MaxDelay(1*time.Second),
		retry.RetryIf(func(err error) bool {
			if errors.Is(err, pgx.ErrNoRows) {
				return false
			}
			return p.errorClassifier.Classify(err) == pgerrors.Retriable
		}),
		retry.OnRetry(func(n uint, err error) {
			p.logger.Info("Retrying database operation",
				zap.Uint("attempt", n+1),
				zap.Error(err),
				zap.String("table", p.tableName),
				zap.String("id", id))
		}),
		retry.Context(ctx),
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			p.logger.Info("Entity not found",
				zap.String("table", p.tableName),
				zap.String("id", id))
			return zero, err
		}

		p.logger.Error("Failed to find entity after retries",
			zap.Error(err),
			zap.String("table", p.tableName),
			zap.String("id", id))
		return zero, err
	}

	return result, nil
}

// FindByUserID implements [interfaces.BaseRepository].
func (p *BaseRepository[T]) FindByUserID(ctx context.Context, userID string, limit, offset int) ([]T, error) {
	if userID == "" {
		return nil, nil
	}

	selectBuilder := applySoftDeleteFilter(
		sq.Select("*").
			From(p.tableName).
			Where(sq.Eq{"user_id": userID}),
	)

	if limit > 0 {
		selectBuilder = selectBuilder.Limit(uint64(limit))
	}
	if offset > 0 {
		selectBuilder = selectBuilder.Offset(uint64(offset))
	}

	query, args, err := selectBuilder.PlaceholderFormat(sq.Dollar).ToSql()
	if err != nil {
		p.logger.Error("Failed to build find by user ID query",
			zap.Error(err),
			zap.String("table", p.tableName),
			zap.String("user_id", userID))
		return nil, err
	}

	var result []T

	err = retry.Do(
		func() error {
			conn, err := p.db.GetPool().Acquire(ctx)
			if err != nil {
				p.logger.Error("Failed to acquire database connection",
					zap.Error(err),
					zap.String("table", p.tableName))
				return err
			}
			defer conn.Release()

			rows, err := conn.Query(ctx, query, args...)
			if err != nil {
				return err
			}
			defer rows.Close()

			result = result[:0]

			for rows.Next() {
				entity, err := p.scanner(rows)
				if err != nil {
					return err
				}
				result = append(result, entity)
			}

			return rows.Err()
		},
		retry.Attempts(3),
		retry.Delay(100*time.Millisecond),
		retry.MaxDelay(1*time.Second),
		retry.RetryIf(func(err error) bool {
			return p.errorClassifier.Classify(err) == pgerrors.Retriable
		}),
		retry.Context(ctx),
	)
	if err != nil {
		p.logger.Error("Failed to find entities by user ID",
			zap.Error(err),
			zap.String("table", p.tableName),
			zap.String("user_id", userID))
		return nil, err
	}

	return result, nil
}

// CountByUserID implements [interfaces.BaseRepository].
func (p *BaseRepository[T]) CountByUserID(ctx context.Context, userID string) (int64, error) {
	if userID == "" {
		return 0, nil
	}

	query, args, err := applySoftDeleteFilter(
		sq.Select("COUNT(*)").
			From(p.tableName).
			Where(sq.Eq{"user_id": userID}),
	).
		PlaceholderFormat(sq.Dollar).
		ToSql()
	if err != nil {
		p.logger.Error("Failed to build count by user ID query",
			zap.Error(err),
			zap.String("table", p.tableName),
			zap.String("user_id", userID))
		return 0, err
	}

	var count int64

	err = retry.Do(
		func() error {
			conn, err := p.db.GetPool().Acquire(ctx)
			if err != nil {
				p.logger.Error("Failed to acquire database connection",
					zap.Error(err),
					zap.String("table", p.tableName))
				return err
			}
			defer conn.Release()

			return conn.QueryRow(ctx, query, args...).Scan(&count)
		},
		retry.Attempts(3),
		retry.Delay(100*time.Millisecond),
		retry.MaxDelay(1*time.Second),
		retry.RetryIf(func(err error) bool {
			return p.errorClassifier.Classify(err) == pgerrors.Retriable
		}),
		retry.Context(ctx),
	)
	if err != nil {
		p.logger.Error("Failed to count entities by user ID",
			zap.Error(err),
			zap.String("table", p.tableName),
			zap.String("user_id", userID))
		return 0, err
	}

	return count, nil
}
