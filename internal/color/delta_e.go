package color

import (
	"math"
)

// DeltaE2000 calculates the CIEDE2000 color difference between two Lab colors.
// This is the gold standard for perceptual color difference, accounting for
// the non-uniformity of Lab space, particularly in the blue region.
func DeltaE2000(lab1, lab2 Lab) float64 {
	// Extract Lab values
	L1, a1, b1 := lab1.L, lab1.A, lab1.B
	L2, a2, b2 := lab2.L, lab2.A, lab2.B

	// Calculate C* (chroma) for both colors
	C1 := math.Sqrt(a1*a1 + b1*b1)
	C2 := math.Sqrt(a2*a2 + b2*b2)
	Cbar := (C1 + C2) / 2

	// Calculate G (adjustment factor for a*)
	Cbar7 := math.Pow(Cbar, 7)
	G := 0.5 * (1 - math.Sqrt(Cbar7/(Cbar7+6103515625))) // 25^7 = 6103515625

	// Adjusted a* values
	a1Prime := a1 * (1 + G)
	a2Prime := a2 * (1 + G)

	// Calculate C'
	C1Prime := math.Sqrt(a1Prime*a1Prime + b1*b1)
	C2Prime := math.Sqrt(a2Prime*a2Prime + b2*b2)

	// Calculate h' (hue angle)
	h1Prime := hueAngle(a1Prime, b1)
	h2Prime := hueAngle(a2Prime, b2)

	// Calculate ΔL', ΔC', ΔH'
	deltaLPrime := L2 - L1
	deltaCPrime := C2Prime - C1Prime

	// Calculate Δh'
	var deltahPrime float64
	if C1Prime*C2Prime == 0 {
		deltahPrime = 0
	} else {
		diff := h2Prime - h1Prime
		if math.Abs(diff) <= 180 {
			deltahPrime = diff
		} else if diff > 180 {
			deltahPrime = diff - 360
		} else {
			deltahPrime = diff + 360
		}
	}

	// Calculate ΔH'
	deltaHPrime := 2 * math.Sqrt(C1Prime*C2Prime) * math.Sin(degToRad(deltahPrime/2))

	// Calculate L̄', C̄', h̄'
	LbarPrime := (L1 + L2) / 2
	CbarPrime := (C1Prime + C2Prime) / 2

	// Calculate h̄'
	var hbarPrime float64
	if C1Prime*C2Prime == 0 {
		hbarPrime = h1Prime + h2Prime
	} else {
		if math.Abs(h1Prime-h2Prime) <= 180 {
			hbarPrime = (h1Prime + h2Prime) / 2
		} else if h1Prime+h2Prime < 360 {
			hbarPrime = (h1Prime + h2Prime + 360) / 2
		} else {
			hbarPrime = (h1Prime + h2Prime - 360) / 2
		}
	}

	// Calculate T
	T := 1 - 0.17*math.Cos(degToRad(hbarPrime-30)) +
		0.24*math.Cos(degToRad(2*hbarPrime)) +
		0.32*math.Cos(degToRad(3*hbarPrime+6)) -
		0.20*math.Cos(degToRad(4*hbarPrime-63))

	// Calculate SL, SC, SH
	LbarPrime50Sq := (LbarPrime - 50) * (LbarPrime - 50)
	SL := 1 + (0.015*LbarPrime50Sq)/math.Sqrt(20+LbarPrime50Sq)
	SC := 1 + 0.045*CbarPrime
	SH := 1 + 0.015*CbarPrime*T

	// Calculate RT (rotation term for blue colors)
	deltaTheta := 30 * math.Exp(-math.Pow((hbarPrime-275)/25, 2))
	CbarPrime7 := math.Pow(CbarPrime, 7)
	RC := 2 * math.Sqrt(CbarPrime7/(CbarPrime7+6103515625))
	RT := -RC * math.Sin(degToRad(2*deltaTheta))

	// Parametric weighting factors (default = 1)
	kL, kC, kH := 1.0, 1.0, 1.0

	// Calculate final ΔE00
	termL := deltaLPrime / (kL * SL)
	termC := deltaCPrime / (kC * SC)
	termH := deltaHPrime / (kH * SH)

	deltaE := math.Sqrt(termL*termL + termC*termC + termH*termH + RT*termC*termH)

	return deltaE
}

// DeltaE76 calculates the CIE76 color difference (simple Euclidean distance in Lab space)
// Less accurate than CIEDE2000 but faster
func DeltaE76(lab1, lab2 Lab) float64 {
	dL := lab1.L - lab2.L
	dA := lab1.A - lab2.A
	dB := lab1.B - lab2.B
	return math.Sqrt(dL*dL + dA*dA + dB*dB)
}

// hueAngle calculates the hue angle in degrees from a* and b*
func hueAngle(a, b float64) float64 {
	if a == 0 && b == 0 {
		return 0
	}
	h := radToDeg(math.Atan2(b, a))
	if h < 0 {
		h += 360
	}
	return h
}

// degToRad converts degrees to radians
func degToRad(deg float64) float64 {
	return deg * math.Pi / 180
}

// radToDeg converts radians to degrees
func radToDeg(rad float64) float64 {
	return rad * 180 / math.Pi
}
