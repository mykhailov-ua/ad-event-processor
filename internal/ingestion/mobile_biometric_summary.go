package ingestion

import "ad-event-processor/internal/domain"

const (
	mobileGyroFlatThreshold  = 2
	mobileGyroMinFlatSamples = 3
)

type mobileBiometricSummary struct {
	touchCount   uint8
	gyroSamples  uint8
	gyroVariance uint16
	gyroFlat     uint8
}

func summarizeMobileBiometrics(events []domain.BehaviorTelemetryEvent) mobileBiometricSummary {
	var sum mobileBiometricSummary
	if len(events) == 0 {
		return sum
	}

	var minX, maxX, minY, maxY int
	var gyroRangeInit bool
	var sumVal, sumSq int64
	var gyroN int

	for i := range events {
		e := events[i]
		switch e.T {
		case "touchstart", "touchmove":
			if sum.touchCount < 255 {
				sum.touchCount++
			}
		case "deviceorientation", "devicemotion":
			if sum.gyroSamples < 255 {
				sum.gyroSamples++
			}
			if !gyroRangeInit {
				minX, maxX = e.X, e.X
				minY, maxY = e.Y, e.Y
				gyroRangeInit = true
			} else {
				if e.X < minX {
					minX = e.X
				}
				if e.X > maxX {
					maxX = e.X
				}
				if e.Y < minY {
					minY = e.Y
				}
				if e.Y > maxY {
					maxY = e.Y
				}
			}
			v := int64(e.X + e.Y)
			sumVal += v
			sumSq += v * v
			gyroN++
		}
	}

	if gyroN > 1 {
		mean := sumVal / int64(gyroN)
		variance := float64(sumSq)/float64(gyroN) - float64(mean)*float64(mean)
		if variance < 0 {
			variance = 0
		}
		if variance > 65535 {
			variance = 65535
		}
		sum.gyroVariance = uint16(variance)
	}

	if sum.gyroSamples >= mobileGyroMinFlatSamples && gyroRangeInit {
		if maxX-minX <= mobileGyroFlatThreshold && maxY-minY <= mobileGyroFlatThreshold {
			sum.gyroFlat = 1
		}
	}
	return sum
}

func applyMobileBiometricSummary(evt *domain.Event) {
	if evt == nil || evt.TelemetrySet == 0 {
		return
	}
	sum := summarizeMobileBiometrics(evt.TelemetryEvents)
	evt.MobileTouchCount = sum.touchCount
	evt.MobileGyroSamples = sum.gyroSamples
	evt.MobileGyroVariance = sum.gyroVariance
	evt.MobileGyroFlat = sum.gyroFlat
	evt.MobileBiometricSet = 1
	if scanUAFamily(evt.UA) == uaFamilyMobile {
		evt.MobileBiometricMobile = 1
	}
}
