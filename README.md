# Post Viewer and Analyzer with Go

## Introduction
The Post Viewer and Analyzer is a web-based application built with Go. It serves a web interface that allows users to fetch posts from the JSONPlaceholder API, save these posts to a file, and perform a character frequency analysis on the saved data. This application demonstrates the use of Go for server-side web development, including handling HTTP requests, processing JSON, and rendering HTML templates.

## Features
- **Fetch Data**: Users can fetch posts from the external JSONPlaceholder API.
- **Save Data**: Automatically saves fetched posts into a local JSON file.
- **Analyze Data**: Performs a character frequency analysis on the contents of the saved JSON file.
- **Web Interface**: Simple and user-friendly web interface to interact with the application.

## Technology Stack
- **Go**: All server-side logic is implemented in Go, utilizing its standard library for web server functionality, file I/O, and concurrency.
- **HTML/CSS**: Front-end layout and styling.
- **JSONPlaceholder API**: External REST API used for fetching sample post data.

## Getting Started

### Prerequisites
- Go (version 1.14 or higher recommended)
- Internet connection (for fetching data from the external API)

### Installation
1. **Clone the repository:**
   ```
   git clone https://github.com/yourusername/post-viewer-analyzer.git
   cd post-viewer-analyzer
   ```

2. **Run the application:**
   ```
   go run main.go
   ```

### Usage
1. **Open your web browser.**
2. **Navigate to `http://localhost:8080/` to access the application.**
3. **Use the following endpoints to interact with the application:**
    - **Home Page**: `/`
    - **Fetch Posts**: `/fetch` - Fetches posts from the JSONPlaceholder and saves them to a local file.
    - **Analyze Character Frequency**: `/analyze` - Analyzes the frequency of each character in the saved posts.

## Application Structure
- **main.go**: Contains all the server-side logic including API calls, concurrency handling, file operations, and web server setup.
- **home.html**: HTML template file used for rendering the web interface.

## Contributing
Contributions are what make the open-source community such an amazing place to learn, inspire, and create. Any contributions you make are **greatly appreciated**.

1. Fork the Project
2. Create your Feature Branch (`git checkout -b feature/AmazingFeature`)
3. Commit your Changes (`git commit -m 'Add some AmazingFeature'`)
4. Push to the Branch (`git push origin feature/AmazingFeature`)
5. Open a Pull Request

## License
Distributed under the MIT License. See `LICENSE` for more information.

## Contact
Son Nguyen - [https://github.com/hoangsonww](https://github.com/hoangsonww)  

## Acknowledgements
- [Go](https://golang.org/)
- [JSONPlaceholder](https://jsonplaceholder.typicode.com/)

---
