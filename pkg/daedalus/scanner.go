package daedalus

import (
	"bufio"
	"bytes"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"gopkg.in/yaml.v3"
)

const bodyPreviewMaxLines = 20

// ScanBoard scans directory and builds in-memory state
func ScanBoard(rootPath string) (*BoardState, error) {
	absRoot, err := filepath.Abs(rootPath)
	if err != nil {
		return nil, fmt.Errorf("resolving root path: %w", err)
	}

	state := &BoardState{
		Lists:    make(map[string][]KanbanCard),
		RootPath: absRoot,
		MaxID:    0,
	}

	entries, err := os.ReadDir(absRoot)
	if err != nil {
		return nil, err
	}

	configStart := time.Now()
	config, err := LoadBoardConfig(absRoot)
	if err != nil {
		return nil, fmt.Errorf("loading board config: %w", err)
	}
	state.Config = config
	state.ConfigLoadTime = time.Since(configStart)

	scanStart := time.Now()
	var wg sync.WaitGroup
	var mutex sync.Mutex // protect state writes

	// loop over lists (directories)
	for _, entry := range entries {
		if entry.IsDir() && !strings.HasPrefix(entry.Name(), ".") && entry.Name() != "_assets" {
			dirName := entry.Name()
			listPath := filepath.Join(absRoot, dirName)

			wg.Add(1)
			go func(path, name string) {
				defer wg.Done()
				cards, localMaxID, localBytes := scanList(path, name)

				mutex.Lock()
				state.Lists[name] = cards
				if localMaxID > state.MaxID {
					state.MaxID = localMaxID
				}
				state.TotalFileBytes += localBytes
				mutex.Unlock()

			}(listPath, dirName)
		}
	}
	wg.Wait()
	state.ScanTime = time.Since(scanStart)
	return state, nil
}

// scanList iterates over a directory (list) of markdown files (cards)
func scanList(listPath, listName string) ([]KanbanCard, int, int64) {
	files, err := os.ReadDir(listPath)
	if err != nil {
		slog.Error("failed to read list directory", "list", listName, "path", listPath, "error", err)
		return nil, 0, 0
	}

	cards := []KanbanCard{}
	localMaxID := 0
	var localBytes int64

	for _, file := range files {

		if !file.IsDir() && strings.HasSuffix(file.Name(), ".md") {
			fileBase := strings.TrimSuffix(file.Name(), ".md")
			idFromFileName, _ := strconv.Atoi(fileBase)
			fullPath := filepath.Join(listPath, file.Name())

			meta, preview, err := parseFileHeader(fullPath)
			if err != nil {
				slog.Warn("skipping invalid card file", "file", file.Name(), "list", listName, "error", err)
				continue
			}

			// set ID if missing from frontmatter
			if meta.ID == 0 {
				meta.ID = idFromFileName
				slog.Debug("card ID missing from frontmatter, using filename", "file", file.Name(), "id", idFromFileName)
			}
			if meta.ID > localMaxID {
				localMaxID = meta.ID
			}

			if info, err := file.Info(); err == nil {
				localBytes += info.Size()
			} else {
				slog.Warn("failed to stat card file", "file", file.Name(), "error", err)
			}

			cards = append(cards, KanbanCard{
				FilePath:    fullPath,
				ListName:    listName,
				Metadata:    meta,
				PreviewText: preview,
			})
		}
	}
	sort.Slice(cards, func(i, j int) bool {
		// primary - list order
		if cards[i].Metadata.ListOrder != cards[j].Metadata.ListOrder {
			return cards[i].Metadata.ListOrder < cards[j].Metadata.ListOrder
		}
		// secondary - ID
		return cards[i].Metadata.ID < cards[j].Metadata.ID
	})

	slog.Debug("list scanned", "list", listName, "cards", len(cards), "maxID", localMaxID, "bytes", localBytes)
	return cards, localMaxID, localBytes
}

// scanCardFile reads a card file line by line, calling onFrontmatter for lines inside the --- delimiters
// and onBody for lines after the frontmatter block. Callbacks return false to stop scanning early.
func scanCardFile(s *bufio.Scanner, onFrontmatter, onBody func(line string) bool) {
	inFrontmatter := false
	dashCount := 0
	for s.Scan() {
		line := s.Text()
		if strings.TrimSpace(line) == "---" {
			dashCount++
			if dashCount == 1 {
				inFrontmatter = true
				continue
			}
			if dashCount == 2 {
				inFrontmatter = false
				continue
			}
		}
		if inFrontmatter {
			if onFrontmatter != nil && !onFrontmatter(line) {
				return
			}
		} else if dashCount >= 2 {
			if onBody != nil && !onBody(line) {
				return
			}
		}
	}
}

// parseFileHeader reads frontmatter and first few lines of card body
func parseFileHeader(path string) (CardMetadata, string, error) {
	file, err := os.Open(path)
	if err != nil {
		return CardMetadata{}, "", err
	}
	defer file.Close()

	var frontmatterBuf bytes.Buffer
	var bodyPreviewBuf bytes.Buffer
	bodyLines := 0

	s := bufio.NewScanner(file)
	scanCardFile(s,
		func(line string) bool {
			frontmatterBuf.WriteString(line + "\n")
			return true
		},
		func(line string) bool {
			bodyLines++
			if bodyLines > bodyPreviewMaxLines {
				return false
			}
			if bodyPreviewBuf.Len() < PreviewMaxLen {
				bodyPreviewBuf.WriteString(line + "\n")
			}
			return true
		},
	)

	if err := s.Err(); err != nil {
		return CardMetadata{}, "", fmt.Errorf("reading card file: %w", err)
	}

	var meta CardMetadata
	if frontmatterBuf.Len() > 0 {
		if err := yaml.Unmarshal(frontmatterBuf.Bytes(), &meta); err != nil {
			slog.Warn("failed to parse card frontmatter", "path", path, "error", err)
			return CardMetadata{}, "", fmt.Errorf("yaml parse error: %w", err)
		}
	}
	return meta, bodyPreviewBuf.String(), nil
}

// readRawFrontmatter reads an existing file and parses the YAML between --- delimiters into a raw map.
// Uses scanCardFile for robust line-by-line delimiter matching.
func readRawFrontmatter(path string) (map[string]any, error) {
	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	defer file.Close()

	var buf bytes.Buffer
	s := bufio.NewScanner(file)
	scanCardFile(s, func(line string) bool {
		buf.WriteString(line + "\n")
		return true
	}, nil)

	if err := s.Err(); err != nil {
		return nil, fmt.Errorf("reading frontmatter: %w", err)
	}

	if buf.Len() == 0 {
		return nil, nil
	}

	raw := make(map[string]any)
	if err := yaml.Unmarshal(buf.Bytes(), &raw); err != nil {
		slog.Warn("failed to parse existing frontmatter", "path", path, "error", err)
		return nil, fmt.Errorf("yaml parse error: %w", err)
	}
	return raw, nil
}

// knownMetaKeys caches YAML field names from CardMetadata struct tags.
// Computed once via sync.Once since the struct definition never changes at runtime.
var (
	knownMetaKeys     map[string]bool
	knownMetaKeysOnce sync.Once
)

// getKnownMetaKeys returns the cached set of YAML field names from CardMetadata.
func getKnownMetaKeys() map[string]bool {
	knownMetaKeysOnce.Do(func() {
		keys := make(map[string]bool)
		t := reflect.TypeFor[CardMetadata]()
		for i := 0; i < t.NumField(); i++ {
			tag := t.Field(i).Tag.Get("yaml")
			name := strings.Split(tag, ",")[0]
			if name != "" {
				keys[name] = true
			}
		}
		knownMetaKeys = keys
	})
	return knownMetaKeys
}

// mergeUnknownFields merges unknown keys from existingRaw into metaRaw.
// Known CardMetadata keys are NOT merged back -- yaml.Marshal already handled
// omitempty correctly, so re-merging old values would undo intentional removals.
func mergeUnknownFields(metaRaw, existingRaw map[string]any) map[string]any {
	knownKeys := getKnownMetaKeys()
	for k, v := range existingRaw {
		if _, exists := metaRaw[k]; !exists && !knownKeys[k] {
			metaRaw[k] = v
		}
	}
	return metaRaw
}

// marshalOrderedYAML marshals a merged field map into YAML bytes with a
// controlled key order: priority keys first, then remaining keys sorted
// alphabetically, with trello_data forced to the very end.
func marshalOrderedYAML(merged map[string]any) ([]byte, error) {
	priorityKeys := []string{
		"id", "title", "list_order", "created", "updated",
		"due", "range", "labels", "icon", "url", "estimate",
	}
	added := make(map[string]bool)
	var yamlBuf bytes.Buffer

	for _, key := range priorityKeys {
		if val, ok := merged[key]; ok {
			b, err := yaml.Marshal(map[string]any{key: val})
			if err != nil {
				return nil, fmt.Errorf("marshaling field %s: %w", key, err)
			}
			yamlBuf.Write(b)
			added[key] = true
		}
	}

	// Remaining keys sorted alphabetically, with trello_data forced to the very end
	var remaining []string
	for key := range merged {
		if !added[key] && key != "trello_data" {
			remaining = append(remaining, key)
		}
	}
	sort.Strings(remaining)
	if _, ok := merged["trello_data"]; ok {
		remaining = append(remaining, "trello_data")
	}

	for _, key := range remaining {
		b, err := yaml.Marshal(map[string]any{key: merged[key]})
		if err != nil {
			return nil, fmt.Errorf("marshaling field %s: %w", key, err)
		}
		yamlBuf.Write(b)
	}
	return yamlBuf.Bytes(), nil
}

// WriteCardFile writes a card's metadata and body to a markdown file, preserving unknown YAML fields.
func WriteCardFile(path string, meta CardMetadata, body string) error {
	existingRaw, err := readRawFrontmatter(path)
	if err != nil {
		return fmt.Errorf("reading existing frontmatter: %w", err)
	}

	metaYaml, err := yaml.Marshal(&meta)
	if err != nil {
		return fmt.Errorf("marshaling metadata: %w", err)
	}
	metaRaw := make(map[string]any)
	if err := yaml.Unmarshal(metaYaml, &metaRaw); err != nil {
		return fmt.Errorf("unmarshaling metadata map: %w", err)
	}

	merged := mergeUnknownFields(metaRaw, existingRaw)

	// Force-quote checklist desc fields to prevent YAML parsing issues
	if checklist, ok := merged["checklist"].([]any); ok {
		for _, item := range checklist {
			if m, ok := item.(map[string]any); ok {
				if desc, ok := m["desc"].(string); ok {
					m["desc"] = &yaml.Node{Kind: yaml.ScalarNode, Value: desc, Style: yaml.DoubleQuotedStyle}
				}
			}
		}
	}

	finalYaml, err := marshalOrderedYAML(merged)
	if err != nil {
		return fmt.Errorf("marshaling ordered YAML: %w", err)
	}

	var buf bytes.Buffer
	buf.WriteString("---\n")
	buf.Write(finalYaml)
	buf.WriteString("---\n")
	buf.WriteString(body)

	if err := os.WriteFile(path, buf.Bytes(), 0644); err != nil {
		slog.Error("failed to write card file", "path", path, "error", err)
		return err
	}
	return nil
}

// ReadCardContent reads a card file and returns the full markdown body (after frontmatter)
func ReadCardContent(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		slog.Error("failed to open card file", "path", path, "error", err)
		return "", err
	}
	defer file.Close()

	s := bufio.NewScanner(file)
	var bodyBuf bytes.Buffer

	scanCardFile(s, nil, func(line string) bool {
		bodyBuf.WriteString(line + "\n")
		return true
	})

	if err := s.Err(); err != nil {
		slog.Error("error reading card file", "path", path, "error", err)
		return "", err
	}
	return bodyBuf.String(), nil
}
