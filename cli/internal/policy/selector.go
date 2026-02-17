package policy

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
)

// SelectPackage resolves a package by flag value, auto-selection, or interactive menu.
// If flagValue is non-empty it loads that package directly.
// If only one package exists it auto-selects.
// Otherwise it presents a numbered menu reading from r and writing to w.
func SelectPackage(flagValue string, r io.Reader, w io.Writer) (*Package, error) {
	if flagValue != "" {
		return LoadPackage(flagValue)
	}

	names, err := ListPackages()
	if err != nil {
		return nil, err
	}
	if len(names) == 0 {
		return nil, fmt.Errorf("no fragment packages found")
	}
	if len(names) == 1 {
		return LoadPackage(names[0])
	}

	// Interactive menu
	fmt.Fprintln(w, "Available fragment packages:")
	for i, name := range names {
		m, err := LoadPackageManifest(name)
		if err != nil {
			fmt.Fprintf(w, "  %d) %s\n", i+1, name)
		} else {
			versionSuffix := ""
			if m.Version != "" {
				versionSuffix = "@" + m.Version
			}
			fmt.Fprintf(w, "  %d) %s%s - %s\n", i+1, name, versionSuffix, m.Description)
		}
	}
	fmt.Fprint(w, "Select package [1]: ")

	scanner := bufio.NewScanner(r)
	if !scanner.Scan() {
		return nil, fmt.Errorf("no input received")
	}
	input := strings.TrimSpace(scanner.Text())
	if input == "" {
		input = "1"
	}

	choice, err := strconv.Atoi(input)
	if err != nil || choice < 1 || choice > len(names) {
		return nil, fmt.Errorf("invalid selection: %q", input)
	}

	return LoadPackage(names[choice-1])
}

// SelectPackageStdin is a convenience wrapper that uses os.Stdin and os.Stdout.
func SelectPackageStdin(flagValue string) (*Package, error) {
	return SelectPackage(flagValue, os.Stdin, os.Stdout)
}
