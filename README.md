# OpenFrame

## Required Config

I'm certainly missing others... but here is a start.

```
sudo apt-get update
sudo apt-get install cec-utils
```

### Build source

Right now I think this assumes the build is in the source repo as `main` binary - that should be changed =D

### Systemd


#### Strategy

Instead of letting the Go code decide when to power the TV on/off, we will use **systemd** to:

- **Start** our `openframe.service` (and power on the TV + set HDMI input) at 06:00 (6 AM).  
- **Stop** our `openframe.service` (and power off the TV) at 20:00 (8 PM).

Once `openframe.service` is running, it can continue its usual slideshow (without worrying about display removal). When systemd *stops* the service at 8 PM, we can run the CEC “standby” command to turn off the TV *after* stopping the Ebiten application—thus avoiding any crashes in Ebiten due to the removal of the active display.

#### Why Use systemd Timers?

- systemd has a built‐in notion of starting services on a schedule using `.timer` units (similar to cron, but more robust).  
- We can define one timer to start at 06:00 (which powers on the TV, sets input to HDMI 2, and starts the slideshow) and another timer at 20:00 (which stops the slideshow and then powers the TV off).

## Enable and Test the Timers

1. **Copy** all the `.service` and `.timer` files to either your user systemd location (`~/.config/systemd/user/`) or the system‐wide location (`/etc/systemd/system/`).  
2. **Enable** and **start** the timers so they run every day:
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
   - You can manually trigger a start (simulating 06:00) via  
     ```bash
     systemctl --user start openframe-start.service
     ```  
   - Check that the TV turns on and the slideshow appears.  
   - Then manually trigger a stop (simulating 20:00) via  
     ```bash
     systemctl --user start openframe-stop.service
     ```  
   - Check that the slideshow stops and the TV turns off.  

Once verified, systemd will automatically do these every day at 06:00 (start) and 20:00 (stop).
