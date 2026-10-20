# dfplayer

[![Go Reference](https://pkg.go.dev/badge/github.com/gasoid/dfplayer.svg)](https://pkg.go.dev/github.com/gasoid/dfplayer)

TinyGo driver for the **DFPlayer Mini** MP3 module over UART.

The DFPlayer Mini is a low-cost MP3 player module controlled via serial commands at 9600 baud. This package provides a clean, idiomatic Go API for microcontrollers running TinyGo — just pass your board's `machine.UART` and start playing.

<img width="70%" alt="IMG_6117" src="https://github.com/user-attachments/assets/6304741c-243b-4478-8e8a-e78f15be582e" />

## Features

- Full command set: play, pause, stop, volume, EQ, repeat, shuffle, and more
- Works directly via ReadWrite interface, no need for a separate library
- Checksum validation on received packets
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
	player.WaitTrack(2 * time.Second) // Wait for 2 seconds

	// Play track 0001.mp3 from mp3 folder
	player.PlayMp3Track(1)
	player.WaitTrack(3 * time.Second) // Wait for 3 seconds

	// Play track 0001.mp3 from large 01 folder
	player.PlayLargeFolder(1, 1)
	player.WaitTrack(3 * time.Second) // Wait for 3 seconds

	player.Pause()
}
```


## License

MIT