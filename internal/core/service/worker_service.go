package service

import(
	"fmt"
	"time"
	"context"
	"net/http"
	"encoding/json"
	"errors"

	"github.com/rs/zerolog/log"

	"github.com/go-worker-webhook/internal/adapter/database"
	"github.com/go-worker-webhook/internal/core/model"
	"github.com/go-worker-webhook/internal/core/erro"
	go_core_observ "github.com/eliezerraj/go-core/observability"
	go_core_api "github.com/eliezerraj/go-core/api"
)

var childLogger = log.With().Str("component","go-worker-webhook").Str("package","internal.core.service").Logger()
var tracerProvider go_core_observ.TracerProvider
var apiService go_core_api.ApiService

type WorkerService struct {
	goCoreRestApiService	go_core_api.ApiService
	workerRepository *database.WorkerRepository
}

// About create a new worker service
func NewWorkerService(	goCoreRestApiService	go_core_api.ApiService,	
						workerRepository *database.WorkerRepository ) *WorkerService{
	childLogger.Debug().Str("func","NewWorkerService").Send()

	return &WorkerService{
		goCoreRestApiService: goCoreRestApiService,
		workerRepository: workerRepository,
	}
}

// About handle/convert http status code
func errorStatusCode(statusCode int, serviceName string, msg_err error) error{
	childLogger.Info().Str("func","errorStatusCode").Interface("serviceName", serviceName).Interface("statusCode", statusCode).Send()
	var err error
	switch statusCode {
		case http.StatusUnauthorized:
			err = erro.ErrUnauthorized
		case http.StatusForbidden:
			err = erro.ErrHTTPForbiden
		case http.StatusNotFound:
			err = erro.ErrNotFound
		default:
			err = errors.New(fmt.Sprintf("service %s in outage => cause error: %s", serviceName, msg_err.Error() ))
		}
	return err
}

// About insert webhook
func (s *WorkerService) InsertWebHook(ctx context.Context, webhook *model.WebHook) (*model.WebHook, error){
	childLogger.Info().Str("func","InsertWebHook").Interface("trace-resquest-id", ctx.Value("trace-request-id")).Send()

	//Trace
	span := tracerProvider.Span(ctx, "service.InsertWebHook")
	trace_id := fmt.Sprintf("%v", ctx.Value("trace-request-id"))

	// Get the database connection
	tx, conn, err := s.workerRepository.DatabasePGServer.StartTx(ctx)
	if err != nil {
		return nil, err
	}
	defer s.workerRepository.DatabasePGServer.ReleaseTx(conn)

	// Handle the transaction
	defer func() {
		if err != nil {
			childLogger.Info().Interface("trace-request-id", trace_id ).Msg("ROLLBACK TX !!!")
			tx.Rollback(ctx)
		} else {
			childLogger.Info().Interface("trace-request-id", trace_id ).Msg("COMMIT TX !!!")
			tx.Commit(ctx)
		}	
		span.End()
	}()

	// ------------------------  STEP-1 ----------------------------------//
	childLogger.Info().Str("func","InsertWebHook").Msg("===> STEP - 01 (GET WEBHOOK CONFIG) <===")
	
	switch webhook.Type {
	case "TOPIC:PIX":
		pixTransaction := model.PixTransaction{}
		json.Unmarshal([]byte(webhook.Payload), &pixTransaction)

		childLogger.Info().Interface("payload:", pixTransaction).Send()

		webhook.Receiver = "ACCOUNT:" + pixTransaction.AccountFrom.AccountID
	default:
		childLogger.Info().Interface("topic:", webhook.Type).Msg("NOT REGISTER, MSG DISCARDED  !!!")
		return nil, nil
	}

	res, err := s.workerRepository.GetSetupWebHook(ctx, *webhook)
	if err != nil {
		childLogger.Info().Err(err).Send()
		webhook.Status = "IN-QUEUE:MSG-DISCARDED-NO-WEBHOOK-SETUP"
	} else {
		webhook.Status = "IN-QUEUE:WAITING-FOR-SEND"
		webhook.ID = res.ID
		webhook.Receiver = res.Receiver
		webhook.Host = res.Host
		webhook.Url = res.Url
		webhook.Method = res.Method
	}

	// ------------------------  STEP-2 ----------------------------------//
	childLogger.Info().Str("func","InsertWebHook").Msg("===> STEP - 02 (INSERT WEBHOOK) <===")

	//webhook.Status = "IN-QUEUE:WAITING-FOR-SEND"
	res, err = s.workerRepository.InsertWebHook(ctx, tx, *webhook)
	if err != nil {
		return nil, err
	}
	webhook.ID = res.ID	

	return webhook, nil
}

// About send the webhook
func (s *WorkerService) SendWebHook(ctx context.Context, webhook *model.WebHook) (*model.WebHook, error){
	childLogger.Info().Str("func","SendWebHook").Send()

	//Trace
	span := tracerProvider.Span(ctx, "service.SendWebHook")
	trace_id := fmt.Sprintf("%v", "11111-222222")

	// Get the database connection
	tx, conn, err := s.workerRepository.DatabasePGServer.StartTx(ctx)
	if err != nil {
		return nil, err
	}
	defer s.workerRepository.DatabasePGServer.ReleaseTx(conn)

	// Handle the transaction
	defer func() {
		if err != nil {
			childLogger.Info().Interface("trace-request-id", trace_id ).Msg("ROLLBACK TX !!!")
			tx.Rollback(ctx)
		} else {
			childLogger.Info().Interface("trace-request-id", trace_id ).Msg("COMMIT TX !!!")
			tx.Commit(ctx)
		}	
		span.End()
	}()

	// ------------------------  STEP-1 ----------------------------------//
	childLogger.Info().Str("func","SendWebHook").Msg("===> STEP - 02 (SEND WEBHOOK) <===")

	// prepare headers
	headers := map[string]string{
		"Content-Type":"application/json;charset=UTF-8",
	}

	httpClient := go_core_api.HttpClient {
		Url:	webhook.Host + webhook.Url,
		Method: webhook.Method,
		Timeout: 10,
		Headers: &headers,
	}

	_, statusCode, err := apiService.CallRestApiV1(	ctx,
													s.goCoreRestApiService.Client,
													httpClient, 
													webhook.Payload)
	if err != nil {
		childLogger.Error().Err(err).Interface("error",errorStatusCode(statusCode, webhook.Host, err)).Send()
	}

	// setting status
	if statusCode == 200 {
		webhook.Status = fmt.Sprintf("IN-QUEUE:SENDED:%v", statusCode)
	} else {
		webhook.Status = fmt.Sprintf("IN-QUEUE:ERROR:%v", statusCode)
	}
	
	// ------------------------  STEP-2 ----------------------------------//
	childLogger.Info().Str("func","SendWebHook").Msg("===> STEP - 03 (UPDATE) <===")

	update := time.Now()
	webhook.UpdatedAt = &update

	// update status payment
	res_update, err := s.workerRepository.UpdateWebHook(ctx, tx, *webhook)
	if err != nil {
		return nil, err
	}
	if res_update == 0 {
		err = erro.ErrUpdate
		return nil, err
	}

	return webhook, nil
}

// About get all webhook to send 
func (s *WorkerService) GetWebHook(ctx context.Context, webhook *model.WebHook) (*model.WebHook, error){
	childLogger.Debug().Str("func","GetWebHook").Send()
	
	webhook, err := s.workerRepository.GetWebHook(ctx, webhook)
	if err != nil {
		return nil, err
	}

	return webhook, nil
}