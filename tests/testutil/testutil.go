package testutil

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

const (
	defaultDBURL    = "postgres://nexus_user:nexus_password@localhost:5432/nexus?sslmode=disable"
	healthEndpoint  = "/health"
	healthTimeout   = 30 * time.Second
	healthInterval  = 500 * time.Millisecond
	serverStartWait = 2 * time.Second
)

type Env struct {
	Addr string
}

func Run(m interface{ Run() int }) int {
	addr := ""
	serverAddr := ":8081"
	if v := os.Getenv("TEST_SERVER_ADDR"); v != "" {
		addr = v
		if parsed, err := url.Parse(v); err == nil && parsed.Host != "" {
			if host, port, splitErr := net.SplitHostPort(parsed.Host); splitErr == nil {
				if host == "" {
					host = "localhost"
				}
				addr = "http://" + net.JoinHostPort(host, port)
				serverAddr = ":" + port
			}
		}
	} else {
		ephemeralAddr, err := pickAddr()
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to pick port: %v\n", err)
			return 1
		}
		addr = ephemeralAddr
		if parsed, err := url.Parse(ephemeralAddr); err == nil && parsed.Host != "" {
			if _, port, splitErr := net.SplitHostPort(parsed.Host); splitErr == nil {
				serverAddr = ":" + port
			}
		}
		_ = os.Setenv("TEST_SERVER_ADDR", addr)
	}

	dbURL := defaultDBURL
	if v := os.Getenv("DATABASE_URL"); v != "" {
		dbURL = v
	}

	if err := composeUp(); err != nil {
		fmt.Fprintf(os.Stderr, "compose up failed: %v\n", err)
		return 1
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	serverCmd, err := startServer(ctx, dbURL, serverAddr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "start server failed: %v\n", err)
		_ = composeDown()
		return 1
	}

	fmt.Println("server Start...", os.Getenv("TEST_SERVER_ADDR"))

	if err := waitForHealth(addr); err != nil {
		fmt.Fprintf(os.Stderr, "server health check failed: %v\n", err)
		_ = stopServer(serverCmd)
		_ = composeDown()
		return 1
	}

	code := m.Run()

	_ = stopServer(serverCmd)
	_ = composeDown()

	return code
}

func startServer(ctx context.Context, dbURL string, addr string) (*exec.Cmd, error) {
	root, err := findModuleRoot()
	if err != nil {
		return nil, err
	}
	binPath, err := buildTestServerBinary(ctx, root)
	if err != nil {
		return nil, err
	}

	cmd := exec.CommandContext(ctx, binPath, "-addr", addr)
	cmd.Dir = root
	cmd.Env = append(os.Environ(), "DATABASE_URL="+dbURL)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		return nil, err
	}

	time.Sleep(serverStartWait)
	return cmd, nil
}

func buildTestServerBinary(ctx context.Context, root string) (string, error) {
	tmpDir, err := os.MkdirTemp("", "nexus-test-server-")
	if err != nil {
		return "", err
	}

	binPath := filepath.Join(tmpDir, "nexus-test-server")
	buildCmd := exec.CommandContext(ctx, "go", "build", "-o", binPath, ".")
	buildCmd.Dir = root
	buildCmd.Stdout = os.Stdout
	buildCmd.Stderr = os.Stderr
	if err := buildCmd.Run(); err != nil {
		_ = os.RemoveAll(tmpDir)
		return "", err
	}

	return binPath, nil
}

func findModuleRoot() (string, error) {
	start, err := os.Getwd()
	if err != nil {
		return "", err
	}
	dir := start
	for {
		if _, statErr := os.Stat(filepath.Join(dir, "go.mod")); statErr == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("go.mod not found from %s", start)
		}
		dir = parent
	}
}

func stopServer(cmd *exec.Cmd) error {
	if cmd == nil || cmd.Process == nil {
		return nil
	}
	_ = cmd.Process.Kill()
	_, err := cmd.Process.Wait()
	cleanupServerBinary(cmd.Path)
	return err
}

func cleanupServerBinary(path string) {
	if path == "" {
		return
	}
	dir := filepath.Dir(path)
	if strings.HasPrefix(filepath.Base(dir), "nexus-test-server-") {
		_ = os.RemoveAll(dir)
	}
}

func waitForHealth(addr string) error {
	deadline := time.Now().Add(healthTimeout)
	url := addr + healthEndpoint
	for time.Now().Before(deadline) {
		resp, err := http.Get(url)
		if err == nil {
			_ = resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return nil
			}
		}
		time.Sleep(healthInterval)
	}
	return fmt.Errorf("health endpoint not ready at %s", url)
}

func pickAddr() (string, error) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return "", err
	}
	defer func() {
		_ = ln.Close()
	}()

	_, port, err := net.SplitHostPort(ln.Addr().String())
	if err != nil {
		return "", err
	}

	return "http://127.0.0.1:" + port, nil
}

func composeUp() error {
	if err := runCompose("up", "-d"); err == nil {
		return nil
	}
	return runComposeLegacy("up", "-d")
}

func composeDown() error {
	if err := runCompose("down"); err == nil {
		return nil
	}
	return runComposeLegacy("down")
}

func runCompose(args ...string) error {
	cmd := exec.Command("docker", append([]string{"compose"}, args...)...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func runComposeLegacy(args ...string) error {
	cmd := exec.Command("docker-compose", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
