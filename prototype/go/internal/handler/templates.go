package handler

// htmlTemplates contains all HTML page templates used by the Handler.
// The templates share a header/footer pair to keep them DRY.
const htmlTemplates = `
{{define "header"}}<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>Menata Runtime</title>
<script src="https://unpkg.com/htmx.org@2.0.4" defer></script>
<style>
*{box-sizing:border-box;margin:0;padding:0}
body{font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',sans-serif;background:#f8fafc;color:#1e293b;padding:1rem}
nav{display:flex;gap:1rem;align-items:center;padding:.75rem 1rem;background:#fff;border-radius:.5rem;margin-bottom:1.5rem;box-shadow:0 1px 3px rgba(0,0,0,.08)}
nav a{color:#2563eb;text-decoration:none;font-size:.875rem}
nav a:hover{text-decoration:underline}
nav .spacer{flex:1}
nav .role{font-size:.8rem;background:#eff6ff;color:#1d4ed8;padding:.25rem .6rem;border-radius:9999px}
.card{background:#fff;border-radius:.5rem;box-shadow:0 1px 3px rgba(0,0,0,.08);padding:1.25rem;margin-bottom:1rem}
h1{font-size:1.25rem;font-weight:600;margin-bottom:1rem}
h2{font-size:1rem;font-weight:600;margin-bottom:.75rem}
table{width:100%;border-collapse:collapse;font-size:.875rem}
th{text-align:left;padding:.5rem .75rem;background:#f1f5f9;font-weight:600;border-bottom:2px solid #e2e8f0}
td{padding:.5rem .75rem;border-bottom:1px solid #f1f5f9}
tr:hover td{background:#fafbfc}
.btn{display:inline-block;padding:.375rem .875rem;border:none;border-radius:.375rem;cursor:pointer;font-size:.8rem;font-weight:500;text-decoration:none;line-height:1.5}
.btn-primary{background:#2563eb;color:#fff}
.btn-primary:hover{background:#1d4ed8}
.btn-sm{background:#f1f5f9;color:#475569}
.btn-sm:hover{background:#e2e8f0}
.btn-event{background:#0f172a;color:#fff;margin-right:.375rem}
.btn-event:hover{background:#1e293b}
.actions{display:flex;gap:.5rem;align-items:center;margin-bottom:1rem;flex-wrap:wrap}
.field-group{margin-bottom:1rem}
label{display:block;font-size:.8rem;font-weight:500;color:#475569;margin-bottom:.25rem}
input,select,textarea{width:100%;padding:.4rem .625rem;border:1px solid #cbd5e1;border-radius:.375rem;font-size:.875rem;background:#fff}
input:focus,select:focus,textarea:focus{outline:none;border-color:#2563eb;box-shadow:0 0 0 2px rgba(37,99,235,.15)}
textarea{min-height:80px;resize:vertical}
.errors{background:#fef2f2;border:1px solid #fecaca;border-radius:.375rem;padding:.75rem 1rem;margin-bottom:1rem;font-size:.875rem;color:#dc2626}
.errors li{margin-left:1.25rem}
.badge{display:inline-block;padding:.125rem .5rem;border-radius:9999px;font-size:.75rem;font-weight:500;background:#e2e8f0;color:#475569}
.badge-draft{background:#f1f5f9;color:#64748b}
.badge-submitted{background:#dbeafe;color:#1d4ed8}
.badge-accepted{background:#dcfce7;color:#15803d}
.badge-rejected{background:#fee2e2;color:#b91c1c}
.badge-in_progress,.badge-in-progress{background:#fef9c3;color:#854d0e}
.badge-completed{background:#d1fae5;color:#065f46}
dl{display:grid;grid-template-columns:max-content 1fr;gap:.375rem 1rem;font-size:.875rem}
dt{color:#64748b;font-weight:500}
dd{color:#1e293b}
.machine-grid{display:grid;grid-template-columns:repeat(auto-fill,minmax(220px,1fr));gap:1rem}
.machine-card{background:#fff;border-radius:.5rem;border:1px solid #e2e8f0;padding:1rem;text-decoration:none;color:inherit;display:block}
.machine-card:hover{border-color:#2563eb;box-shadow:0 0 0 2px rgba(37,99,235,.1)}
.machine-card h3{font-size:.9375rem;font-weight:600;margin-bottom:.25rem}
.machine-card p{font-size:.8rem;color:#64748b}
</style>
</head>
<body>
<nav>
  <a href="/"><strong>Menata Runtime</strong></a>
  <span class="spacer"></span>
  <span class="role">{{.Role}}</span>
  <a href="/login">Switch role</a>
</nav>
<main>
{{end}}

{{define "footer"}}
</main>
</body>
</html>
{{end}}

{{define "home"}}
{{template "header" .}}
<div class="card">
  <h1>Machines</h1>
  <div class="machine-grid">
  {{range .Machines}}
  <a class="machine-card" href="/{{.ID}}">
    <h3>{{.Name}}</h3>
    <p>{{len .Fields}} fields · {{len .Events}} events</p>
  </a>
  {{end}}
  </div>
</div>
{{template "footer" .}}
{{end}}

{{define "login"}}
{{template "header" .}}
<div class="card" style="max-width:360px">
  <h1>Switch role</h1>
  <p style="font-size:.875rem;color:#64748b;margin-bottom:1rem">
    Prototype: choose which role to act as. No password required.
  </p>
  <form method="POST" action="/login">
    <div class="field-group">
      <label for="role">Role</label>
      <select id="role" name="role">
        <option value="Requester" {{if eq .Role "Requester"}}selected{{end}}>Requester</option>
        <option value="Designer"  {{if eq .Role "Designer"}}selected{{end}}>Designer</option>
      </select>
    </div>
    <button type="submit" class="btn btn-primary">Confirm</button>
  </form>
</div>
{{template "footer" .}}
{{end}}

{{define "list"}}
{{template "header" .}}
<div class="card">
  <div class="actions">
    <h1 style="margin:0">{{.Machine.Name}}</h1>
    <span class="spacer"></span>
    <a href="/{{.Machine.ID}}/new" class="btn btn-primary">+ New</a>
  </div>
  {{if .Rows}}
  <table>
    <thead>
      <tr>
        {{range .Columns}}<th>{{.Name}}</th>{{end}}
        <th style="width:1%"></th>
      </tr>
    </thead>
    <tbody>
      {{range .Rows}}
      <tr>
        {{range .Values}}
        <td>{{if .}}<span class="badge badge-{{.}}">{{.}}</span>{{end}}</td>
        {{end}}
        <td><a href="/{{$.Machine.ID}}/{{.ID}}" class="btn btn-sm">View</a></td>
      </tr>
      {{end}}
    </tbody>
  </table>
  {{else}}
  <p style="color:#94a3b8;font-size:.875rem;padding:1rem 0">No records yet.</p>
  {{end}}
</div>
{{template "footer" .}}
{{end}}

{{define "form"}}
{{template "header" .}}
<div class="card" style="max-width:560px">
  <h1>New {{.Machine.Name}}</h1>
  {{if .Errors}}
  <div class="errors">
    <strong>Please fix the following:</strong>
    <ul>{{range .Errors}}<li>{{.}}</li>{{end}}</ul>
  </div>
  {{end}}
  <form method="POST" action="/{{.Machine.ID}}">
    {{range .Fields}}
    <div class="field-group">
      <label for="{{.Field.ID}}">
        {{.Field.Name}}{{if .Field.Required}} <span style="color:#dc2626">*</span>{{end}}
      </label>
      {{if eq .Field.Type "rich_text"}}
        <textarea id="{{.Field.ID}}" name="{{.Field.ID}}">{{.Value}}</textarea>
      {{else if eq .Field.Type "value_list"}}
        <select id="{{.Field.ID}}" name="{{.Field.ID}}">
          <option value="">— select —</option>
          {{range .Field.Options.Values}}
          <option value="{{.}}" {{if eq . $.Value}}selected{{end}}>{{.}}</option>
          {{end}}
        </select>
      {{else if eq .Field.Type "date"}}
        <input type="date" id="{{.Field.ID}}" name="{{.Field.ID}}" value="{{.Value}}">
      {{else if eq .Field.Type "file"}}
        <input type="file" id="{{.Field.ID}}" name="{{.Field.ID}}">
      {{else if eq .Field.Type "user"}}
        <input type="text" id="{{.Field.ID}}" name="{{.Field.ID}}" value="{{.Value}}"
               placeholder="User name or email">
      {{else}}
        <input type="text" id="{{.Field.ID}}" name="{{.Field.ID}}" value="{{.Value}}">
      {{end}}
    </div>
    {{end}}
    <div class="actions" style="margin-top:1.25rem">
      <button type="submit" class="btn btn-primary">Create</button>
      <a href="/{{.Machine.ID}}" class="btn btn-sm">Cancel</a>
    </div>
  </form>
</div>
{{template "footer" .}}
{{end}}

{{define "detail"}}
{{template "header" .}}
<div class="card">
  <div class="actions" style="margin-bottom:1.25rem">
    <a href="/{{.Machine.ID}}" class="btn btn-sm">← Back</a>
    <h1 style="margin:0">{{.Machine.Name}}</h1>
  </div>
  <dl>
    {{range .Fields}}
    <dt>{{.Name}}</dt>
    <dd>{{if .Value}}<span class="badge badge-{{.Value}}">{{.Value}}</span>{{else}}<span style="color:#94a3b8">—</span>{{end}}</dd>
    {{end}}
  </dl>
  {{if .PermittedEvents}}
  <div class="actions" style="margin-top:1.5rem;padding-top:1rem;border-top:1px solid #f1f5f9">
    {{range .PermittedEvents}}
    <form method="POST" action="/{{$.Machine.ID}}/{{$.Record.ID}}/events/{{.ID}}" style="display:inline">
      <button type="submit" class="btn btn-event">{{.Name}}</button>
    </form>
    {{end}}
  </div>
  {{end}}
</div>
{{template "footer" .}}
{{end}}
`
