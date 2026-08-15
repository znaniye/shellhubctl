package tui

import (
	"context"

	"github.com/znaniye/shellhubctl/internal/auth"
	"github.com/znaniye/shellhubctl/internal/shellhub"
	"github.com/znaniye/shellhubctl/internal/tui/theme"
)

const (
	headerHeight             = 3
	footerHeight             = 1
	contentHorizontalPadding = 1
	minBodyHeight            = minTableHeight + 1
	minMainContentWidth      = 40
	minMainContentHeight     = headerHeight + minBodyHeight
)

type ProgramContext struct {
	Ctx    context.Context
	Client *shellhub.Client
	Store  *auth.Store

	Theme  theme.Theme
	Styles Styles

	ScreenWidth  int
	ScreenHeight int

	MainContentWidth  int
	MainContentHeight int

	BodyHeight int
}

func newProgramContext(ctx context.Context, opts Options) *ProgramContext {
	t := theme.Default()

	pctx := &ProgramContext{
		Ctx:    ctx,
		Client: opts.Client,
		Store:  opts.Store,
		Theme:  t,
		Styles: InitStyles(t),
	}

	pctx.SetSize(defaultWidth, defaultHeight)

	return pctx
}

func (c *ProgramContext) SetSize(width, height int) {
	c.ScreenWidth = width
	c.ScreenHeight = height
	c.MainContentWidth = max(width-contentHorizontalPadding*2, 0)
	c.MainContentHeight = max(height-footerHeight, 0)

	c.setBodyHeight(c.MainContentHeight - headerHeight)
}

func (c *ProgramContext) setBodyHeight(height int) {
	c.BodyHeight = max(height, 0)
}

func (c *ProgramContext) TooSmall() bool {
	return c.MainContentWidth < minMainContentWidth ||
		c.MainContentHeight < minMainContentHeight
}
