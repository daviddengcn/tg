package main

var index_html = `
<head>
<style>

body {
  padding: 0px;
  margin: 0px;
}

div.tbl {
  font-family: monospace;
  font-size: 10px;
  padding-top: 20px;
  background: rgba(255, 255, 255, 50);
}

div.tbl div.head {
  position: fixed;
  top: -1px;
  height: 30px;
  background: blue;
}

div.tbl div.ts {
  width: 150px;
  height: 5px;
  text-align: center;
  float: left;
  background: cyan;
}

div.tbl div.queries {
  width: 150px;
  float: left;
  background: yellow;
}

pre {
  padding-left: 2em;
  text-indent: -2em;
  margin: 2px 0px;
}

td {
  padding: 0px;
}

span.errmsg {
  color: red;
}


table.rowtable tr.rowtable:nth-child(6n+1), table.rowtable tr.rowtable:nth-child(6n+2), table.rowtable tr.rowtable:nth-child(6n+3) {
  background-color: #f7f7f7;
}

form {
	margin-bottom: 0px;
}

td, th {
	vertical-align: top;
	text-align: left;
	border:1px solid silver;
	word-break: keep-all;
	word-wrap: break-word;
	max-width: {{.ColumnWidth}}px;
	min-width: {{.ColumnWidth}}px;
}

td.ts, th.ts {
	max-width: 150px;
	min-width: 150px;
}

tr:hover td.ts {
  background-color: #ddd;
}

th.ts {
}

th {
	border: 1px solid white;
	background-color: white;
}

tbody {
	margin-top: 40px;
}

.s-1 {
	background-color: #ff7;
}
.s-2 {
	background-color: #f7f;
}
.s-3 {
	background-color: #7ff;
}
.s-4 {
	background-color: #77f;
}
.s-5 {
	background-color: #7f7;
}
.s-6 {
	background-color: #f77;
}

td pre:last-child {
	border-bottom: none;
}

p {
	padding-left: 2em;
	text-indent: -2em;
}

input {
	width: 100%;
}

input.newq {
	width: 200px;
}

div.toplink {
	position: fixed;
	top: 0px;
	right: 0px;
}

span.button {
  display: inline-block;
  width: 80px;
  height: 14px;
  text-align: center;
  cursor: default;
  border: solid 1px gray;
  border-radius: 5px;
}

span.button:hover {
  background: #eee;
}

</style>
</head>
<body>
<a href="#top">
<div class="toplink"><a href="#top">top</a></div>

<script>
function new_query(idx) {
  var q = prompt("New query")
  if (q) {
    var new_q = document.getElementById('new-q-' + idx);
    new_q.value = q
    document.getElementById('wholeform').submit();
  }
}
</script>

<div class="tbl">
  <div style="height: 50px; position: fixed; top: 0px; width: 9999px; background: rgba(220, 220, 220, 0.3);">
<form id="wholeform">
    {{$root := .}}
    <div style="float: left; width: 150px; height: 40px;">
      <div style="margin-top: 15px; height: 20px; text-align: center;">
        Timestamp
      </div>
    </div>
    <div style="float: left; width: {{.TotalHeadWidth}}px; height: 40px;">
      <input type="hidden" name="src-cnt" value="{{len .Sources}}">
      {{range $index, $source := .Sources}}
      <div style="width: {{$source.TotalWidth}}px; height: 20px; float: left;">
        <div style="">
          <input style="width: {{$source.SourceWidth}}px;" type="text" name="src-{{$index}}" value="{{$source.Source}}" placeholder="Source: Local path, Jenkins URL or remote server">
          <span class="button" onclick="new_query({{$index}})">New Query</span>
          <input type="hidden" name="new-q-{{$index}}" id="new-q-{{$index}}" value="">
        </div>
        <div style="clear: both; height: 20px;">
          {{range $source.Queries}}
          <div style="width: {{$root.ColumnWidth}}px; height: 20px; float: left">
            <input style="width: 100%;" type="search" name="q-{{$index}}" value="{{.}}" placeholder="leave empty to remove">
          </div>
          {{end}}
        </div>
      </div>
      {{end}}
      <div style="width: 400px; float:left">
        <input style="width: 100%;" type="text" name="new-src" value="" placeholder="Source: Local path, Jenkins URL or remote server">
        <input type="text" style="width: 150px" name="time-begin" value="{{.StartTime}}" placeholder="Start time">
        <input type="text" style="width: 150px" name="time-end" value="{{.EndTime}}" placeholder="End time">
        <button>Submit</button>
      </div>
    </div>
    <input type="hidden" name="cw" value="{{.ColumnWidth}}">
</form>
  </div>
  <div style="margin-top: 40px; clear: both;">
  <table class="rowtable" style="font-size: 10px; border-spacing: 0px; border: 0px; width: {{.TotalLogRowWidth}}px">
  {{range .Logs }}
    <tr class="rowtable">
      <td class="ts" style="border: 0px; padding: 0px;">
        {{.Time}}
      </td>
    {{range $srcidx, $srcpart := .Parts}}
      <td style="padding: 0px;border: 0px; width: {{(index $root.Sources $srcidx).TotalWidth}}px;">
        <div style="width: {{(index $root.Sources $srcidx).TotalWidth}}px">
        <table style="font-size: 10px; border: 0px; border-spacing: 0px; max-width: {{(index $root.Sources $srcidx).TotalWidth}}px">
          <tr>
          {{range $qidx, $qpart := $srcpart}}
            <td style="border: 0px; width: {{$root.ColumnWidth}}px; padding: 0px;">
              {{range .}}<pre>{{.Log}}</pre>{{end}}
            </td>
          {{end}}
          </tr>
        </table>
        </div>
      </td>
    {{end}}
    </tr>
  {{end}}
  </table>
  </div>
</div>

{{define "helpinfo"}}
[Data Source](/source)

Tips:

* Use <code>cw</code> parameter in URL to set width per column in pixels.
* Format of remote source: http://server.name:port/absolute/path/on/remote
* Clear a query (and submit) to remove a column.
{{end}}
{{markdown "helpinfo"}}
</body>
`

const jenkinsin_html = `
<head>
<style>
</style>
</head>
<body>
<div>
	{{.Info}}
</div>
<form>
  <div>
    <input type="text" name="localfn" placeholder="Local Path" size="200">
  </div>
  <div>
    <input type="text" name="jenkinsurl" placeholder="Jeknins URL" size="200">
  </div>
  <div>
    <button>Submit</button>
  </div>
</form>
</body>
`
