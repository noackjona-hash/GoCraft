package mcaudio

import (
	"math"
	"math/rand"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// MCAudioEngine synthesizes Minecraft block break/place, footsteps, TNT, and peaceful piano chords
type MCAudioEngine struct {
	AudioAvailable bool
	Stream         rl.AudioStream
	SampleRate     uint32

	// SFX Timers
	BreakTimer float32
	BreakPitch float64
	PlaceTimer float32
	StepTimer  float32
	TNTTimer   float32
	SplashTimer float32

	// Underwater Audio
	IsUnderwater    bool
	BubbleTimer     float32
	UnderwaterHumPhase float64

	// Ambient Music Generator (C418 Piano chords)
	MusicTimer float32
	ChordIndex int
	NotePhase  [3]float64
}

// NewMCAudioEngine initializes raylib audio device and procedural Minecraft sound synthesizer
func NewMCAudioEngine() *MCAudioEngine {
	ae := &MCAudioEngine{
		SampleRate: 48000,
	}

	if !rl.IsAudioDeviceReady() {
		rl.InitAudioDevice()
	}

	if rl.IsAudioDeviceReady() {
		ae.AudioAvailable = true
		ae.Stream = rl.LoadAudioStream(ae.SampleRate, 16, 1)
		rl.PlayAudioStream(ae.Stream)
	}

	return ae
}

// Close cleans up audio stream
func (ae *MCAudioEngine) Close() {
	if ae.AudioAvailable {
		rl.UnloadAudioStream(ae.Stream)
		rl.CloseAudioDevice()
		ae.AudioAvailable = false
	}
}

// TriggerBlockBreak triggers the crisp block breaking crunch
func (ae *MCAudioEngine) TriggerBlockBreak() {
	ae.BreakTimer = 0.16
	ae.BreakPitch = 0.8 + rand.Float64()*0.4
}

// TriggerBlockPlace triggers block placing thud
func (ae *MCAudioEngine) TriggerBlockPlace() {
	ae.PlaceTimer = 0.12
}

// TriggerFootstep triggers walking footstep sound
func (ae *MCAudioEngine) TriggerFootstep() {
	ae.StepTimer = 0.08
}

// TriggerTNTExplosion triggers TNT blast
func (ae *MCAudioEngine) TriggerTNTExplosion() {
	ae.TNTTimer = 0.75
}

// TriggerWaterSplash plays realistic water splashing sound
func (ae *MCAudioEngine) TriggerWaterSplash() {
	ae.SplashTimer = 0.28
}

// TriggerWaterPaddle plays soft water swimming strokes
func (ae *MCAudioEngine) TriggerWaterPaddle() {
	ae.SplashTimer = 0.14
}

// Update advances timers and ambient music
func (ae *MCAudioEngine) Update(dt float32) {
	if ae.BreakTimer > 0 {
		ae.BreakTimer -= dt
	}
	if ae.PlaceTimer > 0 {
		ae.PlaceTimer -= dt
	}
	if ae.StepTimer > 0 {
		ae.StepTimer -= dt
	}
	if ae.TNTTimer > 0 {
		ae.TNTTimer -= dt
	}
	if ae.SplashTimer > 0 {
		ae.SplashTimer -= dt
	}

	if ae.IsUnderwater {
		ae.BubbleTimer += dt
		if ae.BubbleTimer > 2.5 {
			ae.BubbleTimer = 0
		}
	}

	// Ambient piano music interval (~18 seconds)
	ae.MusicTimer += dt
	if ae.MusicTimer > 18.0 {
		ae.MusicTimer = 0
		ae.ChordIndex = (ae.ChordIndex + 1) % 4
	}
}

// UpdateAudioStream synthesizes and fills the audio buffer
func (ae *MCAudioEngine) UpdateAudioStream() {
	if !ae.AudioAvailable || !rl.IsAudioStreamProcessed(ae.Stream) {
		return
	}

	bufferSize := 1024
	samples := make([]int16, bufferSize)

	dtSample := 1.0 / float64(ae.SampleRate)
	twoPi := 2.0 * math.Pi

	// C418-style Pentatonic Chords (F, C, G, Am)
	chordFreqs := [4][3]float64{
		{261.63, 329.63, 392.00}, // C Major
		{220.00, 261.63, 329.63}, // A Minor
		{174.61, 220.00, 261.63}, // F Major
		{196.00, 246.94, 293.66}, // G Major
	}

	for i := 0; i < bufferSize; i++ {
		totalOut := 0.0

		// 0. Underwater Ambient Rumble & Muffled Tone
		if ae.IsUnderwater {
			ae.UnderwaterHumPhase += 55.0 * dtSample * twoPi
			if ae.UnderwaterHumPhase > twoPi {
				ae.UnderwaterHumPhase -= twoPi
			}
			hum := math.Sin(ae.UnderwaterHumPhase) * 0.12
			noise := (rand.Float64()*2.0 - 1.0) * 0.04
			totalOut += hum + noise

			if ae.BubbleTimer < 0.15 {
				bPhase := float64(i) * 380.0 * dtSample * twoPi
				totalOut += math.Sin(bPhase) * 0.08
			}
		}

		// 1. Block Break Sound (Crunchy noise transient)
		if ae.BreakTimer > 0 {
			progress := float64(ae.BreakTimer / 0.16)
			noise := (rand.Float64()*2.0 - 1.0) * progress * 0.45
			if ae.IsUnderwater {
				noise *= 0.4 // Muffled underwater
			}
			totalOut += noise
		}

		// 2. Block Place Sound (Soft thud)
		if ae.PlaceTimer > 0 {
			progress := float64(ae.PlaceTimer / 0.12)
			fThud := 120.0 * progress
			thud := math.Sin(float64(i)*fThud*dtSample*twoPi) * progress * 0.38
			if ae.IsUnderwater {
				thud *= 0.5
			}
			totalOut += thud
		}

		// 3. Footstep
		if ae.StepTimer > 0 {
			noise := (rand.Float64()*2.0 - 1.0) * float64(ae.StepTimer/0.08) * 0.18
			totalOut += noise
		}

		// 4. TNT Explosion
		if ae.TNTTimer > 0 {
			progress := float64(ae.TNTTimer / 0.75)
			subRumble := math.Sin(float64(i)*0.015) * progress * 0.5
			noise := (rand.Float64()*2.0 - 1.0) * progress * 0.6
			totalOut += subRumble + noise
		}

		// 5. Water Splash & Swimming
		if ae.SplashTimer > 0 {
			progress := float64(ae.SplashTimer / 0.28)
			splashNoise := (rand.Float64()*2.0 - 1.0) * progress * 0.38
			splashTone := math.Sin(float64(i)*260.0*progress*dtSample*twoPi) * progress * 0.22
			totalOut += splashNoise + splashTone
		}

		// 6. Peaceful Ambient Piano Chords
		if ae.MusicTimer < 6.0 {
			musicEnv := math.Sin((float64(ae.MusicTimer) / 6.0) * math.Pi) * 0.12
			curChord := chordFreqs[ae.ChordIndex]

			for note := 0; note < 3; note++ {
				noteWave := math.Sin(ae.NotePhase[note]) * musicEnv
				totalOut += noteWave
				ae.NotePhase[note] += twoPi * curChord[note] * dtSample
				if ae.NotePhase[note] > twoPi {
					ae.NotePhase[note] -= twoPi
				}
			}
		}

		if totalOut > 0.95 {
			totalOut = 0.95
		} else if totalOut < -0.95 {
			totalOut = -0.95
		}

		samples[i] = int16(totalOut * 32767.0)
	}

	rl.UpdateAudioStream(ae.Stream, samples)
}
