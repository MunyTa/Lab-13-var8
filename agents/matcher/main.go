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

type MatchResultMessage struct {
	TaskID      string         `json:"task_id"`
	Success     bool           `json:"success"`
	Candidate   hr.Candidate   `json:"candidate"`
	MatchScore  float64        `json:"match_score"`
	Output      string         `json:"output"`
	Error       string         `json:"error,omitempty"`
	Timestamp   time.Time      `json:"timestamp"`
}

func main() {
	initTracer()
	log.Println("Vacancy Matcher Agent started")

	nc, err := nats.Connect(nats.DefaultURL)
	if err != nil {
		log.Fatalf("Failed to connect to NATS: %v", err)
	}
	defer nc.Close()

	_, err = nc.Subscribe(hr.SubjectMatcher, func(m *nats.Msg) {
		ctx := extractTraceContext(m)
		ctx, span := tracer.Start(ctx, "match_candidates")
		defer span.End()

		var parsedResult ParsedResultMessage
		if err := json.Unmarshal(m.Data, &parsedResult); err != nil {
			log.Printf("Failed to unmarshal: %v", err)
			return
		}

		log.Printf("Matching resume %s to vacancies", parsedResult.TaskID)

		result := processMatching(parsedResult, ctx, span)
		
		response, _ := json.Marshal(result)
		if err := nc.Publish(hr.SubjectScheduler, response); err != nil {
			log.Printf("Failed to publish result: %v", err)
		}

		span.SetAttributes(attribute.String("task.id", parsedResult.TaskID))
		span.SetAttributes(attribute.Float64("match.score", result.MatchScore))
	})

	log.Printf("Subscribed to %s", hr.SubjectMatcher)
	
	select {}
}

type ParsedResultMessage struct {
	TaskID  string       `json:"task_id"`
	Resume  hr.Resume    `json:"resume"`
	Parsed  *hr.ParsedResume `json:"parsed"`
}

func initTracer() {
	ctx := context.Background()
	
	exporter, err := otlptracegrpc.New(ctx,
		otlptracegrpc.WithEndpoint("jaeger:4317"),
		otlptracegrpc.WithInsecure(),
	)
	if err != nil {
		log.Printf("Warning: Jaeger exporter not available: %v", err)
		tracer = otel.Tracer("vacancy-matcher")
		return
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceName("vacancy-matcher"),
		)),
	)

	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	tracer = tp.Tracer("vacancy-matcher")
}

func processMatching(parsed ParsedResultMessage, ctx context.Context, span trace.Span) MatchResultMessage {
	result := MatchResultMessage{
		TaskID:    parsed.TaskID,
		Success:   true,
		Timestamp: time.Now(),
	}

	vacancy := getDefaultVacancy()

	matchResult := hr.MatchCandidateToVacancy(parsed.Resume, vacancy)
	
	result.MatchScore = matchResult.MatchScore
	result.Candidate = hr.Candidate{
		Resume:      parsed.Resume,
		Vacancy:     vacancy,
		MatchScore:  matchResult.MatchScore,
		Strengths:   matchResult.Strengths,
		Weaknesses:  matchResult.Weaknesses,
	}

	if os.Getenv("ENABLE_LOGGING") == "true" {
		log.Printf("Match score: %.2f%%, Strengths: %v, Weaknesses: %v",
			matchResult.MatchScore, matchResult.Strengths, matchResult.Weaknesses)
	}

	result.Output = fmt.Sprintf("Match score: %.2f%% with %d strengths and %d gaps",
		matchResult.MatchScore, len(matchResult.Strengths), len(matchResult.Weaknesses))

	span.SetAttributes(
		attribute.Float64("match.score", matchResult.MatchScore),
		attribute.Int("match.strengths", len(matchResult.Strengths)),
		attribute.Int("match.weaknesses", len(matchResult.Weaknesses)),
	)

	return result
}

func getDefaultVacancy() hr.Vacancy {
	return hr.Vacancy{
		ID:    "vac-hr-default",
		Title: "Go Backend Developer",
		Description: "Senior Go developer for microservices development",
		Requirements: []string{
			"5+ years of experience",
			"Proficiency in Go",
			"Experience with microservices",
		},
		Skills: []string{
			"Go", "Golang", "Docker", "Kubernetes",
			"PostgreSQL", "Redis", "gRPC", "REST API",
			"Microservices", "CI/CD",
		},
		Experience: "5+ years",
	}
}

func extractTraceContext(m *nats.Msg) context.Context {
	propagator := otel.GetTextMapPropagator()
	return propagator.Extract(context.Background(), propagation.HeaderCarrier(m.Header))
}
