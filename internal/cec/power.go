package cec

import (
    "fmt"
    "os/exec"
)

// PowerOffTV sends a "standby" command to the TV (logical address 0).
func PowerOffTV() error {
    cmdString := `echo "standby 0" | cec-client -s -d 1`
    cmd := exec.Command("sh", "-c", cmdString)
    return cmd.Run()
}

// PowerOnTV attempts to turn the TV on by sending "on 0" over CEC.
func PowerOnTV() error {
    cmdString := `echo "on 0" | cec-client -s -d 1`
    cmd := exec.Command("sh", "-c", cmdString)
    return cmd.Run()
}

// SwitchToHDMI sends an "Active Source" command based on your hdmiInput (1, 2, etc.).
// See the cec-client spec for physical address codes. For example, "2.0.0.0" = 20:00 in hex.
func SwitchToHDMI(input int) error {
    // For simplicity, assume input=1 => "10:00", input=2 => "20:00", etc.
    var address string
    switch input {
    case 1:
        address = "10:00"
    case 2:
        address = "20:00"
    default:
        address = "10:00" // fallback
    }

    cmdString := fmt.Sprintf(`echo "tx 1F:82:%s" | cec-client -s -d 1`, address)
    cmd := exec.Command("sh", "-c", cmdString)
    return cmd.Run()
}
