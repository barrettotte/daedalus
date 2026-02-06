package daedalus

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"

	"gopkg.in/yaml.v3"
)

// ScanBoard scans directory and builds in-memory state
func ScanBoard(rootPath string) (*BoardState, error) {
	state := &BoardState{
		Lists:    make(map[string][]KanbanCard),
		RootPath: rootPath,
		MaxID:    0,
	}

	entries, err := os.ReadDir(rootPath)
	if err != nil {
		return nil, err
	}

	config, err := LoadBoardConfig(rootPath)
	if err != nil {
		return nil, fmt.Errorf("loading board config: %w", err)
	}
	state.Config = config

	var wg sync.WaitGroup
	var mutex sync.Mutex // protect state writes

	// loop over lists (directories)
	for _, entry := range entries {
		if entry.IsDir() && !strings.HasPrefix(entry.Name(), ".") {
			dirName := entry.Name()
			listPath := filepath.Join(rootPath, dirName)
			displayName := dirName

			if parts := strings.SplitN(dirName, "___", 2); len(parts) == 2 {
				displayName = parts[1]
			}

			wg.Add(1)
			go func(path, realName, displayName string) {
				defer wg.Done()
				cards, localMaxID := scanList(path, realName, displayName)

				mutex.Lock()

				state.Lists[realName] = cards
				if localMaxID > state.MaxID {
					state.MaxID = localMaxID
				}
				mutex.Unlock()

			}(listPath, dirName, displayName)
		}
	}
	wg.Wait()
	return state, nil
}

// scanList iterates over a directory (list) of markdown files (cards)
func scanList(listPath, realName, displayName string) ([]KanbanCard, int) {
	files, err := os.ReadDir(listPath)
	if err != nil {
		fmt.Printf("Error reading list %s: %v\n", realName, err)
		return nil, 0
	}

	var cards []KanbanCard
	localMaxID := 0

	for _, file := range files {

		if !file.IsDir() && strings.HasSuffix(file.Name(), ".md") {
			fileBase := strings.TrimSuffix(file.Name(), ".md")
			idFromFileName, _ := strconv.Atoi(fileBase)
			fullPath := filepath.Join(listPath, file.Name())

			meta, preview, err := parseFileHeader(fullPath)
			if err != nil {
				fmt.Printf("Skipping invalid file %s: %v\n", file.Name(), err)
				continue
			}

			// set ID if missing from frontmatter
			if meta.ID == 0 {
				meta.ID = idFromFileName
			}
			if meta.ID > localMaxID {
				localMaxID = meta.ID
			}

			cards = append(cards, KanbanCard{
				FilePath:    fullPath,
				ListName:    displayName,
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

	return cards, localMaxID
}

// parseFileHeader reads frontmatter and first few lines of card body
func parseFileHeader(path string) (CardMetadata, string, error) {
	file, err := os.Open(path)
	if err != nil {
		return CardMetadata{}, "", err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	var frontmatterBuf bytes.Buffer
	var bodyPreviewBuf bytes.Buffer

	inFrontmatter := false
	dashCount := 0
	linesToRead := 50 // only read a bit

	for scanner.Scan() && linesToRead > 0 {
		line := scanner.Text()
		linesToRead--

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
			frontmatterBuf.WriteString(line + "\n")
		} else if dashCount >= 2 {
			// in body
			if bodyPreviewBuf.Len() < 150 {
				bodyPreviewBuf.WriteString(line + "\n")
			}
		}
	}

	var meta CardMetadata
	if frontmatterBuf.Len() > 0 {
		if err := yaml.Unmarshal(frontmatterBuf.Bytes(), &meta); err != nil {
			return CardMetadata{}, "", fmt.Errorf("yaml parse error: %w", err)
		}
	}
	return meta, bodyPreviewBuf.String(), nil
}

// ReadCardContent reads a card file and returns the full markdown body (after frontmatter)
func ReadCardContent(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	var bodyBuf bytes.Buffer

	inFrontmatter := false
	dashCount := 0

	for scanner.Scan() {
		line := scanner.Text()

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
		if !inFrontmatter && dashCount >= 2 {
			bodyBuf.WriteString(line + "\n")
		}
	}

	if err := scanner.Err(); err != nil {
		return "", err
	}
	return bodyBuf.String(), nil
}
