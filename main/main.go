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

type cacheConfig struct {
	WorkingDirectory string   `json:"WorkingDirectory"`
	RootDirs         []string `json:"RootDirs"`
	FollowSymlinks   bool     `json:"FollowSymlinks"`
	ExcludeDirs      []string `json:"ExcludeDirs"`
	PruneFiles       []string `json:"PruneFiles"`
	IncludeFiles     []string `json:"IncludeFiles"`
	IncludeSuffixes  []string `json:"IncludeSuffixes"`
	FilesystemView   string   `json:"FilesystemView"`
}

type cacheMetadata struct {
	Version string      `json:"Version"`
	Config  cacheConfig `json:"Config"`
}

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

	// Read and parse version line
	versionBytes, err := readLine(reader)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading version: %v\n", err)
		os.Exit(1)
	}
	if len(versionBytes) > 0 && versionBytes[len(versionBytes)-1] == lineSeparator {
		versionBytes = versionBytes[:len(versionBytes)-1]
	}
	versionString := string(versionBytes)
	fmt.Printf("Database version: %s\n", versionString)

	// Read and parse config line
	configBytes, err := readLine(reader)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading config: %v\n", err)
		os.Exit(1)
	}
	if len(configBytes) > 0 && configBytes[len(configBytes)-1] == lineSeparator {
		configBytes = configBytes[:len(configBytes)-1]
	}

	var metadata cacheMetadata
	if err := json.Unmarshal(configBytes, &metadata); err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing config JSON: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Config:\n")
	fmt.Printf("  WorkingDirectory: %s\n", metadata.Config.WorkingDirectory)
	fmt.Printf("  RootDirs: %v\n", metadata.Config.RootDirs)
	fmt.Printf("  FilesystemView: %s\n", metadata.Config.FilesystemView)
	fmt.Printf("  FollowSymlinks: %v\n", metadata.Config.FollowSymlinks)
	fmt.Printf("  ExcludeDirs: %v\n", metadata.Config.ExcludeDirs)
	fmt.Printf("  PruneFiles: %v\n", metadata.Config.PruneFiles)
	fmt.Printf("  IncludeFiles: %v\n", metadata.Config.IncludeFiles)
	fmt.Printf("  IncludeSuffixes: %v\n", metadata.Config.IncludeSuffixes)
	fmt.Println()

	// Read and parse cache entries
	allPaths := []string{}
	entryCount := 0
	fileCount := 0

	processCacheEntry := func(cacheEntry CacheEntry) {
		for _, persistedDirs := range cacheEntry {
			for _, dir := range persistedDirs.Dirs {
				path := joinCleanPaths(persistedDirs.Root, dir.P)
				// Clean the path
				path = filepath.Clean(path)
				if !filepath.IsAbs(path) {
					path = "/" + path
				}

				allPaths = append(allPaths, path)
				entryCount++
				fileCount += len(dir.F)

				// Also add file paths if there are files in this directory
				for _, filename := range dir.F {
					filePath := joinCleanPaths(path, filename)
					filePath = filepath.Clean(filePath)
					if !filepath.IsAbs(filePath) {
						filePath = "/" + filePath
					}
					allPaths = append(allPaths, filePath)
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

	// Remove duplicates and sort
	uniquePaths := make(map[string]bool)
	for _, path := range allPaths {
		uniquePaths[path] = true
	}

	sortedPaths := make([]string, 0, len(uniquePaths))
	for path := range uniquePaths {
		sortedPaths = append(sortedPaths, path)
	}

	// Sort paths
	for i := 0; i < len(sortedPaths); i++ {
		for j := i + 1; j < len(sortedPaths); j++ {
			if sortedPaths[i] > sortedPaths[j] {
				sortedPaths[i], sortedPaths[j] = sortedPaths[j], sortedPaths[i]
			}
		}
	}

	fmt.Printf("Database contains %d directory entries with %d files\n", entryCount, fileCount)
	fmt.Printf("Total unique paths: %d\n\n", len(sortedPaths))
	fmt.Println("All paths in database:")

	for _, path := range sortedPaths {
		fmt.Println(path)
	}
}
