package database

import (
	"time"
	"context"
	"errors"
	
	"github.com/go-worker-webhook/internal/core/model"
	"github.com/go-worker-webhook/internal/core/erro"

	go_core_observ "github.com/eliezerraj/go-core/observability"
	go_core_pg "github.com/eliezerraj/go-core/database/pg"

	"github.com/jackc/pgx/v5"
	"github.com/rs/zerolog/log"
)

var childLogger = log.With().Str("component","go-worker-webhook").Str("package","internal.adapter.database").Logger()
var tracerProvider go_core_observ.TracerProvider

type WorkerRepository struct {
	DatabasePGServer *go_core_pg.DatabasePGServer
}

func NewWorkerRepository(databasePGServer *go_core_pg.DatabasePGServer) *WorkerRepository{
	childLogger.Info().Msg("NewWorkerRepository")

	return &WorkerRepository{
		DatabasePGServer: databasePGServer,
	}
}

func (w WorkerRepository) GetTransactionUUID(ctx context.Context) (*string, error){
	childLogger.Info().Interface("trace-resquest-id", ctx.Value("trace-request-id")).Msg("GetTransactionUUID")
	
	// Trace
	span := tracerProvider.Span(ctx, "database.GetTransactionUUID")
	defer span.End()

	// prepare database connection
	conn, err := w.DatabasePGServer.Acquire(ctx)
	if err != nil {
		return nil, errors.New(err.Error())
	}
	defer w.DatabasePGServer.Release(conn)

	// Prepare
	var uuid string

	// Query and Execute
	query := `SELECT uuid_generate_v4()`

	rows, err := conn.Query(ctx, query)
	if err != nil {
		return nil, errors.New(err.Error())
	}
	defer rows.Close()

	for rows.Next() {
		err := rows.Scan(&uuid) 
		if err != nil {
			return nil, errors.New(err.Error())
        }
		return &uuid, nil
	}
	
	return &uuid, nil
}

// About get webhook
func (w WorkerRepository) GetSetupWebHook(ctx context.Context, webhook model.WebHook) (*model.WebHook, error){
	childLogger.Info().Str("func","GetSetupWebHook").Interface("trace-resquest-id", ctx.Value("trace-request-id")).Send()

	span := tracerProvider.Span(ctx, "database.GetSetupWebHook")
	defer span.End()

	conn, err := w.DatabasePGServer.Acquire(ctx)
	if err != nil {
		return nil, errors.New(err.Error())
	}
	defer w.DatabasePGServer.Release(conn)

	res_webhook := model.WebHook{}

	query := `SELECT id,
					receiver,
					host,
					type,	 
					url,
					method,
					created_at,
					updated_at 
				FROM public.webhook_config 
				WHERE receiver = $1
				and	type = $2`

	rows, err := conn.Query(ctx, query, webhook.Receiver, webhook.Type)
	if err != nil {
		return nil, errors.New(err.Error())
	}
	defer rows.Close()

	for rows.Next() {
		err := rows.Scan( 	&res_webhook.ID,
							&res_webhook.Receiver, 
							&res_webhook.Host,
							&res_webhook.Type,  
							&res_webhook.Url, 
							&res_webhook.Method, 
							&res_webhook.CreatedAt,
							&res_webhook.UpdatedAt,)
		if err != nil {
			return nil, errors.New(err.Error())
        }
		return &res_webhook, nil
	}
	
	return nil, erro.ErrNotFound
}

// About get all webhook waiting for sending
func (w WorkerRepository) GetWebHook(ctx context.Context, webhook *model.WebHook) (*model.WebHook, error){
	childLogger.Info().Str("func","GetWebHook").Interface("trace-resquest-id", ctx.Value("trace-request-id")).Send()

	span := tracerProvider.Span(ctx, "database.GetWebHook")
	defer span.End()

	conn, err := w.DatabasePGServer.Acquire(ctx)
	if err != nil {
		return nil, errors.New(err.Error())
	}
	defer w.DatabasePGServer.Release(conn)

	res_webhook := model.WebHook{}

	query := `SELECT id,
					host,	 
					url,
					method,
					payload,
					status,
					created_at,
					updated_at 
				FROM public.webhook_transaction 
				WHERE status =$1
				order by created_at asc`

	rows, err := conn.Query(ctx, query, webhook.Status)
	if err != nil {
		return nil, errors.New(err.Error())
	}
	defer rows.Close()

	for rows.Next() {
		err := rows.Scan( 	&res_webhook.ID,
							&res_webhook.Host, 
							&res_webhook.Url, 
							&res_webhook.Method, 
							&res_webhook.Payload, 
							&res_webhook.Status, 
							&res_webhook.CreatedAt,
							&res_webhook.UpdatedAt,
						)
		if err != nil {
			return nil, errors.New(err.Error())
        }
		return &res_webhook, nil
	}
	
	return nil, erro.ErrNotFound
}

// About insert webhook
func (w *WorkerRepository) InsertWebHook(ctx context.Context, tx pgx.Tx, webHook model.WebHook) (*model.WebHook, error){
	childLogger.Info().Str("func","InsertWebHook").Interface("trace-resquest-id", ctx.Value("trace-request-id")).Send()

	// Trace
	span := tracerProvider.Span(ctx, "database.InsertWebHook")
	defer span.End()

	// Query and execute
	query := 	`INSERT INTO webhook_transaction (	receiver,
													host,
													url,
													method, 
													payload,
													status,
													created_at) 
				VALUES($1, $2, $3, $4, $5, $6, $7) RETURNING id`
	row	:= tx.QueryRow(	ctx,
						query,
						webHook.Receiver,
						webHook.Host,
						webHook.Url,
						webHook.Method,
						webHook.Payload,
						webHook.Status,
						time.Now())
	var id int
	
	if err := row.Scan(&id); err != nil {
		return nil, errors.New(err.Error())
	}

	webHook.ID = id

	return &webHook, nil
}

// About update webhook
func (w *WorkerRepository) UpdateWebHook(ctx context.Context, tx pgx.Tx, webHook model.WebHook) (int64, error){
	childLogger.Info().Str("func","UpdateWebHook").Interface("trace-resquest-id", ctx.Value("trace-request-id")).Send()

	// Trace
	span := tracerProvider.Span(ctx, "database.UpdateWebHook")
	defer span.End()

	// Query and execute
	query := `UPDATE webhook_transaction
				SET status = $2,
					updated_at = $3
				WHERE id = $1`

	row, err := tx.Exec(ctx, 
						query,	
						webHook.ID,
						webHook.Status,
						time.Now())
	if err != nil {
		return 0, errors.New(err.Error())
	}
	return row.RowsAffected(), nil
}