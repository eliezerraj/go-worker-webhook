package model

import (
	"time"
	go_core_pg "github.com/eliezerraj/go-core/database/pg"
	go_core_observ "github.com/eliezerraj/go-core/observability"
	go_core_event "github.com/eliezerraj/go-core/event/kafka" 
)

type AppServer struct {
	InfoPod 			*InfoPod 					`json:"info_pod"`
	ConfigOTEL			*go_core_observ.ConfigOTEL	`json:"otel_config"`
	DatabaseConfig		*go_core_pg.DatabaseConfig  `json:"database"`
	KafkaConfigurations	*go_core_event.KafkaConfigurations  `json:"kafka_configurations"`
	Topics 				[]string					`json:"topics"`	
}

type InfoPod struct {
	PodName				string 	`json:"pod_name"`
	ApiVersion			string 	`json:"version"`
	OSPID				string 	`json:"os_pid"`
	IPAddress			string 	`json:"ip_address"`
	AvailabilityZone 	string 	`json:"availabilityZone"`
	IsAZ				bool   	`json:"is_az"`
	Env					string `json:"enviroment,omitempty"`
	AccountID			string `json:"account_id,omitempty"`
}

type StepProcess struct {
	Name		string  	`json:"step_process,omitempty"`
	ProcessedAt	time.Time 	`json:"processed_at,omitempty"`
}

type WebHook struct {
	ID				int			`json:"id,omitempty"`
	Receiver		string 		`json:"receiver,omitempty"`
	Host			string 		`json:"host,omitempty"`
	Url				string 		`json:"url,omitempty"`
	Method			string 		`json:"method,omitempty"`
	Topic			string 		`json:"topic,omitempty"`
	Type			string 		`json:"type,omitempty"`	
	Payload			[]byte	 	`json:"payload,omitempty"`
	Status			string  	`json:"status,omitempty"`
	CreatedAt		time.Time 	`json:"created_at,omitempty"`
	UpdatedAt		*time.Time 	`json:"updated_at,omitempty"`
}

type Account struct {
	ID				int			`json:"id,omitempty"`
	AccountID		string		`json:"account_id,omitempty"`
	PersonID		string  	`json:"person_id,omitempty"`
}

type PixTransaction struct {
	ID				int			`json:"id,omitempty"`
	TransactionId	string 		`json:"transaction_id,omitempty"`
	TransactionAt	time.Time 	`json:"transaction_at,omitempty"`	
	RequestId		string 		`json:"request_id,omitempty"`
	AccountFrom		Account		`json:"account_from,omitempty"`
	AccountTo		Account		`json:"account_to,omitempty"`
	Status			string  	`json:"status,omitempty"`
	Currency		string 		`json:"currency,omitempty"`
	Amount			float64 	`json:"amount,omitempty"`	
	StepProcess		*[]StepProcess	`json:"step_process,omitempty"`
	CreatedAt		time.Time 	`json:"created_at,omitempty"`
	UpdatedAt		*time.Time 	`json:"updated_at,omitempty"`
}