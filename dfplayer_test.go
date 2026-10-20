package dfplayer

import (
	"bytes"
	"errors"
	"io"
	"testing"
)

// --- Test helpers ---

// fakeReadWriter implements io.ReadWriter. Writes are captured in writeBuf,
// reads come from readBuf. If readBuf is nil, Read returns (0, io.EOF).
type fakeReadWriter struct {
	writeBuf bytes.Buffer
	readBuf  *bytes.Reader
	writeErr error
}

func (f *fakeReadWriter) Read(p []byte) (int, error) {
	if f.readBuf == nil {
		return 0, io.EOF
	}
	return f.readBuf.Read(p)
}

func (f *fakeReadWriter) Write(p []byte) (int, error) {
	if f.writeErr != nil {
		return 0, f.writeErr
	}
	return f.writeBuf.Write(p)
}

func newFakeReadWriter(data []byte) *fakeReadWriter {
	return &fakeReadWriter{readBuf: bytes.NewReader(data)}
}

// writtenPacket returns the last packet written to the fake and resets the buffer.
func (f *fakeReadWriter) writtenPacket() []byte {
	b := f.writeBuf.Bytes()
	f.writeBuf.Reset()
	return b
}

// chunkReader returns chunks of data in order, then (0, nil) once,
// then (0, io.EOF). Used to simulate UART reads arriving in pieces.
type chunkReader struct {
	chunks    [][]byte
	pos       int
	sentEmpty bool
}

func (r *chunkReader) Read(buf []byte) (int, error) {
	if r.pos >= len(r.chunks) {
		if r.sentEmpty {
			return 0, io.EOF
		}
		r.sentEmpty = true
		return 0, nil
	}
	n := copy(buf, r.chunks[r.pos])
	r.pos++
	return n, nil
}

// parseDPPacket extracts the command code and parameter from a raw 10-byte DFPlayer packet.
func parseDPPacket(data []byte) (DPCode, uint16) {
	if len(data) < 7 {
		return 0, 0
	}
	cmd := DPCode(data[3])
	param := uint16(data[5])<<8 | uint16(data[6])
	return cmd, param
}

// bufferedReadWriter implements io.ReadWriter and bufferedReader.
type bufferedReadWriter struct {
	buf      bytes.Buffer
	writeBuf bytes.Buffer
}

func (b *bufferedReadWriter) Read(p []byte) (int, error)  { return b.buf.Read(p) }
func (b *bufferedReadWriter) Write(p []byte) (int, error) { return b.writeBuf.Write(p) }
func (b *bufferedReadWriter) Buffered() int               { return b.buf.Len() }

// ============================================================================
// Checksum tests
// ============================================================================

func TestCalcChecksum(t *testing.T) {
	tests := []struct {
		name     string
		packet   []byte
		expected uint16
	}{
		{
			name: "play track 1",
			// 0x7E 0xFF 0x06 0x03 0x00 0x00 0x01 CHK_H CHK_L 0xEF
			// sum = 0xFF+0x06+0x03+0x00+0x00+0x01 = 0x0109, checksum = 0x0000-0x0109 = 0xFEF7
			packet:   []byte{0x7E, 0xFF, 0x06, 0x03, 0x00, 0x00, 0x01, 0x00, 0x00, 0xEF},
			expected: 0xFEF7,
		},
		{
			name: "set volume 30",
			// sum = 0xFF+0x06+0x06+0x00+0x00+0x1E = 0x0129, checksum = 0xFED7
			packet:   []byte{0x7E, 0xFF, 0x06, 0x06, 0x00, 0x00, 0x1E, 0x00, 0x00, 0xEF},
			expected: 0xFED7,
		},
		{
			name: "command with all zeros in payload",
			// sum = 0xFF+0x06+0x00+0x00+0x00+0x00 = 0x0105, checksum = 0xFEFB
			packet:   []byte{0x7E, 0xFF, 0x06, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0xEF},
			expected: 0xFEFB,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := calcChecksum(tt.packet)
			if got != tt.expected {
				t.Errorf("calcChecksum() = 0x%04X, want 0x%04X", got, tt.expected)
			}
		})
	}
}

func TestCalcChecksumTwoComplement(t *testing.T) {
	// Verify that sum(bytes 1..6) + checksum ≡ 0 (mod 2^16)
	pkt := []byte{0x7E, 0xFF, 0x06, 0x03, 0x00, 0x00, 0x01, 0x00, 0x00, 0xEF}
	cs := calcChecksum(pkt)

	var sum uint16
	for i := 1; i <= 6; i++ {
		sum += uint16(pkt[i])
	}

	if sum+cs != 0 {
		t.Errorf("checksum invariant violated: sum(0x%04X) + checksum(0x%04X) = 0x%04X, want 0", sum, cs, sum+cs)
	}
}

func TestIsChecksumValid(t *testing.T) {
	valid := []byte{0x7E, 0xFF, 0x06, 0x03, 0x00, 0x00, 0x01, 0xFE, 0xF7, 0xEF}

	corrupted := make([]byte, 10)
	copy(corrupted, valid)
	corrupted[7] ^= 0xFF

	tests := []struct {
		name   string
		packet []byte
		want   bool
	}{
		{"valid packet", valid, true},
		{"corrupted checksum", corrupted, false},
		{"too short", []byte{0x7E, 0xFF, 0x06}, false},
		{"empty", []byte{}, false},
		{"exactly 9 bytes", valid[:9], false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isChecksumValid(tt.packet); got != tt.want {
				t.Errorf("isChecksumValid() = %v, want %v", got, tt.want)
			}
		})
	}
}

// ============================================================================
// sendCmd tests
// ============================================================================

func TestSendCmd_PacketStructure(t *testing.T) {
	f := &fakeReadWriter{}
	err := sendMessage(f, NextTrackCode, noParam)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	pkt := f.writtenPacket()
	if len(pkt) != 10 {
		t.Fatalf("packet length = %d, want 10", len(pkt))
	}
	if pkt[0] != 0x7E {
		t.Errorf("start byte = 0x%02X, want 0x7E", pkt[0])
	}
	if pkt[1] != 0xFF {
		t.Errorf("version byte = 0x%02X, want 0xFF", pkt[1])
	}
	if pkt[2] != 0x06 {
		t.Errorf("length byte = 0x%02X, want 0x06", pkt[2])
	}
	if pkt[3] != byte(NextTrackCode) {
		t.Errorf("command byte = 0x%02X, want 0x%02X", pkt[3], NextTrackCode)
	}
	if pkt[4] != 0x00 {
		t.Errorf("feedback byte = 0x%02X, want 0x00", pkt[4])
	}
	if pkt[9] != 0xEF {
		t.Errorf("end byte = 0x%02X, want 0xEF", pkt[9])
	}

	// Verify checksum
	if !isChecksumValid(pkt) {
		t.Error("packet has invalid checksum")
	}
}

func TestSendCmd_WithParameter(t *testing.T) {
	f := &fakeReadWriter{}
	err := sendMessage(f, SetVolumeCode, 15)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	cmd, param := parseDPPacket(f.writtenPacket())
	if cmd != SetVolumeCode {
		t.Errorf("command = 0x%02X, want 0x%02X", cmd, SetVolumeCode)
	}
	if param != 15 {
		t.Errorf("param = %d, want 15", param)
	}
}

func TestSendCmd_WriteError(t *testing.T) {
	f := &fakeReadWriter{writeErr: errors.New("broken")}
	err := sendMessage(f, NextTrackCode, noParam)

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var ue *ErrUartWrite
	if !errors.As(err, &ue) {
		t.Errorf("expected *ErrUartWrite, got %T", err)
	}
}

// ============================================================================
// readUart tests
// ============================================================================

func TestReadUart_CompletePacket(t *testing.T) {
	// A single read of exactly 10 valid bytes.
	data := []byte{0x7E, 0xFF, 0x06, 0x3F, 0x00, 0x00, 0x01, 0xFE, 0xBB, 0xEF}
	r := &chunkReader{chunks: [][]byte{data}}

	got, err := readUart(r)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !bytes.Equal(got, data) {
		t.Errorf("got % X, want % X", got, data)
	}
}

func TestReadUart_SyncOnStartByte(t *testing.T) {
	// Garbage bytes before the start byte should be skipped.
	garbage := []byte{0xAA, 0xBB, 0xCC}
	packet := []byte{0x7E, 0xFF, 0x06, 0x3F, 0x00, 0x00, 0x01, 0xFE, 0xBB, 0xEF}
	chunk := append(garbage, packet...)
	// Pad to 10 bytes — 3 garbage + 7 from packet in first read.
	r := &chunkReader{chunks: [][]byte{chunk[:10], chunk[10:]}}

	got, err := readUart(r)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !bytes.Equal(got, packet) {
		t.Errorf("got % X, want % X", got, packet)
	}
}

func TestReadUart_MultipleChunks(t *testing.T) {
	// Packet split across multiple reads.
	r := &chunkReader{chunks: [][]byte{
		{0x7E, 0xFF, 0x06},
		{0x3F, 0x00, 0x00, 0x01},
		{0xFE, 0xBB, 0xEF},
	}}

	got, err := readUart(r)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 10 {
		t.Fatalf("got %d bytes, want 10", len(got))
	}
	if got[0] != 0x7E || got[9] != 0xEF {
		t.Errorf("invalid packet framing: % X", got)
	}
}

func TestReadUart_ReadError(t *testing.T) {
	r := &chunkReader{chunks: [][]byte{
		{0x7E, 0xFF, 0x06}, // 3 bytes, then (0,nil), then EOF
	}}

	_, err := readUart(r)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var ue *ErrUartRead
	if !errors.As(err, &ue) {
		t.Errorf("expected *ErrUartRead, got %T", err)
	}
}

// ============================================================================
// New tests
// ============================================================================

func TestNew(t *testing.T) {
	f := &fakeReadWriter{}
	player, err := New(f)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if player == nil {
		t.Fatal("expected non-nil Dfplayer")
	}
	if player.uart != f {
		t.Error("Dfplayer.uart not set correctly")
	}
}

// ============================================================================
// SendCmd method tests
// ============================================================================

func TestDfplayerSendMessage(t *testing.T) {
	f := &fakeReadWriter{}
	d := &Dfplayer{uart: f}

	err := d.SendCmd(SetVolumeCode, 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	cmd, param := parseDPPacket(f.writtenPacket())
	if cmd != SetVolumeCode {
		t.Errorf("command = 0x%02X, want 0x%02X", cmd, SetVolumeCode)
	}
	if param != 10 {
		t.Errorf("param = %d, want 10", param)
	}
}

// ============================================================================
// Command method tests — verify correct DPCode and parameter
// ============================================================================

func TestCommands_NoParam(t *testing.T) {
	tests := []struct {
		name string
		call func(d *Dfplayer) error
		cmd  DPCode
	}{
		{"NextTrack", (*Dfplayer).NextTrack, NextTrackCode},
		{"PreviousTrack", (*Dfplayer).PreviousTrack, PreviousTrackCode},
		{"VolumeUp", (*Dfplayer).VolumeUp, IncreaseVolumeCode},
		{"VolumeDown", (*Dfplayer).VolumeDown, DecreaseVolumeCode},
		{"Sleep", (*Dfplayer).Sleep, SleepCode},
		{"WakeUp", (*Dfplayer).WakeUp, WakeUpCode},
		{"Reset", (*Dfplayer).Reset, ResetCode},
		{"Resume", (*Dfplayer).Resume, ResumeCode},
		{"Pause", (*Dfplayer).Pause, PauseCode},
		{"Stop", (*Dfplayer).Stop, StopCode},
		{"StopAdvertise", (*Dfplayer).StopAdvertise, StopAdvertiseCode},
		{"RandomAll", (*Dfplayer).RandomAll, RandomCode},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &fakeReadWriter{}
			d := &Dfplayer{uart: f}

			err := tt.call(d)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			cmd, param := parseDPPacket(f.writtenPacket())
			if cmd != tt.cmd {
				t.Errorf("command = 0x%02X, want 0x%02X", cmd, tt.cmd)
			}
			if param != 0 {
				t.Errorf("param = %d, want 0", param)
			}
		})
	}
}

func TestCommands_WithParameter(t *testing.T) {
	tests := []struct {
		name  string
		call  func(d *Dfplayer) error
		cmd   DPCode
		param uint16
	}{
		{"SetEQ Normal", func(d *Dfplayer) error { return d.SetEQ(EqNormal) }, SetEQCode, 0},
		{"SetEQ Bass", func(d *Dfplayer) error { return d.SetEQ(EqBass) }, SetEQCode, 5},
		{"SetRepeatMode All", func(d *Dfplayer) error { return d.SetRepeatMode(RepeatModeAll) }, SetRepeatModeCode, 0},
		{"SetRepeatMode Random", func(d *Dfplayer) error { return d.SetRepeatMode(RepeatModeRandom) }, SetRepeatModeCode, 3},
		{"SelectSource SD", func(d *Dfplayer) error { return d.SelectSource(SourceSD) }, SelectSourceCode, 2},
		{"SelectSource USB", func(d *Dfplayer) error { return d.SelectSource(SourceUSB) }, SelectSourceCode, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &fakeReadWriter{}
			d := &Dfplayer{uart: f}

			err := tt.call(d)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			cmd, param := parseDPPacket(f.writtenPacket())
			if cmd != tt.cmd {
				t.Errorf("command = 0x%02X, want 0x%02X", cmd, tt.cmd)
			}
			if param != tt.param {
				t.Errorf("param = %d, want %d", param, tt.param)
			}
		})
	}
}

// ============================================================================
// Play track tests — validation + correct parameter encoding
// ============================================================================

func TestPlayMp3Track(t *testing.T) {
	tests := []struct {
		name    string
		track   uint16
		wantErr bool
	}{
		{"valid min", 1, false},
		{"valid max", 2999, false},
		{"valid mid", 500, false},
		{"invalid zero", 0, true},
		{"invalid too large", 3000, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &fakeReadWriter{}
			d := &Dfplayer{uart: f}

			err := d.PlayMp3Track(tt.track)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			cmd, param := parseDPPacket(f.writtenPacket())
			if cmd != PlayMp3FolderCode {
				t.Errorf("command = 0x%02X, want 0x%02X", cmd, PlayMp3FolderCode)
			}
			if param != tt.track {
				t.Errorf("param = %d, want %d", param, tt.track)
			}
		})
	}
}

func TestPlayRootTrack(t *testing.T) {
	tests := []struct {
		name    string
		track   uint16
		wantErr bool
	}{
		{"valid", 42, false},
		{"invalid zero", 0, true},
		{"invalid too large", 3000, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &fakeReadWriter{}
			d := &Dfplayer{uart: f}

			err := d.PlayRootTrack(tt.track)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			cmd, param := parseDPPacket(f.writtenPacket())
			if cmd != PlayRootFolderCode || param != tt.track {
				t.Errorf("wrong command/param: cmd=%d param=%d", cmd, param)
			}
		})
	}
}

func TestSetVolume(t *testing.T) {
	tests := []struct {
		name    string
		vol     uint16
		wantErr bool
	}{
		{"valid 0", 0, false},
		{"valid 30", 30, false},
		{"valid mid", 15, false},
		{"invalid 31", 31, true},
		{"invalid large", 100, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &fakeReadWriter{}
			d := &Dfplayer{uart: f}

			err := d.SetVolume(tt.vol)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			cmd, param := parseDPPacket(f.writtenPacket())
			if cmd != SetVolumeCode || param != tt.vol {
				t.Errorf("wrong cmd=%d param=%d", cmd, param)
			}
		})
	}
}

func TestPlayFolder(t *testing.T) {
	tests := []struct {
		name    string
		folder  uint16
		track   uint16
		wantErr bool
	}{
		{"valid", 5, 10, false},
		{"valid min", 1, 1, false},
		{"valid max", 99, 255, false},
		{"invalid folder zero", 0, 10, true},
		{"invalid folder too large", 100, 10, true},
		{"invalid track zero", 5, 0, true},
		{"invalid track too large", 5, 256, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &fakeReadWriter{}
			d := &Dfplayer{uart: f}

			err := d.PlayFolder(tt.folder, tt.track)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			cmd, param := parseDPPacket(f.writtenPacket())
			expectedParam := (tt.folder << 8) | tt.track
			if cmd != PlayFolderCode || param != expectedParam {
				t.Errorf("cmd=%d, param=0x%04X, want cmd=%d param=0x%04X",
					cmd, param, PlayFolderCode, expectedParam)
			}
		})
	}
}

func TestPlayLargeFolder(t *testing.T) {
	tests := []struct {
		name    string
		folder  uint16
		track   uint16
		wantErr bool
	}{
		{"valid", 5, 100, false},
		{"valid max", 15, 3000, false},
		{"invalid folder zero", 0, 10, true},
		{"invalid folder too large", 16, 10, true},
		{"invalid track zero", 5, 0, true},
		{"invalid track too large", 5, 3001, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &fakeReadWriter{}
			d := &Dfplayer{uart: f}

			err := d.PlayLargeFolder(tt.folder, tt.track)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			cmd, param := parseDPPacket(f.writtenPacket())
			expectedParam := (tt.folder << 12) | tt.track
			if cmd != PlayLargeFolderCode || param != expectedParam {
				t.Errorf("cmd=%d, param=0x%04X, want cmd=%d param=0x%04X",
					cmd, param, PlayLargeFolderCode, expectedParam)
			}
		})
	}
}

func TestAdvertise(t *testing.T) {
	tests := []struct {
		name    string
		track   uint16
		wantErr bool
	}{
		{"valid", 1, false},
		{"invalid zero", 0, true},
		{"invalid too large", 3000, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &fakeReadWriter{}
			d := &Dfplayer{uart: f}

			err := d.Advertise(tt.track)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			cmd, param := parseDPPacket(f.writtenPacket())
			if cmd != PlayAdvertiseCode || param != tt.track {
				t.Errorf("wrong cmd=%d param=%d", cmd, param)
			}
		})
	}
}

func TestSetOutput(t *testing.T) {
	tests := []struct {
		name    string
		enable  bool
		gain    uint16
		wantErr bool
	}{
		{"enable gain 0", true, 0, false},
		{"enable gain 31", true, 31, false},
		{"disable gain 0", false, 0, false},
		{"enable gain too large", true, 32, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &fakeReadWriter{}
			d := &Dfplayer{uart: f}

			err := d.SetOutput(tt.enable, tt.gain)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			cmd, param := parseDPPacket(f.writtenPacket())
			expectedParam := tt.gain
			if tt.enable {
				expectedParam |= 0x0100
			}
			if cmd != GainCode || param != expectedParam {
				t.Errorf("cmd=%d, param=0x%04X, want cmd=%d param=0x%04X",
					cmd, param, GainCode, expectedParam)
			}
		})
	}
}

func TestRepeatFolder(t *testing.T) {
	tests := []struct {
		name    string
		folder  uint16
		wantErr bool
	}{
		{"valid", 5, false},
		{"valid max", 99, false},
		{"invalid zero", 0, true},
		{"invalid too large", 100, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &fakeReadWriter{}
			d := &Dfplayer{uart: f}

			err := d.RepeatFolder(tt.folder)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			cmd, param := parseDPPacket(f.writtenPacket())
			if cmd != RepeatFolderCode || param != tt.folder {
				t.Errorf("cmd=%d, param=%d, want cmd=%d param=%d",
					cmd, param, RepeatFolderCode, tt.folder)
			}
		})
	}
}

// ============================================================================
// GetCmd tests
// ============================================================================

func TestGetCmd_ValidResponse(t *testing.T) {
	// Simulate a "ready" response from the DFPlayer.
	// 0x7E 0xFF 0x06 0x3F 0x00 0x00 0x01 0xFE 0xBB 0xEF
	// checksum: sum(FF,06,3F,00,00,01)=0x145, 0x0000-0x0145=0xFEBB
	resp := []byte{0x7E, 0xFF, 0x06, 0x3F, 0x00, 0x00, 0x02, 0xFE, 0xBA, 0xEF}
	f := newFakeReadWriter(resp)
	d := &Dfplayer{uart: f}

	msg, err := d.ReadMessage()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if msg.Cmd != ReadyCode {
		t.Errorf("code = 0x%02X, want ReadyCode (0x%02X)", msg.Cmd, ReadyCode)
	}
}

func TestGetCmd_InvalidChecksum(t *testing.T) {
	// Packet with wrong checksum
	resp := []byte{0x7E, 0xFF, 0x06, 0x3F, 0x00, 0x00, 0x02, 0x00, 0x00, 0xEF}
	f := newFakeReadWriter(resp)
	d := &Dfplayer{uart: f}

	_, err := d.ReadMessage()
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var de *ErrDfplayer
	if !errors.As(err, &de) {
		t.Errorf("expected *ErrDfplayer, got %T", err)
	}
}

func TestGetCmd_ErrorResponse(t *testing.T) {
	// Simulate an error response (0x40 = ErrorCode, param=0x01 = Busy)
	// sum: FF+06+40+00+00+01 = 0x146, checksum = 0xFEBA
	resp := []byte{0x7E, 0xFF, 0x06, 0x40, 0x00, 0x00, 0x01, 0xFE, 0xBA, 0xEF}
	f := newFakeReadWriter(resp)
	d := &Dfplayer{uart: f}

	msg, err := d.ReadMessage()
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if msg != nil {
		t.Errorf("expected nil message on error, got %+v", msg)
	}

	if !errors.Is(err, ErrBusy) {
		t.Errorf("expected ErrBusy, got %v", err)
	}
}

func TestGetCmd_ReadError(t *testing.T) {
	// Reader returns EOF immediately.
	f := newFakeReadWriter(nil)
	d := &Dfplayer{uart: f}

	_, err := d.ReadMessage()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var ue *ErrUartRead
	if !errors.As(err, &ue) {
		t.Errorf("expected *ErrUartRead, got %T", err)
	}
}

// ============================================================================
// Enable/Disable tests
// ============================================================================

func TestEnableDisable(t *testing.T) {
	tests := []struct {
		name  string
		call  func(d *Dfplayer) error
		cmd   DPCode
		param uint16
	}{
		{"EnableDAC", (*Dfplayer).EnableDAC, DACCode, enableParam},
		{"DisableDAC", (*Dfplayer).DisableDAC, DACCode, disableParam},
		{"EnableRepeatCurrent", (*Dfplayer).EnableRepeatCurrent, RepeatCurrentCode, enableParam},
		{"DisableRepeatCurrent", (*Dfplayer).DisableRepeatCurrent, RepeatCurrentCode, disableParam},
		{"EnableLoopAll", (*Dfplayer).EnableLoopAll, LoopCode, 0x01},
		{"DisableLoopAll", (*Dfplayer).DisableLoopAll, LoopCode, 0x00},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &fakeReadWriter{}
			d := &Dfplayer{uart: f}

			err := tt.call(d)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			cmd, param := parseDPPacket(f.writtenPacket())
			if cmd != tt.cmd {
				t.Errorf("command = 0x%02X, want 0x%02X", cmd, tt.cmd)
			}
			if param != tt.param {
				t.Errorf("param = 0x%04X, want 0x%04X", param, tt.param)
			}
		})
	}
}

// ============================================================================
// Drain tests
// ============================================================================

func TestDrain_WithBufferedData(t *testing.T) {
	b := &bufferedReadWriter{}
	// Pre-fill with 25 bytes of stale data.
	b.buf.Write(bytes.Repeat([]byte{0x7E}, 25))
	d := &Dfplayer{uart: b}

	err := d.Drain()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if b.Buffered() != 0 {
		t.Errorf("Buffered() = %d after Drain, want 0", b.Buffered())
	}
}

func TestDrain_EmptyBuffer(t *testing.T) {
	b := &bufferedReadWriter{}
	d := &Dfplayer{uart: b}

	err := d.Drain()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if b.Buffered() != 0 {
		t.Errorf("Buffered() = %d, want 0", b.Buffered())
	}
}

func TestDrain_NoBufferedReader(t *testing.T) {
	// A plain io.ReadWriter without Buffered() — Drain should be a no-op.
	f := &fakeReadWriter{}
	d := &Dfplayer{uart: f}

	err := d.Drain()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// ============================================================================
// Error type tests
// ============================================================================

func TestErrUartRead_Error(t *testing.T) {
	cause := errors.New("underlying")
	e := &ErrUartRead{text: "read failed", err: cause}
	if e.Error() != "read failed" {
		t.Errorf("Error() = %q, want %q", e.Error(), "read failed")
	}
	if !errors.Is(e, cause) {
		t.Error("errors.Is should unwrap to underlying error")
	}
}

func TestErrUartRead_UnwrapNil(t *testing.T) {
	e := &ErrUartRead{text: "timeout"}
	if e.Unwrap() != nil {
		t.Error("Unwrap() should return nil when err is nil")
	}
}

func TestErrUartWrite_Error(t *testing.T) {
	cause := errors.New("broken pipe")
	e := &ErrUartWrite{text: "write failed", err: cause}
	if e.Error() != "write failed" {
		t.Errorf("Error() = %q, want %q", e.Error(), "write failed")
	}
	if !errors.Is(e, cause) {
		t.Error("errors.Is should unwrap to underlying error")
	}
}

func TestErrDfplayer_Error(t *testing.T) {
	e := &ErrDfplayer{text: "checksum is invalid"}
	if e.Error() != "checksum is invalid" {
		t.Errorf("Error() = %q, want %q", e.Error(), "checksum is invalid")
	}
}

// ============================================================================
// Source and RepeatMode constant values
// ============================================================================

func TestSourceConstants(t *testing.T) {
	if SourceUSB != 1 {
		t.Errorf("SourceUSB = %d, want 1", SourceUSB)
	}
	if SourceSD != 2 {
		t.Errorf("SourceSD = %d, want 2", SourceSD)
	}
}

func TestRepeatModeConstants(t *testing.T) {
	if RepeatModeAll != 0 {
		t.Errorf("RepeatModeAll = %d, want 0", RepeatModeAll)
	}
	if RepeatModeRandom != 3 {
		t.Errorf("RepeatModeRandom = %d, want 3", RepeatModeRandom)
	}
}

func TestEqModeConstants(t *testing.T) {
	if EqNormal != 0 {
		t.Errorf("EqNormal = %d, want 0", EqNormal)
	}
	if EqBass != 5 {
		t.Errorf("EqBass = %d, want 5", EqBass)
	}
}
