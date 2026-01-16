package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	evdev "github.com/gvalkov/golang-evdev"
)

const (
	defaultMatch        = "osmc remote controller"
	osmcVendor   uint16 = 0x2017
	osmcProduct  uint16 = 0x1690
	pollInterval        = 5 * time.Millisecond
)

var buttonLabels = map[uint16]string{
	evdev.KEY_LEFT:        "LEFT",
	evdev.KEY_RIGHT:       "RIGHT",
	evdev.KEY_UP:          "UP",
	evdev.KEY_DOWN:        "DOWN",
	evdev.KEY_ENTER:       "OK",
	evdev.KEY_OK:          "OK",
	evdev.KEY_HOME:        "HOME",
	evdev.KEY_BACK:        "BACK",
	evdev.KEY_BACKSPACE:   "BACK",
	evdev.KEY_ESC:         "BACK",
	evdev.KEY_MENU:        "MENU",
	evdev.KEY_INFO:        "INFO",
	evdev.KEY_PLAYPAUSE:   "PLAY_PAUSE",
	evdev.KEY_PLAY:        "PLAY",
	evdev.KEY_PAUSE:       "PAUSE",
	evdev.KEY_STOP:        "STOP",
	evdev.KEY_FASTFORWARD: "FAST_FORWARD",
	evdev.KEY_REWIND:      "REWIND",
}

func main() {
	matchFlag := flag.String("match", defaultMatch, "case-insensitive substring used to select /dev/input/event* nodes")
	grabFlag := flag.Bool("grab", false, "attempt to exclusively grab each matching device")
	listFlag := flag.Bool("list", false, "list matching devices and exit")
	flag.Parse()

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	devices, err := findDevices(*matchFlag)
	if err != nil {
		log.Fatalf("scan failed: %v", err)
	}
	if len(devices) == 0 {
		log.Fatalf("no devices matched %q (vendor 0x%04x product 0x%04x fallback)", *matchFlag, osmcVendor, osmcProduct)
	}

	if *listFlag {
		for _, dev := range devices {
			fmt.Printf("%s\t%s\t(bus=0x%04x vendor=0x%04x product=0x%04x)\n", dev.Fn, dev.Name, dev.Bustype, dev.Vendor, dev.Product)
			dev.File.Close()
		}
		return
	}

	defer closeDevices(devices)

	if *grabFlag {
		for _, dev := range devices {
			if err := dev.Grab(); err != nil {
				log.Printf("warn: unable to grab %s: %v", dev.Fn, err)
			}
		}
		defer releaseDevices(devices)
	}

	for _, dev := range devices {
		log.Printf("listening on %s (%s)", dev.Fn, dev.Name)
	}

	if err := readLoop(ctx, devices, onButton); err != nil {
		log.Fatal(err)
	}
}

func findDevices(match string) ([]*evdev.InputDevice, error) {
	matchLower := strings.ToLower(strings.TrimSpace(match))

	candidates, err := filepath.Glob("/dev/input/event*")
	if err != nil {
		return nil, err
	}

	var devices []*evdev.InputDevice
	for _, path := range candidates {
		dev, err := evdev.Open(path)
		if err != nil {
			continue
		}

		if matchLower != "" {
			if strings.Contains(strings.ToLower(dev.Name), matchLower) {
				// matched on name
			} else if dev.Vendor == osmcVendor && dev.Product == osmcProduct {
				// matched on vendor/product fallback
			} else {
				dev.File.Close()
				continue
			}
		} else if dev.Vendor != osmcVendor || dev.Product != osmcProduct {
			dev.File.Close()
			continue
		}

		if err := syscall.SetNonblock(int(dev.File.Fd()), true); err != nil {
			dev.File.Close()
			continue
		}
		devices = append(devices, dev)
	}

	return devices, nil
}

func readLoop(ctx context.Context, devices []*evdev.InputDevice, handler func(string)) error {
	for {
		select {
		case <-ctx.Done():
			return nil
		default:
		}

		idle := true
		for _, dev := range devices {
			events, err := dev.Read()
			if err != nil {
				if errors.Is(err, syscall.EAGAIN) {
					continue
				}
				return fmt.Errorf("read %s: %w", dev.Fn, err)
			}
			if len(events) > 0 {
				idle = false
			}
			for _, event := range events {
				if event.Type != evdev.EV_KEY || event.Value != 1 {
					continue
				}
				if name, ok := buttonLabels[event.Code]; ok {
					handler(name)
				}
			}
		}

		if idle {
			time.Sleep(pollInterval)
		}
	}
}

func onButton(name string) {
	fmt.Println("BUTTON:", name)
}

func closeDevices(devices []*evdev.InputDevice) {
	for _, dev := range devices {
		dev.File.Close()
	}
}

func releaseDevices(devices []*evdev.InputDevice) {
	for _, dev := range devices {
		if err := dev.Release(); err != nil {
			log.Printf("warn: release %s: %v", dev.Fn, err)
		}
	}
}
