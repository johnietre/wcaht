package main

import (
  "encoding/json"
  "errors"
  "flag"
  "fmt"
  "log"
  "net/http"
  urlpkg "net/url"
  "os"
  "path/filepath"
  "runtime"
  "sync"
)

var (
  thisDir string
  serversPath string
  // map[name]addr
  servers map[string]string
  serversLastUpdate int64
  serversMtx sync.RWMutex
)

func init() {
  _, thisFile, _, _ := runtime.Caller(0)
  thisDir = filepath.Dir(thisFile)
}

func main() {
  log.SetFlags(0)

  addr := flag.String("addr", "127.0.0.1:8000", "Address to serve on.")
  flag.StringVar(&serversPath, "servers-path", "./servers.json", "Path to JSON file containing server information. The JSON file should have map (object) with names as keys and URLs as values. The URL should be the entire address, including protocol. The names must be unique.")
  flag.Parse()

  http.Handle(
    "/static/",
    http.StripPrefix(
      "/static",
      http.FileServer(http.Dir(filepath.Join(thisDir, "static"))),
    ),
  )
  http.HandleFunc("/", homeHandler)
  http.HandleFunc("/servers", serversHandler)
  http.HandleFunc("/admin/servers", adminServersHandler)
  log.Printf("Listening on %s", *addr)
  log.Fatal(http.ListenAndServe(*addr, nil))
}

func homeHandler(w http.ResponseWriter, r *http.Request) {
  http.ServeFile(w, r, filepath.Join(thisDir, "index.html"))
}

func serversHandler(w http.ResponseWriter, r *http.Request) {
  serversMtx.RLock()
  json.NewEncoder(w).Encode(servers)
  serversMtx.RUnlock()
}

func adminServersHandler(w http.ResponseWriter, r *http.Request) {
  f, err := os.Open(serversPath)
  if err != nil {
    if errors.Is(err, os.ErrNotExist) {
      // TODO: Bad request?
      http.Error(w, "File does not exist", http.StatusNotFound)
    } else {
      log.Printf("error opening servers file: %v", err)
      http.Error(w, fmt.Sprintf("Internal server error: %v", err), http.StatusInternalServerError)
    }
    return
  }
  defer f.Close()
  // TODO: Just go straight to reading the file?
  info, err := f.Stat()
  if err != nil {
    log.Printf("error getting servers file info: %v", err)
    http.Error(w, fmt.Sprintf("Internal server error: %v", err), http.StatusInternalServerError)
    return
  }
  serversMtx.Lock()
  defer serversMtx.Unlock()
  timestamp := info.ModTime().Unix()
  if serversLastUpdate >= timestamp {
    return
  }
  serversLastUpdate = timestamp
  var srvrs map[string]string
  umtErr := &json.UnmarshalTypeError{}
  if err := json.NewDecoder(f).Decode(&srvrs); err != nil {
    if errors.As(err, &umtErr) {
      http.Error(w, "Malformed JSON", http.StatusBadRequest)
    } else {
      log.Printf("error reading file servers: %v", err)
      http.Error(w, fmt.Sprintf("Internal server error: %v", err), http.StatusInternalServerError)
    }
    return
  }
  errMsg := ""
  for name, url := range srvrs {
    if name == "" {
      errMsg += "Server must have name\n"
    } else if _, err := urlpkg.Parse(url); err != nil {
      errMsg += fmt.Sprintf("Bad URL (%s): %v\n", url, err)
    }
  }
  if errMsg != "" {
    http.Error(w, errMsg, http.StatusBadRequest)
    return
  }
  servers = srvrs
}
