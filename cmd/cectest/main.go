package main

import (
    "bufio"
    "fmt"
    "log"
    "os/exec"
    "regexp"
    "strings"
)

// Map of common HDMI-CEC user control codes to human-readable names.
var cecUserControlMap = map[string]string{
    "00": "Select/Enter",
    "01": "Up",
    "02": "Down",
    "03": "Left",
    "04": "Right",
    "0D": "Back",
    // Add more if needed...
}

// Regex for lines like: ">> 04:44:03"
var reUserControlPressed = regexp.MustCompile(`>>\s+([0-9A-Fa-f]{2}):44:([0-9A-Fa-f]{2})`)

// Regex for lines like: ">> 04:45"
var reUserControlReleased = regexp.MustCompile(`>>\s+([0-9A-Fa-f]{2}):45`)

func main() {
    fmt.Println("Starting cec-client in traffic mode; listening for user control pressed/released.\n")

    // cec-client options:
    //  -t p : Pretend to be a 'playback device'
    //  -d 8 : Set debug level to 8 (verbose). You can use -d 5 or -d 8 depending on your needs.
    cmd := exec.Command("cec-client", "-t", "p", "-d", "8")

    stdout, err := cmd.StdoutPipe()
    if err != nil {
        log.Fatalf("Error getting stdout pipe: %v", err)
    }
    defer stdout.Close()

    // (Optional) capture stderr if you want cec-client's error logs as well
    // stderr, err := cmd.StderrPipe()
    // if err != nil {
    //     log.Fatalf("Error getting stderr pipe: %v", err)
    // }
    // defer stderr.Close()

    // Start cec-client
    if err := cmd.Start(); err != nil {
        log.Fatalf("Failed to start cec-client: %v", err)
    }

    scanner := bufio.NewScanner(stdout)

    for scanner.Scan() {
        line := scanner.Text()
        // For debugging, you might do: fmt.Println(line)

        // Check for "User Control Pressed" matches
        if match := reUserControlPressed.FindStringSubmatch(line); len(match) == 3 {
            // match[1] = source device address (e.g. "04")
            // match[2] = key code (e.g. "03")
            sourceAddr := match[1]
            keyCode := strings.ToUpper(match[2])

            keyName, known := cecUserControlMap[keyCode]
            if !known {
                keyName = "Unknown Keycode " + keyCode
            }
            fmt.Printf("User Control Pressed from 0x%s: %s (0x%s)\n", sourceAddr, keyName, keyCode)
            continue
        }

        // Check for "User Control Released"
        if match := reUserControlReleased.FindStringSubmatch(line); len(match) == 2 {
            // match[1] = source device address
            sourceAddr := match[1]
            fmt.Printf("User Control Released from 0x%s\n", sourceAddr)
            continue
        }

        // Optionally, handle other traffic or debug lines if desired
        // ...
    }

    if err := scanner.Err(); err != nil {
        log.Printf("Scanner error reading cec-client output: %v", err)
    }

    // Wait for the cec-client command to finish
    if err := cmd.Wait(); err != nil {
        log.Printf("cec-client process ended with error: %v", err)
    }
}
