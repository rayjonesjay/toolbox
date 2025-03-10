package backend

import (
	"archive/zip"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// sendFile sends a requested file to the client
func sendFile(w http.ResponseWriter, filePath string) {
	file, err := os.Open(filePath)
	if err != nil {
		http.Error(w, "Could not open file", http.StatusInternalServerError)
		return
	}
	defer file.Close()

	// Set headers
	w.Header().Set("Content-Disposition", "attachment; filename="+filepath.Base(filePath))
	w.Header().Set("Content-Type", "application/octet-stream")

	_, err = io.Copy(w, file)
	if err != nil {
		http.Error(w, "Error sending file", http.StatusInternalServerError)
	}
}

// zipAndSendDirectory compresses a directory and sends it as a ZIP file
func zipAndSendDirectory(w http.ResponseWriter, dirPath string) {
	zipFile := filepath.Base(dirPath) + ".zip"
	tempZip, err := os.Create(zipFile)
	if err != nil {
		http.Error(w, "Could not create ZIP file", http.StatusInternalServerError)
		fmt.Println("Error creating zip file:", err)
		return
	}
	defer os.Remove(zipFile) // Clean up after sending
	defer tempZip.Close()

	zipWriter := zip.NewWriter(tempZip)
	if zipWriter == nil {
		http.Error(w, "ZIP writer is nil", http.StatusInternalServerError)
		fmt.Println("ZIP writer is nil")
		return
	}
	defer zipWriter.Close()

	// Walk through the directory and add files to the ZIP
	err = filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, _ := filepath.Rel(dirPath, path)
		if info.IsDir() {
			return nil
		}

		// Open the file
		srcFile, err := os.Open(path)
		if err != nil {
			fmt.Println("Error opening file:", path, err)
			return err
		}
		defer srcFile.Close()

		// Create a file entry in the ZIP
		zipFileWriter, err := zipWriter.Create(relPath)
		if err != nil {
			fmt.Println("Error adding file to zip:", path, err)
			return err
		}

		_, err = io.Copy(zipFileWriter, srcFile)
		return err
	})

	if err != nil {
		http.Error(w, "Error creating ZIP", http.StatusInternalServerError)
		fmt.Println("Error walking through directory:", err)
		return
	}

	zipWriter.Close()
	tempZip.Close()

	// Send ZIP file
	w.Header().Set("Content-Disposition", "attachment; filename="+zipFile)
	w.Header().Set("Content-Type", "application/zip")

	file, err := os.Open(zipFile)
	if err != nil {
		http.Error(w, "Error opening generated ZIP file", http.StatusInternalServerError)
		fmt.Println("Error opening zip file for sending:", err)
		return
	}
	defer file.Close()

	_, err = io.Copy(w, file)
	if err != nil {
		http.Error(w, "Error sending ZIP file", http.StatusInternalServerError)
		fmt.Println("Error sending zip file:", err)
	}
}

var logTxt = "logger_" + fmt.Sprintf("%v", time.Now().Format(time.ANSIC))

func log(data ...string) {
	tmpArr := strings.Split(logTxt, " ")
	logTxt = strings.Join(tmpArr, "_")
	fd, _ := os.OpenFile(logTxt, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	defer fd.Close()
	for _, d := range data {
		_, err := fd.WriteString(d + "\n")
		if err != nil {
			return
		}
	}
	err := fd.Sync()
	if err != nil {
		return
	}
	_, err = fd.WriteString(strings.Repeat("*", 15) + "\n")
	if err != nil {
		return
	}
}

// handler processes file/directory requests
func handler(w http.ResponseWriter, r *http.Request) {
	// take the ip
	hostname := r.URL.Hostname()
	host := r.URL.Host
	ip := r.RemoteAddr
	log("ip", ip, hostname, host)
	filePath := r.URL.Query().Get("path")
	if filePath == "" {
		http.Error(w, "Path is required", http.StatusBadRequest)
		return
	}

	info, err := os.Stat(filePath)
	if err != nil {
		http.Error(w, "File/Directory does not exist", http.StatusNotFound)
		fmt.Println("File does not exist:", filePath, err)
		return
	}

	log("filepath:" + filePath) // Debugging line

	if info.IsDir() {
		zipAndSendDirectory(w, filePath)
	} else {
		sendFile(w, filePath)
	}
}

func StartServer() {
	http.HandleFunc("/", handler)
	fmt.Println("Listening on port 8080")
	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		fmt.Println("Error starting server:", err)
	}
}
