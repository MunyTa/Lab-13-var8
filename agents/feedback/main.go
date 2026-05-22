package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/MunyTa/Lab-13-var8/internal/hr"
	"github.com/nats-io/nats.go"
	"github.com/redis/go-redis/v9"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
	"go.opentelemetry.io/otel/trace"
)

var (
	tracer trace.Tracer
	redisClient *redis.Client
)

type FeedbackMessage struct {
	TaskID     string          `json:"task_id"`
	Success    bool            `json:"success"`
	Interview  hr.Interview    `json:"interview"`
	Feedbacks  []hr.Feedback   `json:"feedbacks"`
	AvgScore   float64         `json:"avg_score"`
	Output     string          `json:"output"`
	Error      string          `json:"error,omitempty"`
	Timestamp  time.Time       `json:"timestamp"`
}

func main() {
	initTracer()
	initRedis()
	log.Println("Feedback Agent started")

	nc, err := nats.Connect(nats.DefaultURL)
	if err != nil {
		log.Fatalf("Failed to connect to NATS: %v", err)
	}
	defer nc.Close()

	_, err = nc.Subscribe(hr.SubjectFeedback, func(m *nats.Msg) {
		ctx := extractTraceContext(m)
		ctx, span := tracer.Start(ctx, "process_feedback")
		defer span.End()

		var schedulerMsg SchedulerMessage
		if err := json.Unmarshal(m.Data, &schedulerMsg); err != nil {
			log.Printf("Failed to unmarshal: %v", err)
			return
		}

		log.Printf("Processing feedback for interview %s", schedulerMsg.Interview.ID)

		result := processFeedback(schedulerMsg, ctx, span)
		
		response, _ := json.Marshal(result)
		if err := nc.Publish(hr.SubjectTasksCompleted, response); err != nil {
			log.Printf("Failed to publish result: %v", err)
		}

		span.SetAttributes(attribute.String("interview.id", schedulerMsg.Interview.ID))
		span.SetAttributes(attribute.Float64("feedback.avg_score", result.AvgScore))
	})

	log.Printf("Subscribed to %s", hr.SubjectFeedback)
	
	select {}
}

type SchedulerMessage struct {
	TaskID    string        `json:"task_id"`
	Candidate hr.Candidate  `json:"candidate"`
	Interview hr.Interview   `json:"interview"`
}

func initTracer() {
	ctx := context.Background()
	
	exporter, err := otlptracegrpc.New(ctx,
		otlptracegrpc.WithEndpoint("jaeger:4317"),
		otlptracegrpc.WithInsecure(),
	)
	if err != nil {
		log.Printf("Warning: Jaeger exporter not available: %v", err)
		tracer = otel.Tracer("feedback-agent")
		return
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceName("feedback-agent"),
		)),
	)

	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	tracer = tp.Tracer("feedback-agent")
}

func initRedis() {
	redisAddr := os.Getenv("REDIS_URL")
	if redisAddr == "" {
		redisAddr = "redis:6379"
	}
	
	redisClient = redis.NewClient(&redis.Options{
		Addr: redisAddr,
	})
	
	ctx := context.Background()
	if err := redisClient.Ping(ctx).Err(); err != nil {
		log.Printf("Warning: Redis not available: %v", err)
	} else {
		log.Println("Connected to Redis")
	}
}

func processFeedback(msg SchedulerMessage, ctx context.Context, span trace.Span) FeedbackMessage {
	result := FeedbackMessage{
		TaskID:    msg.TaskID,
		Success:   true,
		Timestamp: time.Now(),
		Interview: msg.Interview,
		Feedbacks: generateMockFeedback(msg.Interview),
	}

	avgScore, recommendation := hr.ProcessFeedback(result.Feedbacks)
	result.AvgScore = avgScore

	report := hr.CreateSummaryReport(result.Feedbacks, avgScore, recommendation)
	result.Output = report

	saveToRedis(msg.Interview.CandidateID, avgScore, recommendation)

	if os.Getenv("ENABLE_LOGGING") == "true" {
		log.Printf("Feedback processed: avg_score=%.2f, recommendation=%s", avgScore, recommendation)
	}

	span.SetAttributes(
		attribute.Float64("feedback.score", avgScore),
		attribute.String("feedback.recommendation", recommendation),
		attribute.Int("feedback.count", len(result.Feedbacks)),
	)

	return result
}

func generateMockFeedback(interview hr.Interview) []hr.Feedback {
	return []hr.Feedback{
		{
			ID:             fmt.Sprintf("fb-%s-1", interview.ID),
			InterviewID:    interview.ID,
			CandidateID:    interview.CandidateID,
			Interviewer:    interview.Interviewers[0],
			Rating:         8,
			Pros:           []string{"Strong technical skills", "Good communication"},
			Cons:           []string{"Could improve problem-solving speed"},
			Recommendation: "Hire",
			Comments:       "Solid candidate for the position",
			CreatedAt:      time.Now(),
		},
		{
			ID:             fmt.Sprintf("fb-%s-2", interview.ID),
			InterviewID:    interview.ID,
			CandidateID:    interview.CandidateID,
			Interviewer:    interview.Interviewers[1],
			Rating:         9,
			Pros:           []string{"Excellent Go knowledge", "Team player"},
			Cons:           []string{"Limited experience with Kubernetes"},
			Recommendation: "Strong Hire",
			Comments:       "Would be a great addition to the team",
			CreatedAt:      time.Now(),
		},
	}
}

func saveToRedis(candidateID string, score float64, recommendation string) {
	if redisClient == nil {
		return
	}
	
	ctx := context.Background()
	key := fmt.Sprintf("hr:candidate:%s", candidateID)
	
	data := map[string]interface{}{
		"score":         score,
		"recommendation": recommendation,
		"updated_at":    time.Now().Format(time.RFC3339),
	}
	
	if err := redisClient.HSet(ctx, key, data).Err(); err != nil {
		log.Printf("Failed to save to Redis: %v", err)
	}

	redisClient.Incr(ctx, "hr:total_candidates_processed")
}

func extractTraceContext(m *nats.Msg) context.Context {
	propagator := otel.GetTextMapPropagator()
	return propagator.Extract(context.Background(), propagation.HeaderCarrier(m.Header))
}
