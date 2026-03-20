package policy

import "fmt"

// SharedFragmentIDs returns a set of fragment IDs that are declared by more than one embedded package.
// This is used to support safe partial removal when multiple packages are applied to the same API policy.
func SharedFragmentIDs() (map[string]bool, error) {
	names, err := ListPackages()
	if err != nil {
		return nil, err
	}

	counts := map[string]int{}
	for _, name := range names {
		m, err := LoadPackageManifest(name)
		if err != nil {
			return nil, fmt.Errorf("load manifest %q: %w", name, err)
		}

		seen := map[string]bool{}
		for _, id := range append(append([]string{}, m.Inbound...), m.Outbound...) {
			if seen[id] {
				continue
			}
			seen[id] = true
			counts[id]++
		}
	}

	shared := map[string]bool{}
	for id, c := range counts {
		if c > 1 {
			shared[id] = true
		}
	}
	return shared, nil
}
