package ui

import "fmt"

// ProgressBar renders a simple terminal progress bar.
type ProgressBar struct {
	total   int
	current int
	width   int
}

// NewProgressBar creates a new ProgressBar.
func NewProgressBar(total, width int) *ProgressBar {
	return &ProgressBar{total: total, width: width}
}

// Update sets the current progress and renders.
func (p *ProgressBar) Update(current int) {
	p.current = current
	p.render()
}

func (p *ProgressBar) render() {
	if p.total == 0 {
		return
	}
	pct := float64(p.current) / float64(p.total)
	filled := int(pct * float64(p.width))
	bar := ""
	for i := 0; i < p.width; i++ {
		if i < filled {
			bar += "█"
		} else {
			bar += "░"
		}
	}
	fmt.Printf("\r  [%s] %d/%d (%.0f%%)", bar, p.current, p.total, pct*100)
	if p.current >= p.total {
		fmt.Println()
	}
}
