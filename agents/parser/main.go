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

type ParsedResult struct {
	TaskID    string       `json:"task_id"`
	Success   bool         `json:"success"`
	Resume    hr.Resume    `json:"resume"`
	Parsed    *hr.ParsedResume `json:"parsed"`
	Output    string       `json:"output"`
	Error     string       `json:"error,omitempty"`
	Timestamp time.Time    `json:"timestamp"`
}

func main() {
	initTracer()
	log.Println("Resume Parser Agent started")

	nc, err := nats.Connect(nats.DefaultURL)
	if err != nil {
		log.Fatalf("Failed to connect to NATS: %v", err)
	}
	defer nc.Close()

	_, err = nc.Subscribe(hr.SubjectResumeParser, func(m *nats.Msg) {
		ctx := extractTraceContext(m)
		ctx, span := tracer.Start(ctx, "parse_resume")
		defer span.End()

		var task hr.Task
		if err := json.Unmarshal(m.Data, &task); err != nil {
			log.Printf("Failed to unmarshal task: %v", err)
			return
		}

		log.Printf("Received task %s: parse resume", task.ID)

		result := processResume(task, ctx, span)
		
		response, _ := json.Marshal(result)
		if err := nc.Publish(hr.SubjectMatcher, response); err != nil {
			log.Printf("Failed to publish result: %v", err)
		}

		span.SetAttributes(attribute.String("task.id", task.ID))
		span.SetAttributes(attribute.Bool("success", result.Success))
	})

	log.Printf("Subscribed to %s", hr.SubjectResumeParser)
	
	select {}
}

func initTracer() {
	ctx := context.Background()
	
	exporter, err := otlptracegrpc.New(ctx,
		otlptracegrpc.WithEndpoint("jaeger:4317"),
		otlptracegrpc.WithInsecure(),
	)
	if err != nil {
		log.Printf("Warning: Jaeger exporter not available: %v", err)
		tracer = otel.Tracer("resume-parser")
		return
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceName("resume-parser"),
		)),
	)

	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	tracer = tp.Tracer("resume-parser")
}

func processResume(task hr.Task, ctx context.Context, span trace.Span) ParsedResult {
	result := ParsedResult{
		TaskID:    task.ID,
		Success:   true,
		Timestamp: time.Now(),
	}

	var payload map[string]interface{}
	if err := json.Unmarshal([]byte(task.Payload), &payload); err != nil {
		result.Success = false
		result.Error = fmt.Sprintf("Failed to parse payload: %v", err)
		return result
	}

	rawText, ok := payload["raw_text"].(string)
	if !ok {
		result.Success = false
		result.Error = "Missing raw_text in payload"
		return result
	}

	parsed := hr.ParseResumeText(rawText)

	result.Parsed = parsed
	result.Resume = hr.Resume{
		ID:    task.ID,
		RawText: rawText,
		Email: parsed.Email,
		Phone: parsed.Phone,
		Skills: parsed.Skills,
		Experience: parsed.Experience,
		Education: parsed.Education,
	}

	if os.Getenv("ENABLE_LOGGING") == "true" {
		log.Printf("Parsed resume: %s, Skills: %v", result.Resume.Name, result.Resume.Skills)
	}

	result.Output = fmt.Sprintf("Parsed %d skills, %d experiences, %d education entries",
		len(result.Resume.Skills), len(result.Resume.Experience), len(result.Resume.Education))

	span.SetAttributes(
		attribute.Int("resume.skills_count", len(result.Resume.Skills)),
		attribute.Int("resume.experience_count", len(result.Resume.Experience)),
	)

	return result
}

func extractTraceContext(m *nats.Msg) context.Context {
	propagator := otel.GetTextMapPropagator()
	return propagator.Extract(context.Background(), propagation.HeaderCarrier(m.Header))
}
