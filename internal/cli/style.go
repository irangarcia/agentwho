package cli

import "github.com/irangarcia/agentwho/internal/termstyle"

func (a *app) accent(text string) string  { return termstyle.Paint(a.out, termstyle.Accent, text) }
func (a *app) bold(text string) string    { return termstyle.Paint(a.out, termstyle.Bold, text) }
func (a *app) info(text string) string    { return termstyle.Paint(a.out, termstyle.Info, text) }
func (a *app) success(text string) string { return termstyle.Paint(a.out, termstyle.Success, text) }
func (a *app) warning(text string) string { return termstyle.Paint(a.out, termstyle.Warning, text) }
func (a *app) danger(text string) string  { return termstyle.Paint(a.out, termstyle.Danger, text) }
func (a *app) muted(text string) string   { return termstyle.Paint(a.out, termstyle.Muted, text) }
