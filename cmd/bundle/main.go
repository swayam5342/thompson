// Command bundle turns web/ into one self-contained HTML file: the
// scripts are inlined and main.wasm is embedded as gzip+base64, so the
// page works when dropped anywhere, with no other files next to it.
//
//	go run ./cmd/bundle -o dist/thompson-nfa.html
package main

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	dir := flag.String("web", "web", "directory with index.html, main.wasm and scripts")
	out := flag.String("o", "dist/thompson-nfa.html", "output file")
	flag.Parse()

	html := string(must(os.ReadFile(filepath.Join(*dir, "index.html"))))
	for _, src := range []string{"vendor/dagre.min.js", "wasm_exec.js"} {
		tag := fmt.Sprintf(`<script src="%s"></script>`, src)
		if !strings.Contains(html, tag) {
			log.Fatalf("index.html has no %s", tag)
		}
		js := string(must(os.ReadFile(filepath.Join(*dir, src))))
		js = strings.ReplaceAll(js, "</script", `<\/script`)
		html = strings.Replace(html, tag, "<script>\n"+js+"\n</script>", 1)
	}

	var gz bytes.Buffer
	zw := must(gzip.NewWriterLevel(&gz, gzip.BestCompression))
	must(zw.Write(must(os.ReadFile(filepath.Join(*dir, "main.wasm")))))
	if err := zw.Close(); err != nil {
		log.Fatal(err)
	}
	embed := `<script>window.THOMPSON_WASM_GZ="` + base64.StdEncoding.EncodeToString(gz.Bytes()) + `";</script>`
	if !strings.Contains(html, "<!--WASM-->") {
		log.Fatal("index.html has no <!--WASM--> marker")
	}
	html = strings.Replace(html, "<!--WASM-->", embed, 1)

	if err := os.MkdirAll(filepath.Dir(*out), 0o755); err != nil {
		log.Fatal(err)
	}
	if err := os.WriteFile(*out, []byte(html), 0o644); err != nil {
		log.Fatal(err)
	}
	fmt.Printf("wrote %s (%d KB)\n", *out, len(html)/1024)
}

func must[T any](v T, err error) T {
	if err != nil {
		log.Fatal(err)
	}
	return v
}
