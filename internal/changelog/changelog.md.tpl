{{define "entry" -}}
- {{ if .BreakingChange}}**BREAKING**: {{end}}{{ if .Scope }}**{{.Scope}}**: {{end}}{{.Description}} ([{{.ShortHash}}]({{.URL}}))
{{ end }}

{{- if not .Formatting.HideVersionTitle }}
## [{{.Data.Version}}]({{.Data.VersionLink}})
{{ if .Data.CompareURL }}
[Compare to previous version]({{.Data.CompareURL}})
{{ end -}}
{{ end -}}
{{- if .Data.Prefix }}
{{ .Data.Prefix }}
{{ end -}}
{{- range .Data.Sections }}
### {{ .Title }}

{{ range .Commits -}}{{template "entry" .}}{{end}}
{{- end -}}

{{- if .Data.Suffix }}
{{ .Data.Suffix }}
{{ end }}
