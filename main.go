package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/events"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
	"github.com/natefinch/lumberjack"
)

// Environment variable configuration

const TARGET_CONTAINERS = "TARGET_CONTAINERS"
const MAX_LOG_SIZE_MB = "MAX_LOG_SIZE_MB"
const MAX_BACKUPS = "MAX_BACKUPS"
const MAX_AGE_DAYS = "MAX_AGE_DAYS"
const LOG_DIR = "LOG_DIR"

func main() {
	// Read environment variables
	targetContainersEnv := os.Getenv(TARGET_CONTAINERS)
	maxSize := getEnvInt(MAX_LOG_SIZE_MB, 10)
	maxBackups := getEnvInt(MAX_BACKUPS, 5)
	maxAge := getEnvInt(MAX_AGE_DAYS, 30)
	logDir := os.Getenv(LOG_DIR)

	if logDir == "" {
		logDir = "./logs"
	}

	// Ensure log directory exists
	if err := os.MkdirAll(logDir, 0755); err != nil {
		fmt.Println("Error creating log dir:", err)
		os.Exit(1)
		return
	}

	// Connect to Docker
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		fmt.Println("Error creating Docker client:", err)
		os.Exit(1)
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Graceful shutdown handling
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigs
		fmt.Println("Shutting down...")
		cancel()
	}()

	// Start logging for current containers
	var targetContainers []string
	if targetContainersEnv != "" {
		targetContainers = strings.Split(targetContainersEnv, ",")
	}

	containers, err := cli.ContainerList(ctx, container.ListOptions{All: false})
	if err != nil {
		fmt.Println("Error listing containers:", err)
		os.Exit(1)
		return
	}

	for _, c := range containers {
		name := strings.TrimPrefix(c.Names[0], "/")
		if len(targetContainers) == 0 || contains(targetContainers, name) || contains(targetContainers, c.ID[:12]) {
			go streamLogs(ctx, cli, c.ID, name, logDir, maxSize, maxBackups, maxAge)
		}
	}

	// Watch for new container starts
	eventFilter := filters.NewArgs()
	eventFilter.Add("type", "container")
	eventFilter.Add("event", "start")

	eventsCh, errsCh := cli.Events(ctx, events.ListOptions{Filters: eventFilter})
	for {
		select {
		case <-ctx.Done():
			return
		case err := <-errsCh:
			if err != nil && ctx.Err() == nil {
				fmt.Println("Error watching Docker events:", err)
			}
		case e := <-eventsCh:
			name := e.Actor.Attributes["name"]
			if targetContainersEnv == "" || contains(targetContainers, name) || contains(targetContainers, e.Actor.ID[:12]) {
				go streamLogs(ctx, cli, e.Actor.ID, name, logDir, maxSize, maxBackups, maxAge)
			}
		}
	}
}

func streamLogs(ctx context.Context, cli *client.Client, containerID, containerName, logDir string, maxSize, maxBackups, maxAge int) {
	logFile := fmt.Sprintf("%s/%s.log", logDir, containerName)
	writer := &lumberjack.Logger{
		Filename:   logFile,
		MaxSize:    maxSize,
		MaxBackups: maxBackups,
		MaxAge:     maxAge,
		Compress:   true,
	}

	fmt.Printf("Logging container %s (%s) → %s\n", containerName, containerID[:12], logFile)

	opts := container.LogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Follow:     true,
		Timestamps: false,
		Details:    false,
	}

	rc, err := cli.ContainerLogs(ctx, containerID, opts)
	if err != nil {
		fmt.Println("Error attaching to logs for", containerID, ":", err)
		return
	}
	defer rc.Close()

	_, err = stdcopy.StdCopy(writer, writer, rc)
	if err != nil && ctx.Err() == nil {
		fmt.Println("Error copying logs for", containerID, ":", err)
	}
}

func getEnvInt(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		var i int
		fmt.Sscanf(v, "%d", &i)
		if i > 0 {
			return i
		}
	}
	return def
}

func contains(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}
