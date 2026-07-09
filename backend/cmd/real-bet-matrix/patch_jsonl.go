package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
)

func loadResultsJSONL(path string) (map[int]runResult, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	out := map[int]runResult{}
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		var r runResult
		if err := json.Unmarshal([]byte(line), &r); err != nil {
			return nil, fmt.Errorf("decode %q: %w", path, err)
		}
		out[r.Index] = r
	}
	return out, sc.Err()
}

func writeResultsJSONL(path string, byIndex map[int]runResult) error {
	idxs := make([]int, 0, len(byIndex))
	for i := range byIndex {
		idxs = append(idxs, i)
	}
	sort.Ints(idxs)
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	enc := json.NewEncoder(f)
	for _, i := range idxs {
		if err := enc.Encode(byIndex[i]); err != nil {
			return err
		}
		if _, err := f.WriteString("\n"); err != nil {
			return err
		}
	}
	return nil
}

func patchResultsJSONL(path string, updates map[int]runResult) error {
	existing, err := loadResultsJSONL(path)
	if err != nil {
		return err
	}
	for i, r := range updates {
		existing[i] = r
	}
	return writeResultsJSONL(path, existing)
}
