// Command devenv runs fake Jellyfin, TMDB, Radarr, Sonarr and box-office services so the real Tipsarr server can be
// tried by hand on one machine (web, iOS simulator, Android emulator). Development only: see dev/README.md.
//
//	go run ./cmd/devenv                       serve the fakes
//	go run ./cmd/devenv -seed http://127.0.0.1:8088   fill a running Tipsarr with accounts, requests, issues ...
package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"regexp"
	"strconv"
)

func serve(addr, name string, h http.Handler) {
	go func() {
		log.Printf("%-12s http://127.0.0.1%s", name, addr)
		if err := http.ListenAndServe(addr, h); err != nil {
			log.Fatalf("%s: %v", name, err)
		}
	}()
}

// the box-office source: the test fixture for the latest weekends, an empty page for the rest
func newBoxOffice(fixture string) http.Handler {
	page, err := os.ReadFile(fixture)
	if err != nil {
		log.Fatalf("box office fixture: %v", err)
	}
	re := regexp.MustCompile(`^/weekend/2026W(\d\d)/`)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if m := re.FindStringSubmatch(r.URL.Path); m != nil {
			if n, _ := strconv.Atoi(m[1]); n >= 20 && n <= 40 {
				_, _ = w.Write(page)
				return
			}
		}
		_, _ = w.Write([]byte("<html><body>no data yet</body></html>"))
	})
}

func main() {
	seedURL := flag.String("seed", "", "fill the Tipsarr server at this address with accounts, requests and issues, then exit")
	fixture := flag.String("boxoffice-fixture", "internal/boxoffice/testdata/weekend.html", "box-office chart page to serve")
	flag.Parse()
	if *seedURL != "" {
		seed(*seedURL)
		return
	}
	serve(":18096", "jellyfin", newJellyfin())
	serve(":18097", "tmdb", newTMDB())
	serve(":18098", "tmdb images", newImages())
	serve(":18099", "radarr", newArr("radarr"))
	serve(":18100", "sonarr", newArr("sonarr"))
	serve(":18101", "box office", newBoxOffice(*fixture))
	serve(":18102", "resolver", newFakeResolver())
	fmt.Println("fake services running; Ctrl-C to stop")
	select {}
}
