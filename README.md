# dfplayer

[![Go Reference](https://pkg.go.dev/badge/github.com/gasoid/dfplayer.svg)](https://pkg.go.dev/github.com/gasoid/dfplayer)

TinyGo driver for the **DFPlayer Mini** MP3 module over UART.

The DFPlayer Mini is a low-cost MP3 player module controlled via serial commands at 9600 baud. This package provides a clean, idiomatic Go API for microcontrollers running TinyGo — just pass your board's `machine.UART` and start playing.

## Features

- Full command set: play, pause, stop, volume, EQ, repeat, shuffle, and more
- Works directly with TinyGo's `machine.UART` (no extra dependencies)
- Checksum validation on received packets
- Typed constants for commands, EQ presets, repeat modes, and sources
- Sensible error types with `errors.Unwrap` support
- Zero external dependencies — standard library only

## Installation

```sh
go get github.com/gasoid/dfplayer
```

## Quick Start

```go
package main

import (
	"time"

	"machine"

	"github.com/gasoid/dfplayer"
)

func main() {
	// Configure UART2 at 9600 baud (pins vary by board)
	uart := machine.UART2
	uart.Configure(machine.UARTConfig{
		BaudRate: 9600,
		TX:       machine.UART_TX_PIN, // Replace with your board's TX pin
		RX:       machine.UART_RX_PIN, // Replace with your board's RX pin
	})

	player, _ := dfplayer.New(uart)

	// Wait for the module to finish booting
	if err := player.WaitReady(5 * time.Second); err != nil {
		println("DFPlayer not ready:", err.Error())
		return
	}

	// Set volume to 15 (max 30)
	player.SetVolume(15)

	// Play track 1 from the root folder
	player.PlayRootTrack(1)

	time.Sleep(10 * time.Second)

	player.Pause()
}
```

## API Overview

### Connection

| Method | Description |
|--------|-------------|
| `New(uart io.ReadWriter) (*Dfplayer, error)` | Create a new DFPlayer instance |
| `WaitReady(timeout time.Duration) error` | Block until the module is ready (use after power-on) |
| `Drain() error` | Non-blocking discard of pending RX bytes |
| `GetCmd() (DPCode, error)` | Read the next response code from the module |
| `SendCmd(cmd DPCode, param uint16) error` | Send a raw command with parameter |

### Playback Control

| Method | Description |
|--------|-------------|
| `PlayMp3Track(track uint16)` | Play a track from the "mp3" folder (1–2999) |
| `PlayRootTrack(track uint16)` | Play a track from the root folder (1–2999) |
| `PlayFolder(folder, track uint16)` | Play a track from a specific folder (1–99, 1–255) |
| `PlayLargeFolder(folder, track uint16)` | Play from a folder with many tracks (1–15, 1–3000) |
| `NextTrack()` | Skip to the next track |
| `PreviousTrack()` | Go back to the previous track |
| `Pause()` | Pause playback |
| `Resume()` | Resume playback |
| `Stop()` | Stop playback |

### Volume & EQ

| Method | Description |
|--------|-------------|
| `SetVolume(vol uint16)` | Set volume level (0–30) |
| `VolumeUp()` | Increase volume by 1 |
| `VolumeDown()` | Decrease volume by 1 |
| `SetEQ(eq EqMode)` | Set equalizer preset (`EqNormal`, `EqPop`, `EqRock`, `EqJazz`, `EqClassic`, `EqBass`) |

### Playback Modes

| Method | Description |
|--------|-------------|
| `EnableLoopAll()` / `DisableLoopAll()` | Enable/disable looping all tracks |
| `EnableRepeatCurrent()` / `DisableRepeatCurrent()` | Enable/disable single-track repeat |
| `SetRepeatMode(mode RepeatMode)` | Set playback mode (`RepeatModeAll`, `RepeatModeFolder`, `RepeatModeOne`, `RepeatModeRandom`) |
| `RepeatFolder(folder uint16)` | Loop all tracks in a folder (1–99) |
| `RandomAll()` | Shuffle all tracks across all folders |

### Source & Output

| Method | Description |
|--------|-------------|
| `SelectSource(source Source)` | Switch source (`SourceUSB`, `SourceSD`, `SourceAUX`, `SourceSleep`, `SourceFlash`) |
| `EnableDAC()` / `DisableDAC()` | Enable/disable DAC audio output |
| `SetOutput(enable bool, gain uint16)` | Configure output and gain level (gain 0–31) |

### Power & Reset

| Method | Description |
|--------|-------------|
| `Sleep()` | Enter sleep mode |
| `WakeUp()` | Wake from sleep mode |
| `Reset()` | Software reset |

### Advertisements

| Method | Description |
|--------|-------------|
| `Advertise(track uint16)` | Play an ad track from the "advert" folder |
| `StopAdvertise()` | Stop the current advertisement |

### Status

| Method | Description |
|--------|-------------|
| `IsTrackFinished() (bool, error)` | Check if the current track has finished |
| `IsReady() (bool, error)` | Check if the module is ready to accept commands |
| `WaitTrack(timeout time.Duration) error` | Block until the current track finishes or timeout occurs |

## Error Handling

The package defines three error types:

- **`ErrUartRead`** — UART read failures (timeout, short packet, I/O error)
- **`ErrUartWrite`** — UART write failures
- **`ErrDfplayer`** — DFPlayer protocol errors (invalid parameter, checksum mismatch, module error response)

All error types implement the `error` interface. `ErrUartRead` and `ErrUartWrite` also implement `Unwrap()` for use with `errors.Is` and `errors.As`.

## License

MIT