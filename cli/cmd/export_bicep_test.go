package cmd

import (
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hiddenlayerai/hiddenlayer-azure-apim-guardrails/cli/internal/policy"
)

func resetExportBicepFlags() {
	exportBicepPackage = ""
	exportBicepGroup = policy.ExportGroupAll
	exportBicepOut = ""
}

func TestExportBicepCommand_Success(t *testing.T) {
	resetExportBicepFlags()
	outDir := t.TempDir()

	if err := runRootExport("bicep",
		"--package", "v1-interactions",
		"--group", "outbound",
		"--out", outDir,
	); err != nil {
		t.Fatalf("export bicep command failed: %v", err)
	}

	if _, err := os.Stat(filepath.Join(outDir, "main.bicep")); err != nil {
		t.Fatalf("main.bicep not found: %v", err)
	}
	if _, err := os.Stat(filepath.Join(outDir, "main.bicepparam")); err != nil {
		t.Fatalf("main.bicepparam not found: %v", err)
	}
	if _, err := os.Stat(filepath.Join(outDir, "fragments", "hl-interactions-output.xml")); err != nil {
		t.Fatalf("expected outbound fragment not found: %v", err)
	}
}

func TestExportBicepCommand_InvalidGroup(t *testing.T) {
	resetExportBicepFlags()

	err := runRootExport("bicep",
		"--package", "v1-interactions",
		"--group", "invalid",
		"--out", t.TempDir(),
	)
	if err == nil {
		t.Fatal("export bicep command expected error for invalid group")
	}
}

func TestRootCommand_ExportSkipsConfigLoad(t *testing.T) {
	resetExportBicepFlags()
	outDir := t.TempDir()

	rootCmd.SetOut(io.Discard)
	rootCmd.SetErr(io.Discard)
	rootCmd.SetArgs([]string{
		"export", "bicep",
		"--package", "v1-interactions",
		"--group", "inbound",
		"--out", outDir,
	})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("root export command failed: %v", err)
	}

	if _, err := os.Stat(filepath.Join(outDir, "main.bicep")); err != nil {
		t.Fatalf("main.bicep not found: %v", err)
	}
}

func TestExportBicepCommand_ProducesCompilableBicep(t *testing.T) {
	resetExportBicepFlags()
	requireAzBicep(t)

	outDir := t.TempDir()
	if err := runRootExport("bicep",
		"--package", "v1-interactions",
		"--group", "all",
		"--out", outDir,
	); err != nil {
		t.Fatalf("export bicep command failed: %v", err)
	}

	cmd := azCommand(t, "bicep", "build", "--file", filepath.Join(outDir, "main.bicep"))
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("az bicep build failed: %v\noutput: %s", err, strings.TrimSpace(string(out)))
	}
}

func requireAzBicep(t *testing.T) {
	t.Helper()

	if _, err := exec.LookPath("az"); err != nil {
		t.Skipf("skipping az bicep validation test: az CLI not found: %v", err)
	}

	versionCmd := azCommand(t, "bicep", "version")
	out, err := versionCmd.CombinedOutput()
	if err != nil {
		t.Skipf("skipping az bicep validation test: bicep component unavailable: %v; output: %s", err, strings.TrimSpace(string(out)))
	}
}

func azCommand(t *testing.T, args ...string) *exec.Cmd {
	t.Helper()

	cmd := exec.Command("az", args...)
	cmd.Env = append(
		os.Environ(),
		"AZURE_CONFIG_DIR="+t.TempDir(),
		"AZURE_CORE_COLLECT_TELEMETRY=0",
	)
	return cmd
}

func runRootExport(args ...string) error {
	rootCmd.SetOut(io.Discard)
	rootCmd.SetErr(io.Discard)
	allArgs := append([]string{"export"}, args...)
	rootCmd.SetArgs(allArgs)
	return rootCmd.Execute()
}
