# **OpenFrame Design Document**  

---

## 1. Overview

OpenFrame is an open-source photo frame solution leveraging a Raspberry Pi 4 and the Samsung The Frame TV. It displays multiple local photo albums in chronological order, manages daily on/off schedules, and enables user navigation (skipping photos by year) via HDMI-CEC. This design doc outlines how the Golang application will be structured, how it will handle concurrency, how configuration is parsed, and how HDMI-CEC integration will work.

---

## 2. Goals & Non-Goals

### **Goals**

1. **Chronological Photo Display**:  
   - Merge photos from multiple albums into a single chronological stream.
   - Display images seamlessly with configurable intervals.
   - When two portrait-orientation images are consecutive, and the display allows, place them side-by-side in the same slide.
     Ensure they are centered horizontally on the display (so that the right edge of the left image
     is flush with the left edge of the right image), and use appropriate offsets so they appear
     visually balanced.

2. **Easy Configuration**:  
   - Load settings from a JSON config (e.g., `~/.openframe/config.json`).
   - Support toggling overlays (date/location), setting daily on/off times, and specifying local directories.
   - When two portrait photos are shown side-by-side, if overlays are enabled:
     - Display the metadata for the left photo at the bottom-left corner of that photo.
     - Display the metadata for the right photo at the bottom-right corner of that photo.
     - Continue to apply the same semi-transparent overlay style for each.

3. **HDMI-CEC Integration**:  
   - Power on/off the Samsung The Frame TV daily at specified times.
   - Automatically switch TV input to the Pi’s HDMI port.
   - Use remote navigation events to skip photos by year.

4. **Robust & Resource-Efficient**:  
   - Run continuously on Raspberry Pi 4 with minimal resource overhead.
   - Gracefully handle large image libraries and concurrency.

### **Non-Goals**

1. **Cloud Integration**:  
   - No direct uploading/downloading from cloud services (e.g., Google Photos) in this iteration.

2. **Complex Photo Editing**:  
   - No advanced photo editing features beyond simple overlays.

3. **Deep Remote Control Mapping**:  
   - We only support skipping forward/backward by year. Additional remote controls (e.g., next/previous photo) may be explored in the future.

---

## 3. Definitions and Abbreviations

- **CEC**: Consumer Electronics Control – a feature of HDMI for controlling devices.  
- **EXIF**: Exchangeable Image File Format – metadata embedded in images (timestamps, camera info, GPS location, etc.).  
- **Pi**: Refers to the Raspberry Pi 4 hardware.  
- **Frame**: Refers to the Samsung The Frame TV.

---

## 4. Architecture and System Components

Below is the high-level architecture of the OpenFrame system:

```
                          +--------------------+
                          | Samsung The Frame  |
                          |      (HDMI-CEC)    |
                          +--------------------+
                                    ^
                                    | (CEC commands)
                                    |
                 +------------------------------------+
                 |          Raspberry Pi 4            |
                 |    (Linux + Go OpenFrame App)      |
                 +------------------------------------+
                  |   ^                       |
                  |   | (Render images / CEC) |  
                  |   v                       |
             +-----------+             +---------------+
             |  Storage  |             |  config.json  |
             +-----------+             +---------------+
                (Local photos)          (User config)
```

### **Major Components**

1. **Go OpenFrame Binary**  
   - Responsible for reading config, scanning local photo directories, scheduling TV on/off events, and displaying photos with overlays.

2. **Photo Library**  
   - Fetched from local directories specified in `config.json`.
   - Indexed, sorted by date/time (EXIF or file creation date fallback).

3. **HDMI-CEC Module**  
   - Interacts with underlying CEC libraries (e.g., `libcec` or similar).  
   - Sends and receives commands to turn the TV on/off, switch inputs, and detect remote button presses.

4. **Renderer / Slideshow Engine**  
   - Handles image loading, scaling, overlay rendering, and displaying images on the connected TV.

5. **Scheduler**  
   - Manages daily on/off times, starts/stops the slideshow, sends power control commands.

---

## 5. Detailed Design

### 5.1 Configuration

**File Structure (Example)**  
```jsonc
{
  "albums": [
    "/home/pi/photos/family_2010s",
    "/home/pi/photos/vacations",
    "/media/external_drive/wedding_album"
  ],
  "dateOverlay": true,
  "locationOverlay": false,
  "schedule": {
    "onTime": "06:00",
    "offTime": "21:00"
  },
  "interval": 10,   // in seconds
  "hdmiInput": 1,
  "randomize": true, // randomize slide order
}
```

#### **Parsing**
- Use Go’s `encoding/json` to parse `~/.openframe/config.json`.
- Validate fields, apply defaults if optional fields are omitted (e.g., location overlay defaults to `false`).

### 5.2 Photo Indexing & Metadata

1. **Metadata Extraction**  
   - **Primary**: Extract EXIF date/time from each image using a library such as `github.com/rwcarlsen/goexif/exif`.  
   - **Fallback**: Use file modification time if EXIF data is unavailable.

2. **Data Structures**  
   - A slice of `Photo` objects, each containing:
     ```go
     type Photo struct {
       FilePath  string
       TakenTime time.Time
       Latitude  float64   // optional
       Longitude float64   // optional
       Country   string    // optional, resolved via reverse geocoding if implemented
       // Possibly more fields for overlays, caching, etc.
     }
     ```

3. **Merging Albums**  
   - Combine all `Photo` objects from multiple album directories into a single slice.  
   - Sort by `TakenTime` in ascending order.

4. **Skipping by Year**  
   - Maintain an index pointing to the currently displayed photo.  
   - When a “skip forward” event occurs, find the next `Photo` whose year is one greater than the current year.  
   - Similarly, skipping backward finds the previous `Photo` with a year one less than the current year.

### 5.3 Concurrency Model

- **Goroutines**  
  - **Slideshow Goroutine**: Handles the timed rotation of photos. Sleeps for the configured interval, then advances the photo index.  
  - **CEC Listener Goroutine**: Listens for remote input commands (forward/back). Puts commands into a channel that the Slideshow goroutine consumes to update the photo index.  
  - **Scheduler Goroutine**: Monitors the current time and toggles the TV on/off accordingly (via CEC commands).  

- **Channels**  
  - `remoteEvents := make(chan RemoteCommand)`  
  - `quit := make(chan bool)` (for graceful shutdown)  

- **Synchronization**  
  - Use channels to pass state changes and avoid shared memory pitfalls.  
  - The slideshow goroutine updates the “current photo index” in a thread-safe manner.  

### 5.4 Slideshow Rendering

- **Rendering Approach**:  
  - A minimal full-screen display using a window manager (or running a bare X server on the Pi).  
  - Potentially use a lightweight library like SDL2 or direct framebuffer for performance.  
  - For overlays, use `golang.org/x/image/draw` or `golang.org/x/image/font` to composite text onto the image before rendering.

- **Overlay Logic**:  
  - **Date Overlay**: Render date-time string in the bottom corner.  
  - **Location Overlay** (optional): Render country or city if available.  
  - Use a small bounding box at the bottom corner with a semi-transparent background to ensure readability.

### 5.5 HDMI-CEC Integration

1. **CEC Library**:  
   - Use `libcec` or a Go wrapper library (e.g., `github.com/joshkunz/cec`) to interact with the TV.

2. **Commands**:  
   - **Power On**: CEC “Image/View On”.  
   - **Power Off**: CEC “Standby”.  
   - **Select HDMI Input**: Send a command to switch to input # specified in `config.hdmiInput`.  
   - **Remote Key Events**: Parse “Forward”, “Backward”, “Right Arrow”, “Left Arrow” CEC events.  
     - On detection, push a `RemoteCommand` into `remoteEvents` channel for slideshow skipping.

3. **Scheduler**:  
   - Runs a time-based loop (cron-like or custom) that triggers:
     - At `onTime`: Turn on TV, switch input.  
     - At `offTime`: Turn off TV.

### 5.6 Error Handling & Logging

- **Logging**:  
  - Use a structured logging library (e.g., `logrus` or standard `log`) to emit logs:
    - `INFO`: Startup, loading config, scanning directories, etc.  
    - `WARN`: Missing EXIF data, skipping a corrupted file.  
    - `ERROR`: Unable to send CEC command, unable to load image file.  
  - Log to a file at `~/.openframe/openframe.log` (configurable path).

- **Resilience**:  
  - If an album directory is missing, log a warning and skip.  
  - If CEC commands fail, retry a small number of times or fallback to no-op.

---

## 6. Alternatives Considered

1. **Language Choice**:  
   - **Python**: More common for Pi projects, but concurrency can be more complex. Golang is chosen for performance, concurrency model, and strong ecosystem.  
   - **C++**: Could be used with `libcec` more directly, but less ergonomic for rapid feature development.

2. **Rendering Backends**:  
   - **Direct Framebuffer**: Faster but requires more low-level code.  
   - **SDL2**: Cross-platform library, known for gaming/graphics, but adds an external dependency.  
   - **Lightweight X11**: Possibly the simplest approach on Pi OS with minimal overhead.

3. **Schedule Implementation**:  
   - **Cron job**: Could rely on Linux `cron` to start/stop the binary, but that complicates the continuity of the application.  
   - **In-process Scheduler**: Simpler to keep all scheduling logic inside the single binary.

---

## 7. Implementation Plan

1. **Phase 1: Config & Photo Indexing**  
   - Parse JSON, load photos, sort them by date.  
   - Basic console-based slideshow (no actual rendering yet).

2. **Phase 2: Slideshow & Rendering**  
   - Implement full-screen image display.  
   - Add overlays (date, optional location).

3. **Phase 3: HDMI-CEC Integration**  
   - Power on/off TV, switch input at scheduled times.  
   - Remote command handling for skip-by-year.

4. **Phase 4: Polishing & Optimization**  
   - Add logging, error handling, robust concurrency.  
   - Test with large libraries.  
   - Possibly refine overlays (fonts, styles).

---

## 8. Testing Strategy

1. **Unit Tests**:  
   - Config parsing: ensure missing or invalid fields are handled properly.  
   - Photo indexing: test EXIF date extraction, fallback logic, sorting.  
   - Overlays: ensure date/time text is rendered in correct location.

2. **Integration Tests**:  
   - Run the slideshow with a mock CEC library to test on/off scheduling and remote events.  
   - Validate concurrency (slideshow loop + remote commands) under load.

3. **Manual QA**:  
   - Deploy on a Pi with a real Samsung The Frame TV.  
   - Confirm correct daily power cycles, skipping, and image displays.

---

## 9. Monitoring & Observability

- **Logging**: Key mechanism for diagnosing issues, especially around CEC integration.  
- **Metrics** (optional future work):  
  - Count of displayed photos per day, skip operations, or error rates.  
  - Potentially exposed via a small HTTP server if advanced monitoring is needed.

---

## 10. Security & Privacy Considerations

- **Local Only**: All photos stored locally; no external uploads.  
- **EXIF Data**: Potentially contains GPS tags; user can disable or remove them if concerned.  
- **TV Integration**: Ensure CEC access is restricted to the local Pi environment.  
- **User Config**: Keep `config.json` readable only by the user.

---

## 11. Open Questions / Future Extensions

1. **Geolocation**:  
   - Automatic reverse geocoding for location overlays or user-provided data?  
2. **Cloud Sync**:  
   - Could we eventually add a syncing feature for Google Photos or other providers?  
3. **Extended Remote Controls**:  
   - Map more remote buttons for next/previous photo, jump to oldest/newest, or other advanced navigations.  
4. **Transitions**:  
   - Fade-in/out or Ken Burns effects to improve aesthetics?

---

## 12. Conclusion

This design proposes a robust, modular Golang application to turn a Raspberry Pi 4 and a Samsung The Frame TV into a powerful, configurable digital photo frame. By focusing on clean concurrency, straightforward configuration, and HDMI-CEC integration, OpenFrame can provide a seamless user experience while remaining flexible and open-source for future enhancements.
