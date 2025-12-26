package ui

import "fyne.io/fyne/v2"

// proportionalLayout распределяет элементы пропорционально
type proportionalLayout struct {
	proportions []float32
}

func (p *proportionalLayout) MinSize(objects []fyne.CanvasObject) fyne.Size {
	height := float32(30)
	for _, obj := range objects {
		if obj.MinSize().Height > height {
			height = obj.MinSize().Height
		}
	}
	return fyne.NewSize(400, height)
}

func (p *proportionalLayout) Layout(objects []fyne.CanvasObject, size fyne.Size) {
	if len(objects) == 0 {
		return
	}

	// Вычисляем сумму пропорций
	var totalProportion float32
	for i := 0; i < len(objects) && i < len(p.proportions); i++ {
		totalProportion += p.proportions[i]
	}

	// Распределяем ширину
	x := float32(0)
	for i, obj := range objects {
		proportion := float32(1)
		if i < len(p.proportions) {
			proportion = p.proportions[i]
		}

		width := (proportion / totalProportion) * size.Width
		obj.Resize(fyne.NewSize(width, size.Height))
		obj.Move(fyne.NewPos(x, 0))
		x += width
	}
}

