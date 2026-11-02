package main

import (
	"archive/zip"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
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

	os.MkdirAll("covers", os.ModePerm)

	var books []Metadata
	var mu sync.Mutex
	var processed uint64

	filesChan := make(chan string, 1000)
	var wg sync.WaitGroup

	// Lancer 12 threads de traitement en parallèle
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

	// Parcours des dossiers
	filepath.Walk(rootDir, func(path string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() && strings.HasSuffix(strings.ToLower(info.Name()), ".epub") {
			filesChan <- path
		}
		return nil
	})
	close(filesChan)

	wg.Wait()
	fmt.Printf("\nScan terminé ! %d EPUBs indexés.\n", len(books))

	// Sauvegarde de l'index JSON
	jsonData, _ := json.MarshalIndent(books, "", "  ")
	os.WriteFile("catalog.json", jsonData, 0644)
	fmt.Println("Index généré avec succès dans 'catalog.json'.")
}

func processEpub(epubPath string) (Metadata, error) {
	var meta Metadata
	meta.FilePath = epubPath

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

		// Extraction de la première image trouvée comme couverture
		if meta.CoverPath == "" && (strings.HasSuffix(f.Name, ".jpg") || strings.HasSuffix(f.Name, ".png")) {
			coverFilename := fmt.Sprintf("covers/%x_%s", strings.ToLower(filepath.Base(epubPath)), filepath.Base(f.Name))
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
