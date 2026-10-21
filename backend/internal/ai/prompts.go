package ai

import (
	"bytes"
	"embed"
	"strings"
	"text/template"
)

//go:embed prompts/*.tmpl
var promptFS embed.FS

// Prompt kinds; also the ai_calls.kind values.
const (
	KindRolePlan = "role_plan"
	KindMatching = "matching"
	KindRACI     = "raci"
)

var prompts = template.Must(template.New("").Funcs(template.FuncMap{"join": strings.Join}).ParseFS(promptFS, "prompts/*.tmpl"))

// PromptVersion returns the version declared inside a template
// ({{define "version"}}…{{end}}), so editing a prompt never needs Go changes.
func PromptVersion(kind string) string {
	var b bytes.Buffer
	t := template.Must(template.New("").Funcs(template.FuncMap{"join": strings.Join}).ParseFS(promptFS, "prompts/"+kind+".tmpl"))
	if err := t.ExecuteTemplate(&b, "version", nil); err != nil {
		return kind + "/unknown"
	}
	return strings.TrimSpace(b.String())
}

// RenderPrompt renders prompts/<kind>.tmpl with data.
func RenderPrompt(kind string, data any) (string, error) {
	var b bytes.Buffer
	if err := prompts.ExecuteTemplate(&b, kind+".tmpl", data); err != nil {
		return "", err
	}
	return strings.TrimSpace(b.String()), nil
}

type rolePlanData struct {
	RolePlanInput
	Deadline string
}

type raciData struct {
	RACIInput
	Deadline string
}

func renderRolePlan(in RolePlanInput) (string, error) {
	return RenderPrompt(KindRolePlan, rolePlanData{in, in.Deadline.Format("2 Jan 2006")})
}

func renderMatching(in MatchInput) (string, error) { return RenderPrompt(KindMatching, in) }

func renderRACI(in RACIInput) (string, error) {
	return RenderPrompt(KindRACI, raciData{in, in.Deadline.Format("2 Jan 2006")})
}
