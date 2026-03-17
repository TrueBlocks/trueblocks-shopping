package color

import (
	"math"
)

// Lab represents a color in CIELAB color space
type Lab struct {
	L float64 // Lightness: 0-100
	A float64 // Green-Red axis
	B float64 // Blue-Yellow axis
}

// D65 standard illuminant reference white point
var D65 = struct {
	X, Y, Z float64
}{95.047, 100.0, 108.883}

// RGBToLab converts RGB (0-255) to CIELAB color space
// Pipeline: sRGB -> Linear RGB -> XYZ -> Lab
func RGBToLab(r, g, b uint8) Lab {
	// Step 1: Normalize RGB to 0-1 range
	rNorm := float64(r) / 255.0
	gNorm := float64(g) / 255.0
	bNorm := float64(b) / 255.0

	// Step 2: sRGB to Linear RGB (remove gamma correction ~2.2)
	rLin := sRGBToLinear(rNorm)
	gLin := sRGBToLinear(gNorm)
	bLin := sRGBToLinear(bNorm)

	// Step 3: Linear RGB to XYZ using D65 illuminant matrix
	// Standard sRGB to XYZ transformation matrix
	x := rLin*0.4124564 + gLin*0.3575761 + bLin*0.1804375
	y := rLin*0.2126729 + gLin*0.7151522 + bLin*0.0721750
	z := rLin*0.0193339 + gLin*0.1191920 + bLin*0.9503041

	// Scale to 0-100 range
	x *= 100
	y *= 100
	z *= 100

	// Step 4: XYZ to Lab
	return xyzToLab(x, y, z)
}

// sRGBToLinear converts sRGB component to linear RGB
func sRGBToLinear(c float64) float64 {
	if c <= 0.04045 {
		return c / 12.92
	}
	return math.Pow((c+0.055)/1.055, 2.4)
}

// xyzToLab converts XYZ to CIELAB
func xyzToLab(x, y, z float64) Lab {
	// Normalize by reference white point
	xn := x / D65.X
	yn := y / D65.Y
	zn := z / D65.Z

	// Apply the nonlinear transformation
	fx := labF(xn)
	fy := labF(yn)
	fz := labF(zn)

	// Calculate Lab values
	l := 116*fy - 16
	a := 500 * (fx - fy)
	b := 200 * (fy - fz)

	return Lab{L: l, A: a, B: b}
}

// labF is the nonlinear transformation function for Lab conversion
// Accounts for the eye's logarithmic response to luminance
func labF(t float64) float64 {
	const delta = 6.0 / 29.0
	const delta3 = delta * delta * delta // ~0.008856

	if t > delta3 {
		return math.Pow(t, 1.0/3.0)
	}
	return t/(3*delta*delta) + 4.0/29.0
}

// LabToRGB converts CIELAB to RGB (for display purposes)
func LabToRGB(lab Lab) (r, g, b uint8) {
	// Lab to XYZ
	fy := (lab.L + 16) / 116
	fx := lab.A/500 + fy
	fz := fy - lab.B/200

	const delta = 6.0 / 29.0
	const delta3 = delta * delta * delta

	var x, y, z float64
	if fy > delta {
		y = D65.Y * math.Pow(fy, 3)
	} else {
		y = (fy - 4.0/29.0) * 3 * delta * delta * D65.Y
	}
	if fx > delta {
		x = D65.X * math.Pow(fx, 3)
	} else {
		x = (fx - 4.0/29.0) * 3 * delta * delta * D65.X
	}
	if fz > delta {
		z = D65.Z * math.Pow(fz, 3)
	} else {
		z = (fz - 4.0/29.0) * 3 * delta * delta * D65.Z
	}

	// Scale back to 0-1 range
	x /= 100
	y /= 100
	z /= 100

	// XYZ to Linear RGB
	rLin := x*3.2404542 + y*-1.5371385 + z*-0.4985314
	gLin := x*-0.9692660 + y*1.8760108 + z*0.0415560
	bLin := x*0.0556434 + y*-0.2040259 + z*1.0572252

	// Linear RGB to sRGB
	rNorm := linearToSRGB(rLin)
	gNorm := linearToSRGB(gLin)
	bNorm := linearToSRGB(bLin)

	// Convert to 0-255 range with clamping
	r = clampToByte(rNorm * 255)
	g = clampToByte(gNorm * 255)
	b = clampToByte(bNorm * 255)

	return
}

// linearToSRGB converts linear RGB component to sRGB
func linearToSRGB(c float64) float64 {
	if c <= 0.0031308 {
		return 12.92 * c
	}
	return 1.055*math.Pow(c, 1.0/2.4) - 0.055
}

// clampToByte clamps a float to 0-255 range and converts to uint8
func clampToByte(f float64) uint8 {
	if f < 0 {
		return 0
	}
	if f > 255 {
		return 255
	}
	return uint8(math.Round(f))
}

// HexToRGB converts a hex color string to RGB values
func HexToRGB(hex string) (r, g, b uint8) {
	// Remove # prefix if present
	if len(hex) > 0 && hex[0] == '#' {
		hex = hex[1:]
	}

	// Parse 6-character hex
	if len(hex) == 6 {
		var rgb uint64
		for _, c := range hex {
			rgb <<= 4
			switch {
			case c >= '0' && c <= '9':
				rgb |= uint64(c - '0')
			case c >= 'a' && c <= 'f':
				rgb |= uint64(c - 'a' + 10)
			case c >= 'A' && c <= 'F':
				rgb |= uint64(c - 'A' + 10)
			}
		}
		r = uint8((rgb >> 16) & 0xFF)
		g = uint8((rgb >> 8) & 0xFF)
		b = uint8(rgb & 0xFF)
	}
	return
}

// RGBToHex converts RGB values to a hex string
func RGBToHex(r, g, b uint8) string {
	return "#" + byteToHex(r) + byteToHex(g) + byteToHex(b)
}

func byteToHex(b uint8) string {
	const hexChars = "0123456789ABCDEF"
	return string([]byte{hexChars[b>>4], hexChars[b&0x0F]})
}
