package task

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/SplitFi/go-splitfi/service/auth/basicauth"
	"net/http"
	"time"

	gcptasks "cloud.google.com/go/cloudtasks/apiv2"
	"github.com/SplitFi/go-splitfi/env"
	"github.com/SplitFi/go-splitfi/service/logger"
	"github.com/SplitFi/go-splitfi/service/persist"
	"github.com/SplitFi/go-splitfi/service/tracing"
	"github.com/SplitFi/go-splitfi/util"
	"github.com/getsentry/sentry-go"
	"google.golang.org/api/option"
	taskspb "google.golang.org/genproto/googleapis/cloud/tasks/v2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Client struct {
	skipQueues map[string]bool
	sendFunc   func(ctx context.Context, queue string, task *taskspb.Task) error
}

type TokenTransferProcessingMessage struct {
	Transfers []TokenTransfer `json:"transfers" binding:"required"`
}

type AddEmailToMailingListMessage struct {
	UserID persist.DBID `json:"user_id" binding:"required"`
}

type TokenTransfer struct {
	FromAddress persist.Address           `json:"from_address"`
	ToAddress   persist.Address           `json:"to_address"`
	Token       persist.TokenChainAddress `json:"token"`
	Amount      persist.HexString         `json:"amount"`
}

type TokenProcessingWalletRemovalMessage struct {
	UserID    persist.DBID   `json:"user_id" binding:"required"`
	WalletIDs []persist.DBID `json:"wallet_ids" binding:"required"`
}

type ValidateNFTsMessage struct {
	OwnerAddress persist.EthereumAddress `json:"wallet"`
}

type PushNotificationMessage struct {
	PushTokenID persist.DBID   `json:"pushTokenID"`
	Title       string         `json:"title"`
	Subtitle    string         `json:"subtitle"`
	Body        string         `json:"body"`
	Data        map[string]any `json:"data"`
	Sound       bool           `json:"sound"`
	Badge       int            `json:"badge"`
}

func (c *Client) CreateTaskForPushNotification(ctx context.Context, message PushNotificationMessage) error {
	span, ctx := tracing.StartSpan(ctx, "cloudtask.create", "createTaskForPushNotification")
	defer tracing.FinishSpan(span)
	tracing.AddEventDataToSpan(span, map[string]any{"PushTokenID": message.PushTokenID})
	queue := env.GetString("GCLOUD_PUSH_NOTIFICATIONS_QUEUE")
	url := fmt.Sprintf("%s/tasks/send-push-notification", env.GetString("PUSH_NOTIFICATIONS_URL"))
	secret := env.GetString("PUSH_NOTIFICATIONS_SECRET")
	return c.submitTask(ctx, queue, url, withJSON(message), withTrace(span), withBasicAuth(secret))
}

func (c *Client) CreateTaskForTokenTransferProcessing(ctx context.Context, message TokenTransferProcessingMessage) error {
	span, ctx := tracing.StartSpan(ctx, "cloudtask.create", "CreateTaskForTokenTransferProcessing")
	defer tracing.FinishSpan(span)
	queue := env.GetString("TOKEN_PROCESSING_QUEUE")
	url := fmt.Sprintf("%s/token/transfer", env.GetString("TOKEN_PROCESSING_URL"))
	return c.submitTask(ctx, queue, url, withJSON(message), withTrace(span))
}

func (c *Client) CreateTaskForWalletRemoval(ctx context.Context, message TokenProcessingWalletRemovalMessage) error {
	span, ctx := tracing.StartSpan(ctx, "cloudtask.create", "createTaskForWalletRemoval")
	defer tracing.FinishSpan(span)
	tracing.AddEventDataToSpan(span, map[string]any{"User ID": message.UserID, "Wallet IDs": message.WalletIDs})
	queue := env.GetString("TOKEN_PROCESSING_QUEUE")
	url := fmt.Sprintf("%s/owner/wallet-removal", env.GetString("TOKEN_PROCESSING_URL"))
	return c.submitTask(ctx, queue, url, withJSON(message), withTrace(span))
}

func (c *Client) CreateTaskForAddingEmailToMailingList(ctx context.Context, message AddEmailToMailingListMessage) error {
	span, ctx := tracing.StartSpan(ctx, "cloudtask.create", "createTaskForAddingEmailToMailingList")
	defer tracing.FinishSpan(span)
	tracing.AddEventDataToSpan(span, map[string]any{"User ID": message.UserID})
	queue := env.GetString("EMAILS_QUEUE")
	url := fmt.Sprintf("%s/send/process/add-to-mailing-list", env.GetString("EMAILS_HOST"))
	secret := env.GetString("EMAILS_TASK_SECRET")
	return c.submitTask(ctx, queue, url, withJSON(message), withTrace(span), withBasicAuth(secret))
}

// NewClient returns a new task client with tracing enabled.
func NewClient(ctx context.Context) *Client {
	skipQueues := make(map[string]bool)
	for _, q := range env.GetStringSlice("CLOUD_TASKS_SKIP_QUEUES") {
		skipQueues[q] = true
	}

	if env.GetBool("CLOUD_TASKS_DIRECT_DISPATCH_ENABLED") {
		return &Client{skipQueues: skipQueues, sendFunc: useDirectDispatch(ctx)}
	} else {
		return &Client{skipQueues: skipQueues, sendFunc: useCloudTasks(ctx, newGCPClient(ctx))}
	}
}

func useDirectDispatch(ctx context.Context) func(ctx context.Context, queue string, task *taskspb.Task) error {
	logger.For(ctx).Info("Initializing task client with direct dispatch")
	httpClient := &http.Client{}
	return func(ctx context.Context, queue string, task *taskspb.Task) error {
		go func() {
			// Ignore the passed-in context. We're likely creating a task in response to a gin request,
			// but gin cancels its context as soon as the request ends, and our async dispatcher needs to
			// persist long enough to send the task.
			ctx = context.Background()
			err := sendToTaskTarget(ctx, httpClient, queue, task)
			if err != nil {
				logger.For(ctx).WithError(err).Errorf("failed to direct dispatch task to queue: %s", queue)
			}
		}()
		return nil
	}
}

func useCloudTasks(ctx context.Context, gcpClient *gcptasks.Client) func(ctx context.Context, queue string, task *taskspb.Task) error {
	logger.For(ctx).Info("Initializing task client with cloud tasks")
	return func(ctx context.Context, queue string, task *taskspb.Task) error {
		return sendToTaskQueue(ctx, gcpClient, queue, task)
	}
}

func newGCPClient(ctx context.Context) *gcptasks.Client {
	trace := tracing.NewTracingInterceptor(true)

	copts := []option.ClientOption{
		option.WithGRPCDialOption(grpc.WithUnaryInterceptor(trace.UnaryInterceptor)),
		option.WithGRPCDialOption(grpc.WithTimeout(time.Duration(2) * time.Second)),
	}

	// Configure the client depending on whether or not the cloud task emulator is used.
	if env.GetString("ENV") == "local" {
		if host := env.GetString("TASK_QUEUE_HOST"); host != "" {
			copts = append(
				copts,
				option.WithEndpoint(host),
				option.WithGRPCDialOption(grpc.WithTransportCredentials(insecure.NewCredentials())),
				option.WithoutAuthentication(),
			)
		} else {
			key := "./secrets/dev/service-key-dev.json"
			if env.GetString("GCLOUD_SERVICE_KEY_OVERRIDE") != "" {
				key = env.GetString("GCLOUD_SERVICE_KEY_OVERRIDE")
			}
			fi, err := util.LoadEncryptedServiceKeyOrError(key)
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

func WithDelay(delay time.Duration) func(*taskspb.Task) error {
	return func(t *taskspb.Task) error {
		scheduleOn := time.Now().Add(delay)
		t.ScheduleTime = timestamppb.New(scheduleOn)
		return nil
	}
}

func withDeadline(d time.Duration) func(*taskspb.Task) error {
	return func(t *taskspb.Task) error {
		t.DispatchDeadline = durationpb.New(d)
		return nil
	}
}

func withBasicAuth(secret string) func(*taskspb.Task) error {
	return func(t *taskspb.Task) error {
		addHeader(t.GetHttpRequest(), "Authorization", basicauth.MakeHeader(nil, secret))
		return nil
	}
}

func withJSON(data any) func(*taskspb.Task) error {
	return func(t *taskspb.Task) error {
		body, err := json.Marshal(data)
		if err != nil {
			return err
		}
		t.GetHttpRequest().Body = body
		addHeader(t.GetHttpRequest(), "Content-type", "application/json")
		return nil
	}
}

func withTrace(span *sentry.Span) func(*taskspb.Task) error {
	return func(t *taskspb.Task) error {
		addHeader(t.GetHttpRequest(), "sentry-trace", span.TraceID.String())
		return nil
	}
}

func addHeader(r *taskspb.HttpRequest, key, value string) {
	if r.Headers == nil {
		r.Headers = map[string]string{}
	}
	r.Headers[key] = value
}

func (c *Client) submitTask(ctx context.Context, queue, url string, opts ...func(*taskspb.Task) error) error {
	if c.skipQueues[queue] {
		logger.For(ctx).Infof("skipping task for queue: %s", queue)
		return nil
	}

	task := &taskspb.Task{
		MessageType: &taskspb.Task_HttpRequest{
			HttpRequest: &taskspb.HttpRequest{
				HttpMethod: taskspb.HttpMethod_POST,
				Url:        url,
			},
		},
	}
	for _, opt := range opts {
		if err := opt(task); err != nil {
			return err
		}
	}

	return c.sendFunc(ctx, queue, task)
}

func sendToTaskQueue(ctx context.Context, gcpClient *gcptasks.Client, queue string, task *taskspb.Task) error {
	_, err := gcpClient.CreateTask(ctx, &taskspb.CreateTaskRequest{Parent: queue, Task: task})
	return err
}

func sendToTaskTarget(ctx context.Context, client *http.Client, queue string, task *taskspb.Task) error {
	name := task.GetName()
	method := task.GetHttpRequest().GetHttpMethod().String()
	url := task.GetHttpRequest().GetUrl()
	headers := task.GetHttpRequest().GetHeaders()
	body := bytes.NewReader(task.GetHttpRequest().GetBody())

	if task.ScheduleTime != nil {
		scheduleAt := task.ScheduleTime.AsTime()
		time.Sleep(time.Until(scheduleAt))
	}

	if task.DispatchDeadline != nil {
		var cancel context.CancelFunc
		ctx, cancel = context.WithDeadline(ctx, time.Now().Add(task.DispatchDeadline.AsDuration()))
		defer cancel()
	}

	// Create a new HTTP request
	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return err
	}

	// Add headers to the request
	for key, value := range headers {
		req.Header.Add(key, value)
	}

	// Our task handlers expect these to be set
	if name == "" {
		req.Header.Add("X-CloudTasks-TaskName", "direct-dispatch-task-"+persist.GenerateID().String())
	}
	req.Header.Add("X-CloudTasks-QueueName", queue)

	// Dispatch the request
	resp, err := client.Do(req)
	if err != nil {
		return err
	}

	if resp != nil {
		resp.Body.Close() // Always close the response body
	}

	return nil
}
