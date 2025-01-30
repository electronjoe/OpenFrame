package cec

import (
    "bufio"
    "log"
    "os/exec"
    "regexp"
    "strings"
)

// RemoteCommand is a simple enum for recognized CEC button presses.
type RemoteCommand int

const (
    RemoteUnknown RemoteCommand = iota
    RemoteLeft
    RemoteRight
    RemoteSelect
)

// We’ll capture user-control-pressed lines like: ">> 04:44:03" (where 03 is the key code)
// Key codes mapped to user-friendly names:
var cecUserControlMap = map[string]RemoteCommand{
    "03": RemoteLeft,   // "Left"
    "04": RemoteRight,  // "Right"
    "00": RemoteSelect, // "Select/Enter"
    // Add more if needed...
}

var reUserControlPressed = regexp.MustCompile(`>>\s+([0-9A-Fa-f]{2}):44:([0-9A-Fa-f]{2})`)

// StartCECListener spawns cec-client in a goroutine, parses its output,
// and sends recognized remote commands into remoteEvents.
func StartCECListener(remoteEvents chan<- RemoteCommand) {
    go func() {
        defer func() {
            log.Println("CEC listener goroutine exiting.")
        }()

        // Start cec-client in traffic mode:
        cmd := exec.Command("cec-client", "-t", "p", "-d", "8")

        stdout, err := cmd.StdoutPipe()
        if err != nil {
            log.Printf("Error getting stdout pipe: %v", err)
            return
        }
        defer stdout.Close()

        if err := cmd.Start(); err != nil {
            log.Printf("Failed to start cec-client: %v", err)
            return
        }

        scanner := bufio.NewScanner(stdout)
        for scanner.Scan() {
            line := scanner.Text()
            // Look for "User Control Pressed" lines
            if match := reUserControlPressed.FindStringSubmatch(line); len(match) == 3 {
                keyCode := strings.ToUpper(match[2]) // e.g., "03"
                cmdVal, ok := cecUserControlMap[keyCode]
                if !ok {
                    cmdVal = RemoteUnknown
                }
                if cmdVal != RemoteUnknown {
                    remoteEvents <- cmdVal
                }
            }
        }

        if err := scanner.Err(); err != nil {
            log.Printf("Scanner error: %v", err)
        }

        // cec-client exit code:
        if err := cmd.Wait(); err != nil {
            log.Printf("cec-client ended with error: %v", err)
        }
    }()
}
