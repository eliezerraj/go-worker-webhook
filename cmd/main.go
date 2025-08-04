package main

import(
	"time"
	"os"
	"os/signal"
	"syscall"
	"context"
	"sync"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/go-worker-webhook/internal/infra/configuration"
	"github.com/go-worker-webhook/internal/core/model"
	"github.com/go-worker-webhook/internal/core/service"
	"github.com/go-worker-webhook/internal/adapter/database"
	"github.com/go-worker-webhook/internal/adapter/event"
	"github.com/go-worker-webhook/internal/infra/server"

	go_core_api "github.com/eliezerraj/go-core/api"
	go_core_pg "github.com/eliezerraj/go-core/database/pg"  
)

var(
	appServer			model.AppServer
	databaseConfig 		go_core_pg.DatabaseConfig
	databasePGServer 	go_core_pg.DatabasePGServer
	
	logLevel = zerolog.InfoLevel // zerolog.InfoLevel zerolog.DebugLevel
	childLogger = log.With().Str("component","go-worker-webhook").Str("package", "main").Logger()
)

func init(){
	childLogger.Info().Str("func","init").Send()

	zerolog.SetGlobalLevel(logLevel)

	infoPod 		:= configuration.GetInfoPod()
	configOTEL 		:= configuration.GetOtelEnv()
	databaseConfig 	:= configuration.GetDatabaseEnv() 
	kafkaConfigurations, topics := configuration.GetKafkaEnv() 

	appServer.InfoPod = &infoPod
	appServer.ConfigOTEL = &configOTEL
	appServer.DatabaseConfig = &databaseConfig
	appServer.KafkaConfigurations = &kafkaConfigurations
	appServer.Topics = topics
}

func main()  {
	childLogger.Info().Str("func","main").Interface("appServer",appServer).Send()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGTERM, syscall.SIGINT)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stop()
	// Open Database
	count := 1
	var err error
	for {
		databasePGServer, err = databasePGServer.NewDatabasePGServer(ctx, *appServer.DatabaseConfig)
		if err != nil {
			if count < 3 {
				childLogger.Error().Err(err).Msg("error open database... trying again !!")
			} else {
				childLogger.Error().Err(err).Msg("fatal error open Database aborting")
				panic(err)
			}
			time.Sleep(3 * time.Second) //backoff
			count = count + 1
			continue
		}
		break
	}
		
	// Database
	database := database.NewWorkerRepository(&databasePGServer)

	// Create a go-core api service for client http
	coreRestApiService := go_core_api.NewRestApiService()
	workerService := service.NewWorkerService(*coreRestApiService, database)
	
	// Kafka
	workerEvent, err := event.NewWorkerEvent(ctx, 
											appServer.Topics, 
											appServer.KafkaConfigurations)
	if err != nil {
		childLogger.Error().Err(err).Msg("error open kafka")
		panic(err)
	}

	serverWorker := server.NewServerWorker(workerService, workerEvent)

	var wg, wg_webhook sync.WaitGroup

	wg.Add(1)
	go serverWorker.Consumer(ctx, &appServer, &wg)

	wg_webhook.Add(1)
	go serverWorker.SendWebhook(ctx, &appServer, &wg_webhook)
	
	wg.Wait()
	wg_webhook.Wait()
}