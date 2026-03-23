package tui

import "github.com/charmbracelet/lipgloss"

// Color palette
var (
	colorPass    = lipgloss.Color("#22c55e") // green-500
	colorFail    = lipgloss.Color("#ef4444") // red-500
	colorSkip    = lipgloss.Color("#eab308") // yellow-500
	colorPending = lipgloss.Color("#6b7280") // gray-500
	colorRunning = lipgloss.Color("#06b6d4") // cyan-500
	colorMuted   = lipgloss.Color("#475569") // slate-600
	colorBorder  = lipgloss.Color("#334155") // slate-700
	colorBg      = lipgloss.Color("#0f172a") // slate-900
	colorText    = lipgloss.Color("#f1f5f9") // slate-100
	colorSubtext = lipgloss.Color("#94a3b8") // slate-400
	colorBlue    = lipgloss.Color("#3b82f6") // blue-500
	colorCyan    = lipgloss.Color("#22d3ee") // cyan-400
)

// Status bar
var statusBarStyle = lipgloss.NewStyle().
	Background(lipgloss.Color("#1e293b")).
	Foreground(colorText).
	Padding(0, 1)

var statusBarTitleStyle = lipgloss.NewStyle().
	Background(colorBlue).
	Foreground(lipgloss.Color("#ffffff")).
	Bold(true).
	Padding(0, 1)

// Key bar
var keyBarStyle = lipgloss.NewStyle().
	Background(lipgloss.Color("#1e293b")).
	Foreground(colorMuted).
	Padding(0, 1)

var keyStyle = lipgloss.NewStyle().
	Foreground(colorSubtext)

var keyDescStyle = lipgloss.NewStyle().
	Foreground(colorMuted)

// Panel borders
var panelStyle = lipgloss.NewStyle().
	Border(lipgloss.RoundedBorder()).
	BorderForeground(colorBorder)

var activePanelStyle = lipgloss.NewStyle().
	Border(lipgloss.RoundedBorder()).
	BorderForeground(colorBlue)

// List items
var selectedItemStyle = lipgloss.NewStyle().
	Background(lipgloss.Color("#1e3a5f")).
	Foreground(lipgloss.Color("#e0f2fe"))

var groupHeaderStyle = lipgloss.NewStyle().
	Foreground(colorSubtext).
	Bold(true)

// Section headers in detail pane
var sectionHeaderStyle = lipgloss.NewStyle().
	Foreground(colorSubtext).
	Bold(true).
	MarginTop(1)

// Method colors
var methodStyle = lipgloss.NewStyle().
	Foreground(colorCyan).
	Bold(true)

var urlStyle = lipgloss.NewStyle().
	Foreground(colorText)

var headerKeyStyle = lipgloss.NewStyle().
	Foreground(colorSubtext)

var headerValStyle = lipgloss.NewStyle().
	Foreground(colorPending)

// Response status
var statusOKStyle = lipgloss.NewStyle().
	Foreground(colorPass).
	Bold(true)

var statusErrStyle = lipgloss.NewStyle().
	Foreground(colorFail).
	Bold(true)

var durationStyle = lipgloss.NewStyle().
	Foreground(colorMuted)

// Assertion results
var passIconStyle = lipgloss.NewStyle().
	Foreground(colorPass)

var failIconStyle = lipgloss.NewStyle().
	Foreground(colorFail)

var failMsgStyle = lipgloss.NewStyle().
	Foreground(colorFail)

var errorBoxStyle = lipgloss.NewStyle().
	Border(lipgloss.RoundedBorder()).
	BorderForeground(colorFail).
	Foreground(colorFail).
	Padding(0, 1)

var mutedStyle = lipgloss.NewStyle().
	Foreground(colorMuted)

var bodyStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color("#cbd5e1"))

var detailTitleStyle = lipgloss.NewStyle().
	Foreground(colorText).
	Bold(true)
