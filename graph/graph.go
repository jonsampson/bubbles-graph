package graph

import (
	"container/ring"
	"strings"
	"sync"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Internal ID management. Used during animating to ensure that frame messages
// are received only by spinner components that sent them.
var (
	lastID int
	idMtx  sync.Mutex
)

// Return the next ID we should use on the Model.
func nextID() int {
	idMtx.Lock()
	defer idMtx.Unlock()
	lastID++
	return lastID
}

type graph struct {
	greatestY int
	Inverted  bool
	maxX      int
	Style     lipgloss.Style
	yValues   *ring.Ring
}

type Model struct {
	Graph graph
	id    int
	Style lipgloss.Style
}

type UpdateMessage struct {
	Time   time.Time
	ID     int
	YValue int
}

func New() Model {
	yValues := ring.New(100)
	for i := 0; i < 100; i++ {
		yValues.Value = 0
		yValues = yValues.Next()
	}
	return Model{
		Graph: graph{
			maxX:    0,
			Style:   lipgloss.NewStyle().Align(lipgloss.Center),
			yValues: yValues,
		},
		id:    nextID(),
		Style: lipgloss.NewStyle().Align(lipgloss.Center),
	}
}

func (m Model) WithAutoScale() Model {
	m.Graph.maxX = 0
	return m
}

func (m Model) WithMaxScale(max int) Model {
	m.Graph.maxX = max
	return m
}

func (m Model) Init() tea.Cmd {
	return m.update(m.id)
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case UpdateMessage:
		// Ignore messages that aren't for us.
		if msg.ID != m.id {
			return m, nil
		}
		cmds = append(cmds, m.update(m.id))
	}
	return m, tea.Batch(cmds...)
}

type remainderValue int

type inversion bool

const (
	r0 remainderValue = iota
	r1
	r2
	r3
	r4
	r5
	r6
	r7
	r8
	r9

	rightsideUp inversion = false
	upsideDown inversion = true
)

var remainderRune = map[remainderValue]map[inversion]string{
	r0: {rightsideUp: "⣿", upsideDown: "⣿"},
	r1: {rightsideUp: "⢀", upsideDown: "⠈"},
	r2: {rightsideUp: "⢠", upsideDown: "⠘"},
	r3: {rightsideUp: "⢰", upsideDown: "⠸"},
	r4: {rightsideUp: "⢸", upsideDown: "⢸"},
	r5: {rightsideUp: "⣸", upsideDown: "⢹"},
	r6: {rightsideUp: "⣸", upsideDown: "⢹"},
	r7: {rightsideUp: "⣼", upsideDown: "⢻"},
	r8: {rightsideUp: "⣼", upsideDown: "⢻"},
	r9: {rightsideUp: "⣾", upsideDown: "⢿"},
}

func (m Model) View() string {
	w := m.Graph.Style.GetWidth()
	h := m.Graph.Style.GetHeight()
	pw, ph := m.Graph.Style.GetFrameSize()
	w -= pw
	if w < 0 {
		w = 0
	}
	h -= ph
	if h < 0 {
		h = 0
	}
	runes := make([][]string, m.Graph.yValues.Len())
	for i := range runes {
		runes[i] = make([]string, h)
		for j := range runes[i] {
			runes[i][j] = " "
		}
	}

	xIndex := 0
	m.Graph.yValues.Do(func(p any) {
		y := p.(int)
		if y == 0 {
			return
		}
		if m.Graph.maxX != 0 {
			y = y * h / m.Graph.maxX
		} else if m.Graph.greatestY != 0 {
			y = y * h / m.Graph.greatestY
		}
		if y > h {
			y = h
		}
		for i := 0; i < y; i++ {
			var yIndex int
			if m.Graph.Inverted {
				yIndex = i
			} else {
				yIndex = h - 1 - i
			}
			if i < y-1 {
				runes[xIndex][yIndex] = "⣿"
			} else {
				yRemainder := remainderValue(y % h)
				runes[xIndex][yIndex] = remainderRune[yRemainder][inversion(m.Graph.Inverted)]
			}
		}
		xIndex++
	})

	var sb strings.Builder
	for i := 0; i < h; i++ {
		for j := 0; j < w; j++ {
			_, err := sb.WriteString(runes[j][i])
			if err != nil {
				panic(err)
			}
		}
		_, err := sb.WriteString("\n")
		if err != nil {
			panic(err)
		}
	}
	v := m.Style.Render(
		m.Graph.Style.Render(sb.String()),
	)
	return v
}

func (m *Model) ID() int {
	return m.id
}

func (m Model) SetSize(w, h int) Model {
	m.Style = m.Style.Width(w).Height(h)
	width := w - 2
	m.Graph.Style = m.Graph.Style.Width(width).Height(h - 2)
	newYValues := ring.New(width)
	for i := 0; i < width; i++ {
		newYValues.Value = 0
		newYValues = newYValues.Next()
	}

	n := m.Graph.yValues.Len()
	for i := 0; i < n; i++ {
		if i <= width {
			newYValues.Value = m.Graph.yValues.Value
			newYValues = newYValues.Prev()
			m.Graph.yValues = m.Graph.yValues.Prev()
		}
	}
	m.Graph.yValues = newYValues
	return m
}

func (m *Model) AddNextValue(yValue int) tea.Cmd {
	m.Graph.yValues.Value = yValue
	m.Graph.yValues = m.Graph.yValues.Next()
	m.Graph.greatestY = 0

	m.Graph.yValues.Do(func(p any) {
		v := p.(int)
		if v > m.Graph.greatestY {
			m.Graph.greatestY = v
		}
	})
	return m.update(m.id)
}

func (m *Model) update(id int) tea.Cmd {
	return func() tea.Msg {
		return UpdateMessage{
			Time: time.Now(),
			ID:   id,
		}
	}
}
