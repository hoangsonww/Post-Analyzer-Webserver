package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"io/ioutil"
	"net/http"
	"os"
	"sync"
)

// Post struct to map the JSON data
type Post1 struct {
	UserId int    `json:"userId"`
	Id     int    `json:"id"`
	Title  string `json:"title"`
	Body   string `json:"body"`
}

// Template variables
type HomePageVars1 struct {
	Title       string
	Posts       []Post
	CharFreq    map[rune]int
	Error       string
	HasPosts    bool
	HasAnalysis bool
}

// Custom template functions
var funcMap1 = template.FuncMap{
	"toJSON": func(v interface{}) string {
		data, _ := json.Marshal(v)
		return string(data)
	},
}

var templates1 = template.Must(template.New("").Funcs(funcMap).ParseFiles("home.html"))

func main() {
	http.HandleFunc("/", HomeHandler)
	http.HandleFunc("/fetch", FetchPostsHandler)
	http.HandleFunc("/analyze", AnalyzePostsHandler)
	http.HandleFunc("/add", AddPostHandler)

	fmt.Println("Server starting at http://localhost:8080/")
	http.ListenAndServe(":8080", nil)
}

// HomeHandler serves the home page
func HomeHandler1(w http.ResponseWriter, r *http.Request) {
	posts, err := readPostsFromFile()
	if err != nil {
		renderTemplate(w, HomePageVars{Title: "Home", Error: "Failed to read posts: " + err.Error()})
		return
	}
	renderTemplate(w, HomePageVars{Title: "Home", Posts: posts, HasPosts: len(posts) > 0})
}

// FetchPostsHandler fetches posts and writes them to a file
func FetchPostsHandler1(w http.ResponseWriter, r *http.Request) {
	posts, err := fetchPosts()
	if err != nil {
		renderTemplate(w, HomePageVars{Title: "Error", Error: "Failed to fetch posts: " + err.Error()})
		return
	}

	if err := writePostsToFile(posts); err != nil {
		renderTemplate(w, HomePageVars{Title: "Error", Error: "Failed to write posts to file: " + err.Error()})
		return
	}

	renderTemplate(w, HomePageVars{Title: "Posts Fetched", Posts: posts, HasPosts: true})
}

// AnalyzePostsHandler reads the posts file and analyzes character frequency
func AnalyzePostsHandler1(w http.ResponseWriter, r *http.Request) {
	count, err := countCharacters("posts.json")
	if err != nil {
		renderTemplate(w, HomePageVars{Title: "Error", Error: "Failed to analyze posts: " + err.Error()})
		return
	}

	renderTemplate(w, HomePageVars{Title: "Character Analysis", CharFreq: count, HasAnalysis: true})
}

// AddPostHandler allows the user to add a new post
func AddPostHandler1(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		var post Post
		post.UserId = 1 // You can set this as needed
		post.Id = generatePostID()
		post.Title = r.FormValue("title")
		post.Body = r.FormValue("body")

		posts, err := readPostsFromFile()
		if err != nil {
			renderTemplate(w, HomePageVars{Title: "Error", Error: "Failed to read posts: " + err.Error()})
			return
		}

		posts = append(posts, post)

		if err := writePostsToFile(posts); err != nil {
			renderTemplate(w, HomePageVars{Title: "Error", Error: "Failed to write post to file: " + err.Error()})
			return
		}

		renderTemplate(w, HomePageVars{Title: "Post Added", Posts: posts, HasPosts: true})
	} else {
		renderTemplate(w, HomePageVars{Title: "Add New Post"})
	}
}

func fetchPosts1() ([]Post, error) {
	resp, err := http.Get("https://jsonplaceholder.typicode.com/posts")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var posts []Post
	if err := json.NewDecoder(resp.Body).Decode(&posts); err != nil {
		return nil, err
	}
	return posts, nil
}

func writePostsToFile1(posts []Post) error {
	file, err := os.Create("posts.json")
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	if err := encoder.Encode(posts); err != nil {
		return err
	}
	return nil
}

func readPostsFromFile1() ([]Post, error) {
	data, err := ioutil.ReadFile("posts.json")
	if err != nil {
		return nil, err
	}

	var posts []Post
	if err := json.Unmarshal(data, &posts); err != nil {
		return nil, err
	}
	return posts, nil
}

func countCharacters1(filePath string) (map[rune]int, error) {
	data, err := ioutil.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	charCount := make(map[rune]int)
	mu := sync.Mutex{}
	wg := sync.WaitGroup{}

	for _, byteValue := range string(data) {
		wg.Add(1)
		go func(c rune) {
			defer wg.Done()
			mu.Lock()
			charCount[c]++
			mu.Unlock()
		}(rune(byteValue))
	}

	wg.Wait()
	return charCount, nil
}

func renderTemplate1(w http.ResponseWriter, vars HomePageVars) {
	if err := templates.ExecuteTemplate(w, "home.html", vars); err != nil {
		http.Error(w, "Failed to render template", http.StatusInternalServerError)
	}
}

func generatePostID1() int {
	posts, _ := readPostsFromFile()
	maxID := 0
	for _, post := range posts {
		if post.Id > maxID {
			maxID = post.Id
		}
	}
	return maxID + 1
}
