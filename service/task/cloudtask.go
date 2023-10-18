package task

import (
	"context"
	"fmt"
	"github.com/SplitFi/go-splitfi/tokenprocessing"

	"encoding/json"
	"time"

	gcptasks "cloud.google.com/go/cloudtasks/apiv2"
	"github.com/SplitFi/go-splitfi/env"
	"github.com/SplitFi/go-splitfi/service/logger"
	"github.com/SplitFi/go-splitfi/service/persist"
	"github.com/SplitFi/go-splitfi/service/tracing"
	"github.com/SplitFi/go-splitfi/util"
	"google.golang.org/api/option"
	taskspb "google.golang.org/genproto/googleapis/cloud/tasks/v2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type TokenProcessingUserMessage struct {
	OwnerAddress     persist.Address             `json:"owner_address" binding:"required"`
	TokenIdentifiers []persist.TokenChainAddress `json:"token_identifiers" binding:"required"`
}

type TokenIdentifiersQuantities map[persist.TokenChainAddress]persist.HexString

func (t TokenIdentifiersQuantities) MarshalJSON() ([]byte, error) {
	m := make(map[string]string)
	for k, v := range t {
		m[k.String()] = v.String()
	}
	return json.Marshal(m)
}

func (t *TokenIdentifiersQuantities) UnmarshalJSON(b []byte) error {
	m := make(map[string]string)
	if err := json.Unmarshal(b, &m); err != nil {
		return err
	}
	result := make(TokenIdentifiersQuantities)
	for k, v := range m {
		identifiers, err := persist.TokenChainAddressFromString(k)
		if err != nil {
			return err
		}
		result[identifiers] = persist.HexString(v)
	}
	*t = result
	return nil
}

type TokenProcessingAssetsMessage struct {
	OwnerAddress persist.Address            `json:"owner_address" binding:"required"`
	Balances     TokenIdentifiersQuantities `json:"token_identifiers" binding:"required"`
}

// DeepRefreshMessage is the input message to the indexer-api for deep refreshes
type DeepRefreshMessage struct {
	OwnerAddress    persist.EthereumAddress `json:"owner_address"`
	ContractAddress persist.EthereumAddress `json:"contract_address"`
	RefreshRange    persist.BlockRange      `json:"refresh_range"`
}

type ValidateNFTsMessage struct {
	OwnerAddress persist.EthereumAddress `json:"wallet"`
}

func CreateTaskForTokenProcessing(ctx context.Context, client *gcptasks.Client, message TokenProcessingUserMessage) error {
	span, ctx := tracing.StartSpan(ctx, "cloudtask.create", "createTaskForTokenProcessing")
	defer tracing.FinishSpan(span)

	tracing.AddEventDataToSpan(span, map[string]interface{}{"User ID": message.OwnerAddress})

	queue := env.GetString("TOKEN_PROCESSING_QUEUE")
	task := &taskspb.Task{
		MessageType: &taskspb.Task_HttpRequest{
			HttpRequest: &taskspb.HttpRequest{
				HttpMethod: taskspb.HttpMethod_POST,
				Url:        fmt.Sprintf("%s%s", env.GetString("TOKEN_PROCESSING_URL"), tokenprocessing.ProcessMediaForUsersTokensOfChainPath),
				Headers: map[string]string{
					"Content-type": "application/json",
					"sentry-trace": span.TraceID.String(),
				},
			},
		},
	}

	body, err := json.Marshal(message)
	if err != nil {
		return err
	}

	return submitHttpTask(ctx, client, queue, task, body)
}

func CreateTaskForDeepRefresh(ctx context.Context, message DeepRefreshMessage, client *gcptasks.Client) error {
	span, ctx := tracing.StartSpan(ctx, "cloudtask.create", "createTaskForDeepRefresh")
	defer tracing.FinishSpan(span)

	queue := env.GetString("GCLOUD_REFRESH_TASK_QUEUE")
	task := &taskspb.Task{
		MessageType: &taskspb.Task_HttpRequest{
			HttpRequest: &taskspb.HttpRequest{
				HttpMethod: taskspb.HttpMethod_POST,
				Url:        fmt.Sprintf("%s/tasks/refresh", env.GetString("INDEXER_HOST")),
				Headers: map[string]string{
					"Content-type": "application/json",
					"sentry-trace": span.TraceID.String(),
				},
			},
		},
	}

	body, err := json.Marshal(message)
	if err != nil {
		return err
	}

	return submitHttpTask(ctx, client, queue, task, body)
}

func CreateTaskForWalletValidation(ctx context.Context, message ValidateNFTsMessage, client *gcptasks.Client) error {
	span, ctx := tracing.StartSpan(ctx, "cloudtask.create", "createTaskForWalletValidate")
	defer tracing.FinishSpan(span)

	queue := env.GetString("GCLOUD_WALLET_VALIDATE_QUEUE")
	task := &taskspb.Task{
		MessageType: &taskspb.Task_HttpRequest{
			HttpRequest: &taskspb.HttpRequest{
				HttpMethod: taskspb.HttpMethod_POST,
				Url:        fmt.Sprintf("%s/nfts/validate", env.GetString("INDEXER_HOST")),
				Headers: map[string]string{
					"Content-type": "application/json",
					"sentry-trace": span.TraceID.String(),
				},
			},
		},
	}

	body, err := json.Marshal(message)
	if err != nil {
		return err
	}

	return submitHttpTask(ctx, client, queue, task, body)
}

func CreateTaskForAssetProcessing(ctx context.Context, message TokenProcessingAssetsMessage, client *gcptasks.Client) error {
	span, ctx := tracing.StartSpan(ctx, "cloudtask.create", "createTaskForTokenProcessingUser")
	defer tracing.FinishSpan(span)

	tracing.AddEventDataToSpan(span, map[string]interface{}{
		"Owner Address": message.OwnerAddress,
	})

	queue := env.GetString("TOKEN_PROCESSING_QUEUE")

	task := &taskspb.Task{
		MessageType: &taskspb.Task_HttpRequest{
			HttpRequest: &taskspb.HttpRequest{
				HttpMethod: taskspb.HttpMethod_POST,
				Url:        fmt.Sprintf("%s%s", env.GetString("TOKEN_PROCESSING_URL"), tokenprocessing.ProcessAssetsForOwnerPath),
				Headers: map[string]string{
					"Content-type": "application/json",
					"sentry-trace": span.TraceID.String(),
				},
			},
		},
	}

	body, err := json.Marshal(message)
	if err != nil {
		return err
	}

	return submitHttpTask(ctx, client, queue, task, body)
}

/*
func CreateTaskForWalletRemoval(ctx context.Context, message TokenProcessingWalletRemovalMessage, client *gcptasks.Client) error {
	span, ctx := tracing.StartSpan(ctx, "cloudtask.create", "createTaskForWalletRemoval")
	defer tracing.FinishSpan(span)

	tracing.AddEventDataToSpan(span, map[string]interface{}{
		"User ID":    message.UserID,
		"Wallet IDs": message.WalletIDs,
	})

	queue := env.GetString("TOKEN_PROCESSING_QUEUE")

	task := &taskspb.Task{
		MessageType: &taskspb.Task_HttpRequest{
			HttpRequest: &taskspb.HttpRequest{
				HttpMethod: taskspb.HttpMethod_POST,
				Url:        fmt.Sprintf("%s/owners/process/wallet-removal", env.GetString("TOKEN_PROCESSING_URL")),
				Headers: map[string]string{
					"Content-type": "application/json",
					"sentry-trace": span.TraceID.String(),
				},
			},
		},
	}

	body, err := json.Marshal(message)
	if err != nil {
		return err
	}

	return submitHttpTask(ctx, client, queue, task, body)
}

func CreateTaskForAddingEmailToMailingList(ctx context.Context, message AddEmailToMailingListMessage, client *gcptasks.Client) error {
	span, ctx := tracing.StartSpan(ctx, "cloudtask.create", "createTaskForAddingEmailToMailingList")
	defer tracing.FinishSpan(span)

	tracing.AddEventDataToSpan(span, map[string]interface{}{"User ID": message.UserID})

	queue := env.GetString("EMAILS_QUEUE")

	task := &taskspb.Task{
		MessageType: &taskspb.Task_HttpRequest{
			HttpRequest: &taskspb.HttpRequest{
				HttpMethod: taskspb.HttpMethod_POST,
				Url:        fmt.Sprintf("%s/send/process/add-to-mailing-list", env.GetString("EMAILS_HOST")),
				Headers: map[string]string{
					"Content-type":  "application/json",
					"Authorization": basicauth.MakeHeader(nil, env.GetString("EMAILS_TASK_SECRET")),
					"sentry-trace":  span.TraceID.String(),
				},
			},
		},
	}

	body, err := json.Marshal(message)
	if err != nil {
		return err
	}

	return submitHttpTask(ctx, client, queue, task, body)
}
*/

// NewClient returns a new task client with tracing enabled.
func NewClient(ctx context.Context) *gcptasks.Client {
	trace := tracing.NewTracingInterceptor(true)

	copts := []option.ClientOption{
		option.WithGRPCDialOption(grpc.WithUnaryInterceptor(trace.UnaryInterceptor)),
		option.WithGRPCDialOption(grpc.WithTimeout(time.Duration(2) * time.Second)),
	}

	// Configure the client depending on whether or not
	// the cloud task emulator is used.
	if env.GetString("ENV") == "local" {
		if host := env.GetString("TASK_QUEUE_HOST"); host != "" {
			copts = append(
				copts,
				option.WithEndpoint(host),
				option.WithGRPCDialOption(grpc.WithTransportCredentials(insecure.NewCredentials())),
				option.WithoutAuthentication(),
			)
		} else {
			fi, err := util.LoadEncryptedServiceKeyOrError("./secrets/dev/service-key-dev.json")
			if err != nil {
				logger.For(ctx).WithError(err).Error("failed to find service key, running without task client")
				return nil
			}
			copts = append(
				copts,
				option.WithCredentialsJSON(fi),
			)
		}
	}

	client, err := gcptasks.NewClient(ctx, copts...)
	if err != nil {
		panic(err)
	}

	return client
}

func submitHttpTask(ctx context.Context, client *gcptasks.Client, queue string, task *taskspb.Task, messageBody []byte) error {
	req := &taskspb.CreateTaskRequest{Parent: queue, Task: task}
	req.Task.GetHttpRequest().Body = messageBody
	_, err := client.CreateTask(ctx, req)
	return err
}
