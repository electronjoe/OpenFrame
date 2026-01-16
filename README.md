# OpenFrame

## Configuration

### Album Config (`config.json`)

The openframe service reads its configuration from `~/.openframe/config.json` on the CM5. An example config is provided in `config/config.json`:

```json
{
  "albums": [
    "/path/to/album1",
    "/path/to/album2"
  ],
  "dateOverlay": true,
  "locationOverlay": false,
  "schedule": {
    "onTime": "06:00",
    "offTime": "20:00"
  },
  "interval": 30,
  "hdmiInput": 2,
  "randomize": true
}
```

| Field | Description |
|-------|-------------|
| `albums` | List of directory paths containing photos |
| `dateOverlay` | Show photo date on screen |
| `locationOverlay` | Show photo location on screen |
| `schedule.onTime` | Time to turn display on (HH:MM) |
| `schedule.offTime` | Time to turn display off (HH:MM) |
| `interval` | Seconds between photo transitions |
| `hdmiInput` | HDMI input number to switch to |
| `randomize` | Shuffle photo order |

### System Dependencies

I'm certainly missing others... but here is a start.

```
sudo apt-get update
sudo apt-get install cec-utils
```

### Build source

Right now I think this assumes the build is in the source repo as `main` binary - that should be changed =D

### Systemd


#### Strategy

Instead of letting the Go code decide when to power the TV on/off, we use **systemd** timers plus a small sync service to:

- Trigger `openframe-sync.service` at 06:00 and 20:00.
- The sync service decides whether to start or stop `openframe.service` based on current local time.
- `openframe.service` contains the CEC pre/post hooks to power on, select HDMI, and power off cleanly.
  - You can adjust the window by editing `linux/openframe-sync.service` and changing `OPENFRAME_START_HHMM` / `OPENFRAME_STOP_HHMM`.

Because the timers are `Persistent=true`, if the Pi is powered on after a scheduled time, the sync service runs immediately at boot and reconciles the display state appropriately (turn on if during hours, otherwise defensively power off).

#### Why Use systemd Timers?

- systemd has a built‐in notion of starting services on a schedule using `.timer` units (similar to cron, but more robust).  
- We can define one timer to start at 06:00 (which powers on the TV, sets input to HDMI 2, and starts the slideshow) and another timer at 20:00 (which stops the slideshow and then powers the TV off).

## Enable and Test the Timers

1. **Copy** all the `.service` and `.timer` files to either your user systemd location (`~/.config/systemd/user/`) or the system‐wide location (`/etc/systemd/system/`).  
2. **Enable** and **start** the timers so they run every day (and at boot for missed runs):
   ```bash
   # If using user-level systemd:
   systemctl --user daemon-reload
   
   systemctl --user enable openframe-start.timer
   systemctl --user enable openframe-stop.timer
   
   systemctl --user start openframe-start.timer
   systemctl --user start openframe-stop.timer
   ```
   *(If you’re using system‐wide units, omit `--user` and run as root.)*

3. Check status:
   ```bash
   systemctl --user status openframe-start.timer
   systemctl --user status openframe-stop.timer
   ```
4. Manually test:
   - Force a reconciliation now (regardless of the clock):
     ```bash
     systemctl --user start openframe-sync.service
     ```
   - To simulate “on” hours, temporarily set env vars and run the script directly:
     ```bash
     OPENFRAME_START_HHMM=00:00 OPENFRAME_STOP_HHMM=23:59 bash ~/OpenFrame/linux/openframe-sync.sh
     ```
   - To simulate “off” hours:
     ```bash
     OPENFRAME_START_HHMM=23:59 OPENFRAME_STOP_HHMM=23:59 bash ~/OpenFrame/linux/openframe-sync.sh
     ```

Once verified, systemd will automatically do these every day at 06:00 (start) and 20:00 (stop).
