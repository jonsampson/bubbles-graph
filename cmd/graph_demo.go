package main

// A simple program demonstrating the spinner component from the Bubbles
// component library.

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/NVIDIA/go-nvml/pkg/nvml"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/jonsampson/bubbles-graph/graph"
	"github.com/shirou/gopsutil/v4/cpu"
)

type TickMsg time.Time

func (m model) doTick() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return TickMsg(t)
	})
}

type errMsg error

type model struct {
	cpuGraph graph.Model
	gpuGraph graph.Model
	quitting bool
	err      error
}

func initialModel() model {
	s := graph.New().WithMaxScale(100)
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))
	g := graph.New().WithMaxScale(100)
	g.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("105"))
	g.Graph.Inverted = true
	return model{cpuGraph: s, gpuGraph: g}
}

func (m model) Init() tea.Cmd {
	return m.doTick()
}

func (m model) Update(mesg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := mesg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "esc", "ctrl+c":
			m.quitting = true
			return m, tea.Quit
		default:
			return m, nil
		}

	case tea.WindowSizeMsg:
		cpuH, cpuW := m.cpuGraph.Style.GetFrameSize()
		gpuH, gpuW := m.gpuGraph.Style.GetFrameSize()
		halfHeight := msg.Height / 2
		m.cpuGraph = m.cpuGraph.SetSize(msg.Width-cpuW, halfHeight-cpuH)
		m.gpuGraph = m.gpuGraph.SetSize(msg.Width-gpuW, halfHeight-gpuH)

		return m, nil

	case TickMsg:
		var ccmd, gcmd tea.Cmd
		c, _ := cpu.Percent(time.Second, false)
		m.cpuGraph, ccmd = m.cpuGraph.Update(m.cpuGraph.AddNextValue(int(c[0])))
		device0, ret := nvml.DeviceGetHandleByIndex(0)
		if ret != nvml.SUCCESS {
			log.Fatalf("Failed to get device handle: %v", ret)
		}
		sample, ret := nvml.DeviceGetUtilizationRates(device0)
		if ret != nvml.SUCCESS {
			log.Fatalf("Failed to get vGPU utilization: %v", ret)
		}
		m.gpuGraph, gcmd = m.gpuGraph.Update(m.gpuGraph.AddNextValue(int(sample.Gpu)))
		return m, tea.Batch(m.doTick(), ccmd, gcmd)

	case errMsg:
		m.err = msg
		return m, nil

	default:
		var ccmd, gcmd tea.Cmd
		m.cpuGraph, ccmd = m.cpuGraph.Update(msg)
		m.gpuGraph, gcmd = m.gpuGraph.Update(msg)
		return m, tea.Batch(ccmd, gcmd)
	}
}

func (m model) View() string {
	if m.err != nil {
		return m.err.Error()
	}
	// str := fmt.Sprintf("\n\n   %s Loading forever...press q to quit\n\n", m.cpuGraph.View())
	str := lipgloss.JoinVertical(lipgloss.Center,
		m.cpuGraph.View(),
		m.gpuGraph.View(),
	)
	if m.quitting {
		return str + "\n"
	}
	return str
}

func main() {
	ret := nvml.Init()
	if ret != nvml.SUCCESS {
		log.Fatalf("Failed to initialize NVML: %v", ret)
	}
	defer func() {
		ret := nvml.Shutdown()
		if ret != nvml.SUCCESS {
			log.Fatalf("Unable to shutdown NVML: %v", nvml.ErrorString(ret))
		}
	}()
	p := tea.NewProgram(initialModel())
	if _, err := p.Run(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
