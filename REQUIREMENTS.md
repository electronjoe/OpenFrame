**OpenFrame Requirements Document**  
*Version 1.0*

---

## 1. Overview

OpenFrame is an open-source photo frame solution that combines hardware (Raspberry Pi 4) and software (written in Go if feasible) to display local photo albums on a Samsung The Frame TV. The solution cycles through multiple albums, merging them into correct chronological order, and provides a set of configuration options via a JSON file. HDMI-CEC is used to control The Frame TV’s power, input selection, and navigation commands from the TV remote.

---

## 2. Goals and Objectives

1. **Seamless Photo Display**  
   - Rotate through multiple photo albums in a merged chronological order.  
   - Provide an immersive photo viewing experience on a high-quality display (Samsung The Frame TV).

2. **Highly Configurable**  
   - Offer a JSON configuration file for runtime customization (e.g., on/off times, overlays, album directories).  
   - Provide optional features like photo date or location overlays.

3. **Open Source & Extensible**  
   - Use open-source components (Raspberry Pi 4, Golang) and maintain a permissive license for community contributions.  
   - Promote flexibility for future expansions and user customization.

4. **Low Maintenance & Minimal Interaction**  
   - Automatically power on/off and switch inputs based on user-defined schedules.  
   - Allow simple user navigation (skip forward/backward by year) via the TV remote through HDMI-CEC.

---

## 3. Key Stakeholders

- **End Users**: Individuals or households wanting a dynamic digital photo frame solution on a large TV.  
- **Open Source Community**: Contributors who can extend or modify the software.  
- **Hardware Enthusiasts**: Those interested in DIY setups and configuring Raspberry Pi-based devices.  
- **Third-Party Integrators**: Potentially interested in bundling or selling pre-configured kits.

---

## 4. High-Level Requirements

### 4.1 Hardware Requirements

1. **Raspberry Pi 4**  
   - Minimum 2GB RAM model (recommended 4GB to handle large image sets smoothly).  
   - Running a Linux distribution (e.g., Raspberry Pi OS).

2. **TV Display**  
   - Designed for Samsung The Frame TV (HDMI-CEC capable).  
   - Must support CEC commands to control input source, power on/off, and remote signals.

3. **Connectivity**  
   - HDMI port to connect Raspberry Pi 4 output to the TV.  
   - Stable Wi-Fi or Ethernet connectivity for potential future expansions (e.g., remote updates, remote photo management).  
   - HDMI-CEC support enabled on both Raspberry Pi (via config) and Samsung The Frame TV.

4. **Storage**  
   - Local storage (microSD card, USB drive, or external hard drive) for photo albums.  
   - Adequate space for the largest expected album(s).

### 4.2 Software Requirements

1. **Operating System**  
   - Raspberry Pi OS (Debian-based) or similar Linux distribution.  
   - Pre-installed with necessary libraries for Go development, HDMI-CEC control, and media display.

2. **Programming Language**  
   - Preferred: **Golang** (Go 1.19 or higher, if feasible).  
   - Must handle concurrency for photo loading, scheduling, and CEC events efficiently.

3. **Photo Display Logic**  
   - Support two or more local photo albums.  
   - Merge photos chronologically (oldest to newest).  
     - Must read EXIF data for date/time or rely on file creation timestamps if EXIF is not available.  
   - Cycle through photos in ascending chronological order.  
   - Configurable time interval for each photo display (e.g., 5 seconds to 1 minute).

4. **Configuration Management**  
   - JSON config file (e.g., `~/.openframe/config.json`).  
   - Command-line usage: `openframe --config ~/.openframe/config.json`.  
   - **Configuration parameters**:
     1. **`albums`**: List of local directory paths to scan for photos.  
     2. **`dateOverlay`** (boolean): Whether to overlay the photograph’s date in the bottom corner.  
     3. **`locationOverlay`** (boolean, optional): Whether to overlay the photo’s country location in the bottom corner (nice-to-have, not strictly required).  
     4. **`schedule`**:  
        - **`onTime`**: Time of day to turn on the frame (e.g., `06:00`).  
        - **`offTime`**: Time of day to turn off the frame (e.g., `21:00`).  
     5. **`interval`**: Duration in seconds or milliseconds for how long each photo is displayed.  
     6. **`hdmiInput`**: The HDMI port number for the Raspberry Pi on The Frame TV.

5. **Photo Overlays**  
   - Render text overlays (date, location) in a corner of the screen without obstructing the main image.  
   - Font style and size configurable via the JSON or environment variables.

6. **HDMI-CEC Integration**  
   - **Automatic TV Power On/Off**:  
     - Turn TV on at the specified `onTime`.  
     - Turn TV off at the specified `offTime`.  
   - **Select HDMI Input**:  
     - Automatically switch Samsung The Frame TV to the Raspberry Pi’s HDMI input at `onTime`.  
   - **Remote Navigation**:  
     - Listen for directional pad events from The Frame remote.  
     - When user presses “forward” or “backward” arrow, skip photos by one year forward or backward.  
     - Provide a brief on-screen notification that the skip occurred.

7. **Error Handling & Monitoring**  
   - Log errors to a file or console output (e.g., `~/.openframe/openframe.log`).  
   - Graceful handling of missing directories or invalid config parameters.  
   - Fallback behavior if no valid photos are found or if input switching fails.

8. **Security & Privacy**  
   - Limit network exposure: only local software required for daily operations.  
   - Minimal data collection; no cloud connectivity required.  
   - Users can choose whether to store GPS-based location data or remove it from the photos.

---

## 5. User Flows

1. **Initial Setup**  
   1. User places photos in directories on local storage.  
   2. User edits `config.json` with paths to those directories, desired overlays, on/off times, etc.  
   3. User starts the Golang binary: `openframe --config ~/.openframe/config.json`.  
   4. The software scans and indexes photos, then begins the slideshow according to the schedule.

2. **Daily Operation**  
   1. At `onTime`, Raspberry Pi sends HDMI-CEC command to turn on The Frame TV and switch input.  
   2. Photos cycle in chronological order. Overlays appear if configured.  
   3. At `offTime`, Raspberry Pi sends HDMI-CEC command to turn off the TV.

3. **Remote Interaction**  
   1. User picks up The Frame remote, presses forward (→) to skip forward one year of photos.  
   2. The Golang binary receives the HDMI-CEC event, advances the photo index to the next year’s first photo.  
   3. Similarly, pressing backward (←) navigates to the previous year’s first photo.  
   4. On-screen confirmation is displayed (e.g., “Skipping to 2019”).

---

## 6. Technical Dependencies

1. **Golang**  
   - Official Go compiler and runtime environment.  
   - Libraries for image processing (e.g., `image`, `jpeg`, `png` libraries).

2. **Linux HDMI-CEC Libraries**  
   - `cec-utils` or `libcec` to interact with CEC commands.  
   - Golang wrappers (e.g., `github.com/hdmi-cec-library/…`) or custom integration.

3. **EXIF Parsing**  
   - Third-party Go libraries for reading EXIF metadata (e.g., `github.com/rwcarlsen/goexif/exif`).

4. **JSON Parsing**  
   - Standard Go `encoding/json` for reading `config.json`.

5. **Date/Time Libraries**  
   - Standard Go libraries for scheduling tasks (e.g., using cron-like functionality or custom timers).

6. **Graphics/Overlay**  
   - Minimal 2D drawing libraries to overlay text onto images (e.g., `golang.org/x/image/draw` and `golang.org/x/image/font`).

---

## 7. Implementation Phases

1. **Phase 1: Core Photo Slideshow**  
   - Basic Golang application to load photos from directories, read EXIF timestamps, and display them in chronological order on the TV.  
   - Implement JSON config loading and photo indexing.

2. **Phase 2: HDMI-CEC Power & Input Control**  
   - Integrate `libcec` (or alternative) to power on/off and set input on The Frame.  
   - Implement daily schedule logic for turning TV on/off.

3. **Phase 3: User Interaction via Remote**  
   - Listen to CEC commands for forward/back navigation.  
   - Implement skip-by-year logic in the slideshow index.

4. **Phase 4: Overlays & Advanced Features**  
   - Add date and location overlays on the displayed image.  
   - Provide optional location-based overlays if EXIF GPS data is available.  
   - Refine UI (fonts, positioning, fade in/out transitions, etc.).

5. **Phase 5: Optimization & Polish**  
   - Ensure performance is optimized for large albums.  
   - Improve logging, error handling, and user documentation.  
   - Beta testing with community feedback.

---

## 8. Assumptions and Constraints

1. **Assumption**: The Samsung The Frame TV fully supports HDMI-CEC for power, input switching, and directional pad signals.  
2. **Assumption**: Photo directories fit on local storage (microSD, USB, or attached storage).  
3. **Assumption**: Users have some technical proficiency to configure JSON files and set up a Raspberry Pi environment.  
4. **Constraint**: The Pi must have uninterrupted power for continuous operation.  
5. **Constraint**: Continuous display operation can be power-intensive and must be considered in an energy-conscious environment.

---

## 9. Open Questions

1. **Skip Granularity**:  
   - Currently defined as skipping one year with forward/back arrows. Could we allow monthly or decade-based skipping?  
2. **Additional Remote Controls**:  
   - Could more remote buttons be mapped to other features (e.g., show next/previous photo, jump to oldest/newest)?  
3. **Cloud Integration**:  
   - Is there an interest in a future iteration that fetches photos from cloud providers (Google Photos, etc.)?  
4. **Security Features**:  
   - Do we need any password protection or encryption for specific albums?  

---

## 10. Success Criteria

- **Functional**: System reliably displays photos from multiple albums in chronological order, with optional overlays.  
- **Usability**: Configuration via JSON is straightforward, and remote interactions for skipping years work seamlessly.  
- **Stability**: The device powers on/off on schedule, and handles large photo libraries without crashing.  
- **Community Adoption**: The open-source project sees contributions and enhancements from external developers.
