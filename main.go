package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"os"
	"strings"
)

type Video struct {
	Name      string `json:"name"`
	Completed bool   `json:"completed"`
}

var videos []Video
var dataFile = "progress.json"

func main() {
	loadVideos()
	loadProgress()

	http.HandleFunc("/", handleHome)
	http.HandleFunc("/toggle", handleToggle)
	http.Handle("/videos/", http.StripPrefix("/videos/", http.FileServer(http.Dir("videos"))))

	fmt.Println("Server is running on http://localhost:8080")
	http.ListenAndServe(":8080", nil)
}

func loadVideos() {
	entries, err := os.ReadDir("videos")
	if err != nil {
		fmt.Println("Error reading directory:", err)
		return
	}

	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(strings.ToLower(entry.Name()), ".mp4") {
			videos = append(videos, Video{Name: entry.Name(), Completed: false})
		}
	}
}

func loadProgress() {
	data, err := os.ReadFile(dataFile)
	if err != nil {
		if os.IsNotExist(err) {
			return
		}
		fmt.Println("Error reading progress file:", err)
		return
	}

	var savedVideos []Video
	err = json.Unmarshal(data, &savedVideos)
	if err != nil {
		fmt.Println("Error unmarshaling progress data:", err)
		return
	}

	for i, video := range videos {
		for _, savedVideo := range savedVideos {
			if video.Name == savedVideo.Name {
				videos[i].Completed = savedVideo.Completed
				break
			}
		}
	}
}

func saveProgress() {
	data, err := json.Marshal(videos)
	if err != nil {
		fmt.Println("Error marshaling progress data:", err)
		return
	}

	err = os.WriteFile(dataFile, data, 0644)
	if err != nil {
		fmt.Println("Error writing progress file:", err)
	}
}

func handleHome(w http.ResponseWriter, r *http.Request) {
	tmpl := `
<!DOCTYPE html>
<html>
<head>
	<title>Video Tutorials</title>
	<style>
		body { display: flex; font-family: Arial, sans-serif; }
		#video-player { flex: 2; }
		#video-list { flex: 1; padding-inline: 20px; height:100vh; overflow:auto; }
		.video-item { margin-bottom: 10px; }
	</style>
</head>
<body>
	<div id="video-player">
		<video id="player" width="100%" controls>
			<source src="" type="video/mp4">
		</video>
	</div>
	<div id="video-list">
		<h2>Video Tutorials</h2>
		{{range .}}
		<div class="video-item">
			<input id="id{{.Name}}" type="checkbox" onchange="toggleCompleted('{{.Name}}')" {{if .Completed}}checked{{end}}>
			<a href="#" onclick="playVideo('{{.Name}}'); return false;">{{.Name}}</a>
		</div>
		{{end}}
	</div>
	<script>
		function playVideo(name) {
			const player = document.getElementById('player');
			player.src = '/videos/'+name;

			player.onended = function() {
				toggleCompleted(name);
				 const checkbox = document.getElementById('id'+name);
				  if (checkbox) {
					  checkbox.checked = true;
				  }
			};
		}
		function toggleCompleted(name) {
			fetch('/toggle', {
				method: 'POST',
				headers: {'Content-Type': 'application/x-www-form-urlencoded'},
				body: 'name=' + encodeURIComponent(name)
			}).then(response => response.json())
			  .then(data => console.log(data));
		}
	</script>
</body>
</html>
`
	t, err := template.New("home").Parse(tmpl)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	err = t.Execute(w, videos)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func handleToggle(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	name := r.FormValue("name")
	for i, video := range videos {
		if video.Name == name {
			videos[i].Completed = !videos[i].Completed
			break
		}
	}

	saveProgress()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "success"})
}
