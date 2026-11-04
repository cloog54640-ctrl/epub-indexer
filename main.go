package main

import (
	"archive/zip"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
)

type Metadata struct {
	Title     string   `json:"title"`
	Authors   []string `json:"authors"`
	Subjects  []string `json:"subjects"`
	CoverPath string   `json:"cover_path"`
	FilePath  string   `json:"file_path"`
}

type OPF struct {
	Metadata struct {
		Title   string   `xml:"title"`
		Creator []string `xml:"creator"`
		Subject []string `xml:"subject"`
	} `xml:"metadata"`
}

func main() {
	rootDir := "."
	if len(os.Args) > 1 {
		rootDir = os.Args[1]
	}

	absPath, err := filepath.Abs(rootDir)
	if err == nil {
		rootDir = absPath
	}

	longRootDir := rootDir
	if !strings.HasPrefix(longRootDir, `\\?\`) && len(longRootDir) > 3 {
		longRootDir = `\\?\` + longRootDir
	}

	os.MkdirAll("covers", os.ModePerm)

	var books []Metadata
	var mu sync.Mutex
	var processed uint64

	filesChan := make(chan string, 2000)
	var wg sync.WaitGroup

	for i := 0; i < 12; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for path := range filesChan {
				meta, err := processEpub(path)
				if err == nil {
					mu.Lock()
					books = append(books, meta)
					mu.Unlock()
				}
				count := atomic.AddUint64(&processed, 1)
				if count%1000 == 0 {
					fmt.Printf("\rLivres traités : %d", count)
				}
			}
		}()
	}

	filepath.WalkDir(longRootDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if !d.IsDir() && strings.HasSuffix(strings.ToLower(d.Name()), ".epub") {
			filesChan <- path
		}
		return nil
	})
	close(filesChan)

	wg.Wait()
	fmt.Printf("\nScan terminé ! %d EPUBs indexés au total.\n", len(books))

	jsonData, _ := json.MarshalIndent(books, "", "  ")
	os.WriteFile("catalog.json", jsonData, 0644)
	fmt.Println("Index mis à jour avec succès dans 'catalog.json'.")
}

func processEpub(epubPath string) (Metadata, error) {
	var meta Metadata
	cleanPath := strings.TrimPrefix(epubPath, `\\?\`)
	meta.FilePath = filepath.ToSlash(cleanPath)

	r, err := zip.OpenReader(epubPath)
	if err != nil {
		return meta, err
	}
	defer r.Close()

	for _, f := range r.File {
		if strings.HasSuffix(f.Name, ".opf") {
			rc, err := f.Open()
			if err != nil {
				continue
			}
			var opf OPF
			decoder := xml.NewDecoder(rc)
			decoder.Decode(&opf)
			rc.Close()

			meta.Title = opf.Metadata.Title
			meta.Authors = opf.Metadata.Creator
			meta.Subjects = opf.Metadata.Subject
		}

		if meta.CoverPath == "" && (strings.HasSuffix(f.Name, ".jpg") || strings.HasSuffix(f.Name, ".png")) {
			coverFilename := fmt.Sprintf("covers/cov_%x_%s", strings.ToLower(filepath.Base(epubPath)), filepath.Base(f.Name))
			out, err := os.Create(coverFilename)
			if err == nil {
				rc, _ := f.Open()
				io.Copy(out, rc)
				rc.Close()
				out.Close()
				meta.CoverPath = coverFilename
			}
		}
	}

	if meta.Title == "" {
		meta.Title = strings.TrimSuffix(filepath.Base(epubPath), filepath.Ext(epubPath))
	}

	return meta, nil
}
