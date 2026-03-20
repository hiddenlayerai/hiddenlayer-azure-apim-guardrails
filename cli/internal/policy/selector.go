package policy

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
)

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

	entries := make([]menuEntry, len(names))
	for i, name := range names {
		m, err := LoadPackageManifest(name)
		if err != nil {
			entries[i] = menuEntry{Name: name}
		} else {
			entries[i] = menuEntry{Name: name, Version: m.Version, Desc: m.Description}
		}
	}

	selected, err := selectFromMenu(entries, r, w)
	if err != nil {
		return nil, err
	}
	return LoadPackage(selected)
}

type menuEntry struct {
	Name     string
	Version  string
	Desc     string
	Requires string
}

func selectFromMenu(entries []menuEntry, r io.Reader, w io.Writer) (string, error) {
	if len(entries) == 0 {
		return "", fmt.Errorf("no entries to select from")
	}

	fmt.Fprintln(w, "Available fragment packages:")
	for i, e := range entries {
		versionSuffix := ""
		if e.Version != "" {
			versionSuffix = "@" + e.Version
		}
		if e.Desc != "" {
			fmt.Fprintf(w, "  %d) %s%s - %s\n", i+1, e.Name, versionSuffix, e.Desc)
		} else {
			fmt.Fprintf(w, "  %d) %s%s\n", i+1, e.Name, versionSuffix)
		}
		if e.Requires != "" {
			fmt.Fprintf(w, "     Requires: %s\n", e.Requires)
		}
	}
	fmt.Fprint(w, "Select package [1]: ")

	scanner := bufio.NewScanner(r)
	if !scanner.Scan() {
		return "", fmt.Errorf("no input received")
	}
	input := strings.TrimSpace(scanner.Text())
	if input == "" {
		input = "1"
	}

	choice, err := strconv.Atoi(input)
	if err != nil || choice < 1 || choice > len(entries) {
		return "", fmt.Errorf("invalid selection: %q", input)
	}

	return entries[choice-1].Name, nil
}

func SelectPackageStdin(flagValue string) (*Package, error) {
	return SelectPackage(flagValue, os.Stdin, os.Stdout)
}
