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
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
	"go.opentelemetry.io/otel/trace"
)

var tracer trace.Tracer

type SchedulerMessage struct {
	TaskID    string        `json:"task_id"`
	Success   bool          `json:"success"`
	Candidate hr.Candidate  `json:"candidate"`
	Interview hr.Interview   `json:"interview"`
	Output    string        `json:"output"`
	Error     string        `json:"error,omitempty"`
	Timestamp time.Time     `json:"timestamp"`
}

func main() {
	initTracer()
	log.Println("Interview Scheduler Agent started")

	nc, err := nats.Connect(nats.DefaultURL)
	if err != nil {
		log.Fatalf("Failed to connect to NATS: %v", err)
	}
	defer nc.Close()

	_, err = nc.Subscribe(hr.SubjectScheduler, func(m *nats.Msg) {
		ctx := extractTraceContext(m)
		ctx, span := tracer.Start(ctx, "schedule_interview")
		defer span.End()

		var matchResult MatchResultMessage
		if err := json.Unmarshal(m.Data, &matchResult); err != nil {
			log.Printf("Failed to unmarshal: %v", err)
			return
		}

		log.Printf("Scheduling interview for candidate %s", matchResult.TaskID)

		result := processScheduling(matchResult, ctx, span)
		
		response, _ := json.Marshal(result)
		if err := nc.Publish(hr.SubjectFeedback, response); err != nil {
			log.Printf("Failed to publish result: %v", err)
		}

		span.SetAttributes(attribute.String("candidate.id", matchResult.TaskID))
		span.SetAttributes(attribute.String("interview.time", result.Interview.ScheduledAt.Format(time.RFC3339)))
	})

	log.Printf("Subscribed to %s", hr.SubjectScheduler)
	
	select {}
}

type MatchResultMessage struct {
	TaskID      string       `json:"task_id"`
	MatchScore  float64      `json:"match_score"`
	Candidate   hr.Candidate `json:"candidate"`
}

func initTracer() {
	ctx := context.Background()
	
	exporter, err := otlptracegrpc.New(ctx,
		otlptracegrpc.WithEndpoint("jaeger:4317"),
		otlptracegrpc.WithInsecure(),
	)
	if err != nil {
		log.Printf("Warning: Jaeger exporter not available: %v", err)
		tracer = otel.Tracer("interview-scheduler")
		return
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceName("interview-scheduler"),
		)),
	)

	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	tracer = tp.Tracer("interview-scheduler")
}

func processScheduling(match MatchResultMessage, ctx context.Context, span trace.Span) SchedulerMessage {
	result := SchedulerMessage{
		TaskID:    match.TaskID,
		Success:   true,
		Timestamp: time.Now(),
	}

	interviewDate := time.Now().AddDate(0, 0, 7)
	slots := hr.GenerateTimeSlots(interviewDate, 9, 18, 60)
	preferredTime := time.Date(interviewDate.Year(), interviewDate.Month(), interviewDate.Day(), 10, 0, 0, 0, interviewDate.Location())
	
	var selectedSlot *hr.TimeSlot
	for i := range slots {
		if slots[i].Available {
			slots[i].Available = false
			selectedSlot = &slots[i]
			break
		}
	}

	if selectedSlot == nil {
		selectedSlot = &hr.TimeSlot{
			Start: preferredTime,
			End:   preferredTime.Add(60 * time.Minute),
		}
	}

	allInterviewers := []string{"HR Manager", "Tech Lead", "Senior Developer", "Team Lead"}
	selectedInterviewers := hr.SelectInterviewers(2, allInterviewers)

	result.Interview = hr.Interview{
		ID:             fmt.Sprintf("int-%s", match.TaskID[:8]),
		CandidateID:    match.TaskID,
		VacancyID:      match.Candidate.Vacancy.ID,
		CandidateName:  match.Candidate.Resume.Name,
		Position:       match.Candidate.Vacancy.Title,
		ScheduledAt:    selectedSlot.Start,
		Duration:       60,
		Location:       "Conference Room A / Zoom",
		Interviewers:   selectedInterviewers,
		Status:         "scheduled",
	}

	if os.Getenv("ENABLE_LOGGING") == "true" {
		log.Printf("Interview scheduled: %s at %s", 
			result.Interview.ID, result.Interview.ScheduledAt.Format(time.RFC3339))
	}

	result.Output = fmt.Sprintf("Interview scheduled for %s at %s with %s",
		result.Interview.CandidateName,
		result.Interview.ScheduledAt.Format("15:04 on Monday, 02 Jan"),
		joinStrings(result.Interview.Interviewers, ", "))

	span.SetAttributes(
		attribute.String("interview.id", result.Interview.ID),
		attribute.String("interview.time", result.Interview.ScheduledAt.Format(time.RFC3339)),
		attribute.Int("interview.interviewers_count", len(result.Interview.Interviewers)),
	)

	return result
}

func joinStrings(strs []string, sep string) string {
	if len(strs) == 0 {
		return ""
	}
	result := strs[0]
	for i := 1; i < len(strs); i++ {
		result += sep + strs[i]
	}
	return result
}

func extractTraceContext(m *nats.Msg) context.Context {
	propagator := otel.GetTextMapPropagator()
	return propagator.Extract(context.Background(), propagation.HeaderCarrier(m.Header))
}
