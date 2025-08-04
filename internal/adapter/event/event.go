package event

import (
	"context"

	go_core_observ "github.com/eliezerraj/go-core/observability"
	go_core_event "github.com/eliezerraj/go-core/event/kafka"

	"github.com/rs/zerolog/log"
)

var childLogger = log.With().Str("component","go-worker-webhook").Str("package","internal.adapter.event").Logger()

var tracerProvider go_core_observ.TracerProvider
var consumerWorker go_core_event.ConsumerWorker

type WorkerEvent struct {
	Topics	[]string
	WorkerKafka *go_core_event.ConsumerWorker 
}

func NewWorkerEvent(ctx context.Context, topics []string, kafkaConfigurations *go_core_event.KafkaConfigurations) (*WorkerEvent, error) {
	childLogger.Info().Str("func","NewWorkerEvent").Send()

	//trace
	span := tracerProvider.Span(ctx, "adapter.event.NewWorkerEvent")
	defer span.End()

	workerKafka, err := consumerWorker.NewConsumerWorker(kafkaConfigurations)
	if err != nil {
		return nil, err
	}

	return &WorkerEvent{
		Topics: topics,
		WorkerKafka: workerKafka,
	},nil
}