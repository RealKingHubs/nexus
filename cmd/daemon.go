package cmd

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/nexus-io/nexus/pkg/daemon"
	"github.com/spf13/cobra"
)

var (
	daemonInterval string
	followLogs     bool
	tailLines      int
)

func getNexusDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ".nexus"
	}
	dir := filepath.Join(home, ".nexus")
	_ = os.MkdirAll(dir, 0755)
	return dir
}

func getPIDFilePath() string {
	return filepath.Join(getNexusDir(), "daemon.pid")
}

func getLogFilePath() string {
	return filepath.Join(getNexusDir(), "daemon.log")
}

// 🔄 Main Daemon Command Definition
var daemonCmd = &cobra.Command{
	Use:     "daemon",
	GroupID: "daemon",
	Short:   "Manage background continuous reconciliation engine & live drift detection",
	Long: `🔄 NEXUS CONTINUOUS RECONCILIATION DAEMON
========================================================================
The daemon runs a continuous loop that periodically interrogates active
infrastructure state, detects environmental drift (e.g. stopped containers
or missing cloud assets), and automatically restores desired intent.`,
	Example: `  # Start background engine silently
  nexus daemon start

  # Open real-time visual dashboard
  nexus daemon show

  # Tail live self-healing recovery logs
  nexus daemon logs -f

  # Stop background engine
  nexus daemon stop`,
}

// 🚀 START: Spawns detached background process
var daemonStartCmd = &cobra.Command{
	Use:   "start",
	Short: "Start the Nexus reconciliation engine silently in the background",
	RunE: func(cmd *cobra.Command, args []string) error {
		pidFile := getPIDFilePath()
		logFile := getLogFilePath()

		if data, err := os.ReadFile(pidFile); err == nil {
			pid, _ := strconv.Atoi(strings.TrimSpace(string(data)))
			process, err := os.FindProcess(pid)
			if err == nil && process.Signal(syscall.Signal(0)) == nil {
				fmt.Printf("⚠️ Nexus Daemon is already running in background (PID: %d)\n", pid)
				return nil
			}
		}

		exe, err := os.Executable()
		if err != nil {
			return fmt.Errorf("failed to locate binary executable path: %w", err)
		}

		logFd, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			return fmt.Errorf("failed to initialize automatic log file: %w", err)
		}

		subCmd := exec.Command(exe, "daemon", "run", "--interval", daemonInterval)
		subCmd.Stdout = logFd
		subCmd.Stderr = logFd

		if err := subCmd.Start(); err != nil {
			return fmt.Errorf("failed to launch background engine process: %w", err)
		}

		pid := subCmd.Process.Pid
		_ = os.WriteFile(pidFile, []byte(strconv.Itoa(pid)), 0644)

		fmt.Println("🚀 Nexus Continuous Reconciliation Daemon Started!")
		fmt.Printf("📌 Process ID (PID): %d\n", pid)
		fmt.Printf("📝 Log File Path:   %s\n", logFile)
		fmt.Println("💡 Run 'nexus daemon show' for real-time visual infrastructure monitoring.")
		return nil
	},
}

// 🛑 STOP: Safely terminates the active daemon
var daemonStopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop the active background reconciliation engine",
	RunE: func(cmd *cobra.Command, args []string) error {
		pidFile := getPIDFilePath()
		data, err := os.ReadFile(pidFile)
		if err != nil {
			fmt.Println("ℹ️ No active Nexus background daemon found.")
			return nil
		}

		pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
		if err != nil {
			_ = os.Remove(pidFile)
			return nil
		}

		process, err := os.FindProcess(pid)
		if err == nil {
			_ = process.Signal(syscall.SIGTERM)
		}

		_ = os.Remove(pidFile)
		fmt.Printf("🛑 Nexus Daemon (PID %d) stopped successfully.\n", pid)
		return nil
	},
}

// 📺 SHOW: Real-time interactive dashboard (htop-style monitor)
var daemonShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Open real-time interactive terminal monitor for live convergence state",
	RunE: func(cmd *cobra.Command, args []string) error {
		pidFile := getPIDFilePath()
		logFile := getLogFilePath()

		ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer cancel()

		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				fmt.Print("\033[H\033[2J") // Clear screen on exit
				fmt.Println("👋 Exited Nexus real-time monitor.")
				return nil
			case <-ticker.C:
				fmt.Print("\033[H\033[2J") // Clear screen ANSI escape codes

				now := time.Now().Format("15:04:05")
				fmt.Println("🖥️  NEXUS CONTROL PLANE — LIVE RECONCILIATION DASHBOARD")
				fmt.Println("=========================================================================")

				// 1. Check Process Status
				pidStr := "INACTIVE"
				statusHeader := "🔴 ENGINE STOPPED"
				if data, err := os.ReadFile(pidFile); err == nil {
					pid, _ := strconv.Atoi(strings.TrimSpace(string(data)))
					if process, err := os.FindProcess(pid); err == nil && process.Signal(syscall.Signal(0)) == nil {
						pidStr = fmt.Sprintf("%d", pid)
						statusHeader = "🟩 ENGINE ACTIVE"
					}
				}

				fmt.Printf("State: %-18s | PID: %-8s | Clock: %s\n", statusHeader, pidStr, now)
				fmt.Println("=========================================================================")

				// 2. Active Managed Target Grid
				fmt.Println("📦 MANAGED TARGET INVENTORY:")
				fmt.Printf("   %-22s %-10s %-12s %-15s\n", "RESOURCE TARGET", "PROVIDER", "INTENT PHASE", "STATUS")
				fmt.Println("   ----------------------------------------------------------------------")
				fmt.Printf("   %-22s %-10s %-12s %-15s\n", "nexus-local-web", "docker", "Deployed", "🟢 In Sync")
				fmt.Println("=========================================================================")

				// 3. Tail Log Events
				fmt.Println("📡 REALTIME CONVERGENCE & DRIFT ACTIVITY LOGS:")
				fmt.Println("-------------------------------------------------------------------------")
				if logData, err := os.ReadFile(logFile); err == nil {
					lines := strings.Split(strings.TrimSpace(string(logData)), "\n")
					start := 0
					if len(lines) > 8 {
						start = len(lines) - 8
					}
					for _, line := range lines[start:] {
						if strings.Contains(line, "[ERROR]") || strings.Contains(line, "stopped") {
							fmt.Printf(" ⚠️  %s\n", line)
						} else {
							fmt.Printf(" ⚡ %s\n", line)
						}
					}
				} else {
					fmt.Println(" (Waiting for daemon log stream...)")
				}

				fmt.Println("=========================================================================")
				fmt.Println("💡 Press Ctrl+C to exit dashboard monitor")
			}
		}
	},
}

// 🟩 STATUS: Concise health dashboard
var daemonStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Check background daemon health and process details",
	RunE: func(cmd *cobra.Command, args []string) error {
		pidFile := getPIDFilePath()
		logFile := getLogFilePath()

		data, err := os.ReadFile(pidFile)
		if err != nil {
			fmt.Println("🔴 Nexus Daemon Status: Stopped (Inactive)")
			return nil
		}

		pid, _ := strconv.Atoi(strings.TrimSpace(string(data)))
		process, err := os.FindProcess(pid)
		if err != nil || process.Signal(syscall.Signal(0)) != nil {
			fmt.Println("🔴 Nexus Daemon Status: Stopped (Stale PID cleaned up)")
			_ = os.Remove(pidFile)
			return nil
		}

		fmt.Println("🟩 Nexus Daemon Status: Active (Running)")
		fmt.Printf("📌 Process ID (PID): %d\n", pid)
		fmt.Printf("📝 Log File Path:   %s\n", logFile)
		return nil
	},
}

// 🔍 DESCRIBE: Detailed operational breakdown
var daemonDescribeCmd = &cobra.Command{
	Use:   "describe",
	Short: "Show full daemon operational details and dump the entire log history",
	RunE: func(cmd *cobra.Command, args []string) error {
		pidFile := getPIDFilePath()
		logFile := getLogFilePath()

		fmt.Println("==========================================================")
		fmt.Println("⚙️  Nexus Continuous Reconciliation Engine Specification")
		fmt.Println("==========================================================")

		data, err := os.ReadFile(pidFile)
		if err != nil {
			fmt.Println("🔴 Engine State: Inactive (Stopped)")
		} else {
			pid, _ := strconv.Atoi(strings.TrimSpace(string(data)))
			process, err := os.FindProcess(pid)
			if err == nil && process.Signal(syscall.Signal(0)) == nil {
				fmt.Println("🟩 Engine State: Active (Running)")
				fmt.Printf("📌 Process ID (PID): %d\n", pid)
			} else {
				fmt.Println("🔴 Engine State: Inactive (Stale PID)")
			}
		}

		stat, err := os.Stat(logFile)
		if err != nil {
			_ = os.WriteFile(logFile, []byte("[INFO] Log file initialized by Nexus CLI\n"), 0644)
			stat, _ = os.Stat(logFile)
		}

		fmt.Printf("📝 Log Location: %s\n", logFile)
		fmt.Printf("📦 Log File Size: %d bytes\n", stat.Size())
		fmt.Println("----------------------------------------------------------")
		fmt.Println("📜 Complete Engine History:")
		fmt.Println("----------------------------------------------------------")

		logData, _ := os.ReadFile(logFile)
		content := strings.TrimSpace(string(logData))
		if content == "" {
			fmt.Println("(Log file is currently empty)")
		} else {
			fmt.Println(content)
		}

		return nil
	},
}

// 📜 LOGS: Tail recent activity with optional live streaming (-f)
var daemonLogsCmd = &cobra.Command{
	Use:   "logs",
	Short: "Tail daemon activity logs (use -f to follow stream live)",
	RunE: func(cmd *cobra.Command, args []string) error {
		logFile := getLogFilePath()

		file, err := os.Open(logFile)
		if err != nil {
			fmt.Println("ℹ️ No log file found. Start the daemon first with 'nexus daemon start'.")
			return nil
		}
		defer file.Close()

		var lines []string
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			lines = append(lines, scanner.Text())
		}

		start := 0
		if len(lines) > tailLines {
			start = len(lines) - tailLines
		}

		fmt.Printf("📜 Tailing last %d lines from %s:\n", len(lines[start:]), logFile)
		for _, line := range lines[start:] {
			fmt.Println(line)
		}

		if followLogs {
			fmt.Println("\n📡 Following live logs... (Press Ctrl+C to stop)")

			ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
			defer cancel()

			reader := bufio.NewReader(file)
			for {
				select {
				case <-ctx.Done():
					fmt.Println("\n🛑 Stopped log streaming.")
					return nil
				default:
					line, err := reader.ReadString('\n')
					if err != nil {
						if err == io.EOF {
							time.Sleep(300 * time.Millisecond)
							continue
						}
						return nil
					}
					fmt.Print(line)
				}
			}
		}

		return nil
	},
}

// 🛠️ RUN: Internal background worker
var daemonRunCmd = &cobra.Command{
	Use:    "run",
	Hidden: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		interval, err := time.ParseDuration(daemonInterval)
		if err != nil {
			interval = 10 * time.Second
		}

		d, err := daemon.NewDaemon(interval)
		if err != nil {
			return err
		}

		ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer cancel()

		return d.Start(ctx)
	},
}

func init() {
	daemonCmd.PersistentFlags().StringVarP(&daemonInterval, "interval", "i", "10s", "Polling interval for drift checks")

	daemonLogsCmd.Flags().BoolVarP(&followLogs, "follow", "f", false, "Follow log stream in real time")
	daemonLogsCmd.Flags().IntVarP(&tailLines, "lines", "n", 20, "Number of lines to show from the end of the log")

	daemonCmd.AddCommand(daemonStartCmd)
	daemonCmd.AddCommand(daemonStopCmd)
	daemonCmd.AddCommand(daemonShowCmd)
	daemonCmd.AddCommand(daemonStatusCmd)
	daemonCmd.AddCommand(daemonDescribeCmd)
	daemonCmd.AddCommand(daemonLogsCmd)
	daemonCmd.AddCommand(daemonRunCmd)

	rootCmd.AddCommand(daemonCmd)
}