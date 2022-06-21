package main

import (
	"archive/tools"
	"encoding/json"
	"fmt"
	"github.com/getlantern/systray"
	"github.com/getlantern/systray/example/icon"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
)

type Source struct {
	Src string `json:"src"`
	Dst string `json:"dst"`
}

type Answer struct {
	Message string
	Body    string
}

func rmTmpDir(dir string) error {
	var err error
	var files []os.FileInfo

	if files, err = ioutil.ReadDir(dir); err != nil {
		return err
	}

	for _, file := range files {
		src := path.Join(dir, file.Name())

		if file.IsDir() {
			if strings.HasPrefix(file.Name(), "_") {
				if err = os.RemoveAll(src); err != nil {
					return err
				}
			} else {
				rmTmpDir(src)
			}
		}
	}

	return nil
}

func handler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		var answer = Answer{Message: "Ready"}

		r, err := json.Marshal(answer)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
		}

		w.WriteHeader(http.StatusOK)
		w.Write(r)
	case "POST":
		var source Source
		var err error
		var answer Answer
		var res []byte

		decoder := json.NewDecoder(r.Body)

		// Get decoded json from request
		err = decoder.Decode(&source)
		if err != nil {
			answer.Message = "Decoding params error."
			answer.Body = err.Error()

			res, err = json.Marshal(answer)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
			}

			w.WriteHeader(http.StatusBadRequest)
			w.Write(res)
			return
		}

		// Check if src directory is existed
		_, err = os.Stat(filepath.FromSlash(source.Src))
		if os.IsNotExist(err) {
			answer.Message = "Src directory not exist."
			answer.Body = err.Error()

			res, err = json.Marshal(answer)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
			}

			w.WriteHeader(http.StatusBadRequest)
			w.Write(res)
			return
		}

		// Check if dst directory existed
		_, err = os.Stat(filepath.FromSlash(source.Dst))
		if os.IsExist(err) {
			// If exist we need remove src directory
			err = os.RemoveAll(filepath.FromSlash(source.Src))
			if err != nil {
				answer.Message = "Remove dst directory error."
				answer.Body = err.Error()

				res, err = json.Marshal(answer)
				if err != nil {
					http.Error(w, err.Error(), http.StatusBadRequest)
				}

				w.WriteHeader(http.StatusBadRequest)
				w.Write(res)
				return
			}
		}

		// Search directories like _* and delete them
		err = rmTmpDir(filepath.FromSlash(source.Src))
		if err != nil {
			answer.Message = "Remove tmp directory error."
			answer.Body = err.Error()

			res, err = json.Marshal(answer)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
			}

			w.WriteHeader(http.StatusBadRequest)
			w.Write(res)
			return
		}

		// Copy from src to dst
		err = tools.CopyDir(filepath.FromSlash(source.Src), filepath.FromSlash(source.Dst))
		if err != nil {
			// If error we need remove all copied files
			err := os.RemoveAll(filepath.FromSlash(source.Dst))
			if err != nil {
				answer.Message = "Copy error. Can not delete dst directory"
				answer.Body = err.Error()

				res, err = json.Marshal(answer)
				if err != nil {
					http.Error(w, err.Error(), http.StatusBadRequest)
				}

				w.WriteHeader(http.StatusBadRequest)
				w.Write(res)
				return
			}

			answer.Message = "Copy error. Can not copy src directory"
			answer.Body = err.Error()

			res, err = json.Marshal(answer)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
			}

			w.WriteHeader(http.StatusBadRequest)
			w.Write(res)
			return
		}

		//c := exec.Command("openfiles", "/disconnect", "/a", "*", "/op", "\"" + filepath.FromSlash(source.Src) + "\"")
		//var out bytes.Buffer
		//var stderr bytes.Buffer
		//c.Stdout = &out
		//c.Stderr = &stderr
		//if err := c.Run(); err != nil {
		//	w.WriteHeader(http.StatusBadRequest)
		//	w.Write([]byte(fmt.Sprint(err) + ": " + stderr.String()))
		//	return
		//}

		err = os.RemoveAll(filepath.FromSlash(source.Src))
		if err != nil {
			answer.Message = "Remove error. Can not remove src directory after coping"
			answer.Body = err.Error()

			res, err = json.Marshal(answer)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
			}

			w.WriteHeader(http.StatusBadRequest)
			w.Write(res)
			return
		}

		answer.Message = "Directory is moved"
		answer.Body = strconv.FormatInt(tools.DirSize(filepath.FromSlash(source.Dst)), 10)

		res, err = json.Marshal(answer)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		w.WriteHeader(http.StatusOK)
		w.Write(res)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
		fmt.Fprintf(w, "Method not allowed")
	}
}

func OnReady() {
	systray.SetIcon(icon.Data)
	systray.SetTitle("Design Projects Archives")
	systray.SetTooltip("Design Projects archives listener")
	mQuit := systray.AddMenuItem("Close", "Close app")

	// Sets the icon of a menu item. Only available on Mac and Windows.
	//mQuit.SetIcon(icon.Data)
	go func() {
		<-mQuit.ClickedCh
		systray.Quit()
		fmt.Println("Finished quitting")
	}()

	http.HandleFunc("/", handler)
	http.HandleFunc("/size", handleSize)
	http.ListenAndServe(":8888", nil)
}

func handleSize(writer http.ResponseWriter, request *http.Request) {
	var src Source
	var answer Answer
	var res []byte

	dec := json.NewDecoder(request.Body)
	if err := dec.Decode(&src); err != nil {
		answer.Body = "Decoding params error."
		answer.Message = err.Error()

		res, err = json.Marshal(answer)
		if err != nil {
			http.Error(writer, err.Error(), http.StatusBadRequest)
		}

		writer.WriteHeader(http.StatusBadRequest)
		writer.Write(res)
		return
	}

	answer.Message = "Directory size."
	answer.Body = strconv.FormatInt(tools.DirSize(filepath.FromSlash(src.Src)), 10)

	res, err := json.Marshal(answer)
	if err != nil {
		http.Error(writer, err.Error(), http.StatusBadRequest)
		return
	}

	writer.WriteHeader(http.StatusOK)
	writer.Write(res)
	return
}

func OnExit() {
	fmt.Println("Goodbye")
}

func main() {
	systray.Run(OnReady, OnExit)
}
