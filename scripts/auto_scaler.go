package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/nats-io/nats.go"
)

const (
	QueueThreshold   = 10
	CheckInterval    = 5 * time.Second
	MaxReplicas     = 5
)

var (
	nc          *nats.Conn
	dockerCli   *client.Client
	agentTypes  = []string{
		"resume-parser",
		"vacancy-matcher",
		"interview-scheduler",
		"feedback-agent",
	}
)

type NATSStats struct {
	Subscriptions int `json:"subscriptions"`
	Connections  int `json:"connections"`
}

type ScalingEvent struct {
	Timestamp  time.Time
	AgentType  string
	OldCount   int
	NewCount   int
	Reason     string
}

func main() {
	log.Println("Agent Auto-Scaler started")

	if err := connectNATS(); err != nil {
		log.Fatalf("Failed to connect to NATS: %v", err)
	}
	defer nc.Close()

	if err := initDocker(); err != nil {
		log.Printf("Warning: Docker not available: %v", err)
	}

	log.Println("Starting auto-scaling monitor...")
	runAutoScaler()
}

func connectNATS() error {
	var err error
	nc, err = nats.Connect(nats.DefaultURL)
	return err
}

func initDocker() error {
	var err error
	dockerCli, err = client.NewClientWithOpts(client.FromEnv)
	return err
}

func getQueueLength() int {
	// Try to get queue stats from NATS monitoring
	resp, err := http.Get("http://localhost:8222/varz")
	if err != nil {
		log.Printf("Failed to get NATS stats: %v", err)
		return 0
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0
	}

	var stats NATSStats
	if err := json.NewDecoder(resp.Body).Decode(&stats); err != nil {
		log.Printf("Failed to decode NATS stats: %v", err)
		return 0
	}

	return stats.Subscriptions
}

func getContainerCount(serviceName string) int {
	if dockerCli == nil {
		return 1
	}

	ctx := context.Background()
	containers, err := dockerCli.ContainerList(ctx, container.ListOptions{All: true})
	if err != nil {
		log.Printf("Failed to list containers: %v", err)
		return 1
	}

	count := 0
	for _, c := range containers {
		for _, name := range c.Names {
			if len(name) > 1 && name[1:] == serviceName {
				count++
			}
		}
	}

	if count == 0 {
		count = 1
	}
	return count
}

func scaleService(serviceName string, replicas int) error {
	if dockerCli == nil {
		log.Printf("Would scale %s to %d replicas (Docker not available)", serviceName, replicas)
		return nil
	}

	ctx := context.Background()

	// Get current container count
	currentCount := getContainerCount(serviceName)

	if replicas > currentCount {
		log.Printf("Scaling %s: %d -> %d replicas", serviceName, currentCount, replicas)

		// In a real implementation, we would use docker-compose or Kubernetes API
		// For demo purposes, we log the scaling action
		env := []string{
			"ENABLE_LOGGING=true",
		}

		_, err := dockerCli.ContainerCreate(
			ctx,
			&container.Config{
				Image: serviceName,
				Env:   env,
			},
			nil,
			nil,
			nil,
		)

		if err != nil {
			return fmt.Errorf("failed to create container: %w", err)
		}

		log.Printf("Successfully scaled %s to %d replicas", serviceName, replicas)
	} else if replicas < currentCount {
		log.Printf("Would scale down %s to %d replicas", serviceName, replicas)
	}

	return nil
}

func runAutoScaler() {
	for {
		queueLen := getQueueLength()

		log.Printf("Queue length: %d, Threshold: %d", queueLen, QueueThreshold)

		if queueLen > QueueThreshold {
			for _, agentType := range agentTypes {
				currentCount := getContainerCount(agentType)

				if currentCount < MaxReplicas {
					newReplicas := currentCount + 1
					event := ScalingEvent{
						Timestamp: time.Now(),
						AgentType: agentType,
						OldCount:  currentCount,
						NewCount:  newReplicas,
						Reason:    fmt.Sprintf("Queue length %d > threshold %d", queueLen, QueueThreshold),
					}

					logScalingEvent(event)

					if err := scaleService(agentType, newReplicas); err != nil {
						log.Printf("Failed to scale %s: %v", agentType, err)
					}
				}
			}
		}

		time.Sleep(CheckInterval)
	}
}

func logScalingEvent(event ScalingEvent) {
	log.Printf("[SCALE] %s: %s %d -> %d (reason: %s)",
		event.Timestamp.Format(time.RFC3339),
		event.AgentType,
		event.OldCount,
		event.NewCount,
		event.Reason,
	)
}

func getEnvInt(key string, defaultVal int) int {
	if val := os.Getenv(key); val != "" {
		if intVal, err := strconv.Atoi(val); err == nil {
			return intVal
		}
	}
	return defaultVal
}
