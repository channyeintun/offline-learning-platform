package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

type Video struct {
	Name      string `json:"name"`
	Path      string `json:"path"`
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
	videos = []Video{}
	err := filepath.Walk("videos", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.HasSuffix(strings.ToLower(info.Name()), ".mp4") {

			relPath, err := filepath.Rel("videos", path)
			if err != nil {
				return err
			}

			relPath = filepath.ToSlash(relPath)
			videos = append(videos, Video{
				Name:      info.Name(),
				Path:      relPath,
				Completed: false,
			})
		}
		return nil
	})
	if err != nil {
		fmt.Println("Error walking through directory:", err)
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
			if video.Path == savedVideo.Path {
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
	videosJSON, err := json.Marshal(videos)
	if err != nil {
		http.Error(w, "Failed to encode videos", http.StatusInternalServerError)
		return
	}
	tmpl := `<!DOCTYPE html>
<html>
<head>
	<title>Video Tutorials</title>
	<style>
		body { display: flex; font-family: Arial, sans-serif; }
		#video-player { flex: 2; }
		#video-list { flex: 1; padding-inline: 20px; height:100vh; overflow:auto; }
		.video-item { display:flex; gap:8px; padding:8px 16px; }
		.video-item:hover { background:#d1d7dc; }
		.section-title { background:#f7f9fa; padding:1.6rem; }
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
		
	</div>
	<script>
		const videos = {{.VideosJSON}};
		function playVideo(path) {
			const player = document.getElementById('player');
			player.src = '/videos/'+path;

			player.onended = function() {
				toggleCompleted(path);
				const checkbox = document.getElementById('id'+path);
				if (checkbox) {
					checkbox.checked = true;
				}
			};
		}
		function toggleCompleted(path) {
			fetch('/toggle', {
				method: 'POST',
				headers: {'Content-Type': 'application/x-www-form-urlencoded'},
				body: 'path=' + encodeURIComponent(path)
			}).then(response => response.json())
			  .then(data => console.log(data));
		}

		function renderVideos(videos) {
			const sections = {};
			videos.forEach(video => {
				const splited = video.path.split('/'); 
				if(splited.length === 2){
					if (!sections[sectionTitle]) {
						sections[sectionTitle] = [];
					}
					sections[sectionTitle].push(video);
				}
			});

			const videoList = document.getElementById('video-list');

			// Clear existing content
			videoList.innerHTML = '<h2>Video Tutorials</h2>';

			if(sections.length>0){
				for (const section in sections) {
					const sectionDiv = document.createElement('div');
					sectionDiv.className = 'section';

					const sectionTitle = document.createElement('div');
					sectionTitle.style["margin-block-end"]="10px";
					sectionTitle.className = 'section-title';
					sectionTitle.innerText = section.replace(/^\d+-/, ''); // Remove leading numbers
					sectionTitle.onclick = function() {
						const content = sectionDiv.querySelector('.section-content');
						content.style.display = (content.style.display === 'block') ? 'none' : 'block';
					};

					sectionDiv.appendChild(sectionTitle);

					const contentDiv = document.createElement('div');
					contentDiv.style.display = 'none';
					contentDiv.className = 'section-content';

					sections[section].forEach(video => {
						const videoItem = document.createElement('div');
						videoItem.className = 'video-item';

						videoItem.innerHTML = 
							'<input id="id' + video.path + '" type="checkbox" onchange="toggleCompleted(\'' + video.path + '\')" ' + 
							(video.completed ? 'checked' : '') + '>';

						const link = document.createElement('a');
						link.textContent = video.name;
						link.href="#";
						link.onclick=function (){
							playVideo(video.path);
						}

						videoItem.appendChild(link);
						contentDiv.appendChild(videoItem);
					});

					sectionDiv.appendChild(contentDiv);
					videoList.appendChild(sectionDiv);
				}
			}else{
				videos.forEach(video=>{
					const videoItem = document.createElement('div');
					videoItem.className = 'video-item';

					videoItem.innerHTML = 
						'<input id="id' + video.path + '" type="checkbox" onchange="toggleCompleted(\'' + video.path + '\')" ' + 
						(video.completed ? 'checked' : '') + '>';

					const link = document.createElement('a');
					link.textContent = video.name;
					link.href="#";
					link.onclick=function (){
						playVideo(video.path);
					}

					videoItem.appendChild(link);
					videoList.appendChild(videoItem);
				})
			}
		}

		// Render videos when the document is ready
		document.addEventListener('DOMContentLoaded', () => {
			renderVideos(videos);
		});
	</script>
</body>
</html>`

	t, err := template.New("home").Parse(tmpl)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	err = t.Execute(w, struct {
		VideosJSON template.JS
	}{
		VideosJSON: template.JS(videosJSON),
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func handleToggle(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	path := r.FormValue("path")
	for i, video := range videos {
		if video.Path == path {
			videos[i].Completed = !videos[i].Completed
			break
		}
	}

	saveProgress()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "success"})
}
