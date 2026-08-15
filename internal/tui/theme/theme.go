package theme

import "github.com/charmbracelet/lipgloss"

type Theme struct {
	SelectedBackground lipgloss.TerminalColor
	SurfaceBackground  lipgloss.TerminalColor

	PrimaryBorder   lipgloss.TerminalColor
	SecondaryBorder lipgloss.TerminalColor
	FaintBorder     lipgloss.TerminalColor

	PrimaryText   lipgloss.TerminalColor
	SecondaryText lipgloss.TerminalColor
	FaintText     lipgloss.TerminalColor
	InvertedText  lipgloss.TerminalColor

	SuccessText lipgloss.TerminalColor
	WarningText lipgloss.TerminalColor
	ErrorText   lipgloss.TerminalColor
}

func Default() Theme {
	return ShellHub()
}

func ShellHub() Theme {
	return Theme{
		SelectedBackground: lipgloss.Color("#667ACC"),
		SurfaceBackground:  lipgloss.Color("#22252B"),

		PrimaryBorder:   lipgloss.Color("#667ACC"),
		SecondaryBorder: lipgloss.Color("#383D47"),
		FaintBorder:     lipgloss.Color("#2C2F36"),

		PrimaryText:   lipgloss.Color("#E1E4EA"),
		SecondaryText: lipgloss.Color("#667ACC"),
		FaintText:     lipgloss.Color("#8B8F99"),
		InvertedText:  lipgloss.Color("#18191B"),

		SuccessText: lipgloss.Color("#82A568"),
		WarningText: lipgloss.Color("#BF8C5D"),
		ErrorText:   lipgloss.Color("#D8737B"),
	}
}
