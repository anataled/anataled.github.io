package main

import (
	"context"
	"flag"
	"html/template"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/fsnotify/fsnotify"
)

type content struct {
	Pages []struct {
		Title string `toml:"title"`
		Data  string `toml:"data"`
		Desc  string `toml:"desc"`
	} `toml:"pages"`
}

func build(out string) {
	start := time.Now()
	tmpl := make(map[string]*template.Template)
	tmpl["home"] = template.Must(template.ParseFiles("partials/head.html", "partials/nav.html", "partials/footer.html", "partials/script.html", "base/home.html"))
	tmpl["content"] = template.Must(template.ParseFiles("partials/head.html", "partials/nav.html", "partials/footer.html", "partials/script.html", "base/content.html"))

	idxFile, err := os.Create(filepath.Join(out, "index.html"))
	if err != nil {
		log.Fatalln(err)
	}

	tmpl["home"].ExecuteTemplate(idxFile, "home", nil)
	idxFile.Close()

	bs, err := os.ReadFile("config.toml")
	if err != nil {
		log.Fatalln(err)
	}

	var c content
	_, err = toml.Decode(string(bs), &c)
	if err != nil {
		log.Fatalln(err)
	}

	for _, page := range c.Pages {
		p := filepath.Join(out, strings.ToLower(strings.ReplaceAll(page.Title, " ", "_"))+".html")
		f, err := os.Create(p)
		if err != nil {
			log.Fatalln(page.Title, ":", err)
		}
		d := struct {
			Page struct {
				Title string "toml:\"title\""
				Data  string "toml:\"data\""
				Desc  string "toml:\"desc\""
			}
			Pages content
		}{
			page, c,
		}
		tmpl["content"].ExecuteTemplate(f, "content", d)
		f.Close()
	}
	log.Printf("finished rebuilding in %f seconds\n", time.Since(start).Seconds())
}

func main() {
	out := flag.String("out", "", "output dir")
	watch := flag.Bool("w", false, "watch directory")
	flag.Parse()

	if !*watch {
		build(*out)
		return
	}

	log.Println("watching directory...")

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}
	defer watcher.Close()

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	done := make(chan bool)
	go func() {
		for {
			select {
			case <-ctx.Done():
				done <- true
				return
			case _, ok := <-watcher.Events:
				if !ok {
					done <- true
					return
				}
				build(*out)
			case err, ok := <-watcher.Errors:
				if !ok {
					done <- true
					return
				}
				log.Println("error:", err)
			}
		}
	}()

	dirs := []string{"base", "partials"}

	for _, dir := range dirs {
		err = watcher.Add(dir)
		if err != nil {
			log.Fatal(err)
		}
	}
	<-done
}
