package main

import (
  "flag"
  "log"
  "os"
  "os/exec"
)

func main() {
  log.SetFlags(0)

  dir := flag.String("dir", "", "Directory to run in")
  scriptPath := flag.String("script", "", "Path to script")
  flag.Parse()
  
  if *scriptPath == "" {
    log.Fatal("must provide script path")
  }

  cmd := exec.Command("bash", *scriptPath)
  cmd.Dir = *dir
  cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
  if err := cmd.Run(); err != nil {
    if exitErr, ok := err.(*exec.ExitError); ok {
      if len(exitErr.Stderr) != 0 {
        log.Printf("%s", exitErr.Stderr)
      }
      os.Exit(exitErr.ExitCode())
    } else {
      log.Printf("error running command: %v", err)
    }
  }
}
