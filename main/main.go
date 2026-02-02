package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

const lineSeparator = byte('\n')

type PersistedDirInfo struct {
	P string   `json:"P"` // path
	T int64    `json:"T"` // modification time
	I uint64   `json:"I"` // inode number
	F []string `json:"F"` // relevant filenames contained
}

type PersistedDirs struct {
	Device uint64             `json:"Device"`
	Root   string             `json:"Root"`
	Dirs   []PersistedDirInfo `json:"Dirs"`
}

type CacheEntry []PersistedDirs

func readLine(reader *bufio.Reader) ([]byte, error) {
	return reader.ReadBytes(lineSeparator)
}

func joinCleanPaths(base string, leaf string) string {
	if base == "" {
		return leaf
	}
	if base == "/" {
		return base + leaf
	}
	if leaf == "" {
		return base
	}
	return base + "/" + leaf
}

func main() {
	var dbPath string
	flag.StringVar(&dbPath, "db", "", "path to database file (required)")
	flag.Parse()

	if dbPath == "" {
		fmt.Fprintf(os.Stderr, "Error: -db flag is required\n")
		flag.Usage()
		os.Exit(1)
	}

	file, err := os.Open(dbPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error opening database file: %v\n", err)
		os.Exit(1)
	}
	defer file.Close()

	reader := bufio.NewReader(file)

	// Read and parse version line (skip it)
	_, err = readLine(reader)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading version: %v\n", err)
		os.Exit(1)
	}

	// Read and parse config line (skip it)
	_, err = readLine(reader)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading config: %v\n", err)
		os.Exit(1)
	}

	// Print paths as we read them (stream processing)
	processCacheEntry := func(cacheEntry CacheEntry) {
		for _, persistedDirs := range cacheEntry {
			for _, dir := range persistedDirs.Dirs {
				path := joinCleanPaths(persistedDirs.Root, dir.P)
				// Clean the path
				path = filepath.Clean(path)
				if !filepath.IsAbs(path) {
					path = "/" + path
				}

				// Print directory path
				fmt.Println(path)

				// Print file paths in this directory
				for _, filename := range dir.F {
					filePath := joinCleanPaths(path, filename)
					filePath = filepath.Clean(filePath)
					if !filepath.IsAbs(filePath) {
						filePath = "/" + filePath
					}
					fmt.Println(filePath)
				}
			}
		}
	}

	for {
		entryBytes, err := readLine(reader)
		if err != nil {
			if err == io.EOF {
				// Process any remaining data
				if len(entryBytes) > 0 {
					// Process this last line without newline
					var cacheEntry CacheEntry
					if err := json.Unmarshal(entryBytes, &cacheEntry); err == nil {
						processCacheEntry(cacheEntry)
					}
				}
				break
			}
			fmt.Fprintf(os.Stderr, "Error reading cache entry: %v\n", err)
			os.Exit(1)
		}

		if len(entryBytes) == 0 {
			continue
		}

		if len(entryBytes) > 0 && entryBytes[len(entryBytes)-1] == lineSeparator {
			entryBytes = entryBytes[:len(entryBytes)-1]
		}

		if len(entryBytes) == 0 {
			continue
		}

		var cacheEntry CacheEntry
		if err := json.Unmarshal(entryBytes, &cacheEntry); err != nil {
			fmt.Fprintf(os.Stderr, "Error parsing cache entry JSON: %v\n", err)
			os.Exit(1)
		}

		processCacheEntry(cacheEntry)
	}
}
