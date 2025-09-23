package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"os/exec"
	"regexp"
	"strings"
	"time"

	"github.com/electronjoe/OpenFrame/internal/cec"
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
	hdmiInput := flag.Int("hdmi", 2, "HDMI input number to activate before listening (<=0 skips the switch).")
	skipPower := flag.Bool("skip-power", false, "Skip sending the TV power on command before listening.")
	powerOnDelay := flag.Duration("power-delay", 10*time.Second, "Delay after powering on the TV before switching inputs.")
	inputDelay := flag.Duration("input-delay", 5*time.Second, "Delay after switching HDMI inputs before starting cec-client.")

	flag.Parse()

	if !*skipPower {
		fmt.Println("Sending TV power on command via CEC.")
		if err := cec.PowerOnTV(); err != nil {
			log.Printf("PowerOnTV failed: %v", err)
		} else if *powerOnDelay > 0 {
			delay := *powerOnDelay
			fmt.Printf("Waiting %s for the TV to wake up...\n", delay)
			time.Sleep(delay)
		}
	} else {
		fmt.Println("Skipping TV power on step.")
	}

	if *hdmiInput > 0 {
		fmt.Printf("Switching TV to HDMI input %d via CEC.\n", *hdmiInput)
		if err := cec.SwitchToHDMI(*hdmiInput); err != nil {
			log.Printf("SwitchToHDMI failed: %v", err)
		} else if *inputDelay > 0 {
			delay := *inputDelay
			fmt.Printf("Waiting %s for the HDMI input to settle...\n", delay)
			time.Sleep(delay)
		}
	} else {
		fmt.Println("Skipping HDMI input switch step.")
	}

	fmt.Println("Starting cec-client in traffic mode; listening for user control pressed/released.")

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
