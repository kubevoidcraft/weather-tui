package ui

// sparkBlocks are the eight Unicode block characters used by sparkline,
// ordered from lowest to highest fill level. Together they give a sparkline
// eight distinct vertical levels in a single monospaced cell.
var sparkBlocks = []rune{'▁', '▂', '▃', '▄', '▅', '▆', '▇', '█'}

// shadeBlocks are five-level fill characters used to visualise a value on a
// well-known 0..100 domain (such as precipitation probability). The first
// entry is a hollow square so dry hours render as an empty frame - this
// keeps the row visibly occupied (confirming "data exists, the value is
// zero") without hinting at precipitation that isn't there.
var shadeBlocks = []rune{'◻', '░', '▒', '▓', '█'}

// shadeThresholds maps a probability percentage to an index in shadeBlocks.
// A value v picks shadeBlocks[i] where i is the largest index such that
// v >= shadeThresholds[i]. The 0 threshold guarantees every input maps to
// something, and 10 % is the cut-off below which the cell renders blank.
var shadeThresholds = []float64{0, 10, 30, 60, 85}

// sparkline renders the supplied values as a string of block characters. The
// input is linearly scaled to the [min, max] range found in values, so the
// result shows the relative shape of the series rather than absolute levels.
// For an all-flat series (or empty input) it emits mid-level blocks so the
// caller still gets a stable, same-width visual.
func sparkline(values []float64) string {
	if len(values) == 0 {
		return ""
	}

	minV, maxV := values[0], values[0]
	for _, v := range values[1:] {
		if v < minV {
			minV = v
		}
		if v > maxV {
			maxV = v
		}
	}

	out := make([]rune, len(values))
	span := maxV - minV
	if span == 0 {
		// Flat series - pick a middle block so the row is not misleadingly empty.
		mid := sparkBlocks[len(sparkBlocks)/2]
		for i := range out {
			out[i] = mid
		}
		return string(out)
	}

	// Scale each value into an index in sparkBlocks. Values equal to maxV map
	// to the top block; the rest get a proportional level.
	last := len(sparkBlocks) - 1
	for i, v := range values {
		idx := int(((v - minV) / span) * float64(last))
		if idx < 0 {
			idx = 0
		}
		if idx > last {
			idx = last
		}
		out[i] = sparkBlocks[idx]
	}
	return string(out)
}

// shaded renders values on a fixed 0..100 scale using five shade levels. It
// is used for precipitation probability where the absolute percentage is more
// informative than the relative shape - 5 % should look like no rain, and
// 95 % should look like solid rain. Values below the first non-zero threshold
// render as a blank space so the sparkline doesn't falsely suggest "a little
// bit of rain" when the forecast is effectively dry.
func shaded(values []float64) string {
	if len(values) == 0 {
		return ""
	}
	out := make([]rune, len(values))
	for i, v := range values {
		switch {
		case v < 0:
			v = 0
		case v > 100:
			v = 100
		}
		// Pick the highest threshold the value clears. Iterating from the top
		// lets us short-circuit on the first match.
		idx := 0
		for j := len(shadeThresholds) - 1; j >= 0; j-- {
			if v >= shadeThresholds[j] {
				idx = j
				break
			}
		}
		out[i] = shadeBlocks[idx]
	}
	return string(out)
}
