package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/daviddengcn/go-villa"
	"github.com/russross/blackfriday"
)

const (
	MAX_LINES = 10000
)

var (
	templates   *template.Template
	gLogFile    villa.Path
	gJenkins    []byte
	gSource     string
	gTotalLines int = -1
)

func loadTemplates() {
	templates = template.New("templates").Funcs(template.FuncMap{
		"markdown": Markdown,
	})

	template.Must(templates.New("index.html").Parse(index_html))

	template.Must(templates.New("jenkinsin.html").Parse(jenkinsin_html))
}

func init() {
	loadTemplates()
}

var (
	reIndexHtml = regexp.MustCompile("var index_html = `((?s:[^`]*))`")
)

func loadTemplatesFromDisk() {
	src, err := ioutil.ReadFile("./tools/tg/html.go")
	if err != nil {
		return
	}
	mch := reIndexHtml.FindSubmatch(src)
	if len(mch) < 2 {
		log.Printf("Didn't find index_html")
		return
	}
	index_html = string(mch[1])
	loadTemplates()
}

func Markdown(templ string) template.HTML {
	var out villa.ByteSlice
	templates.ExecuteTemplate(&out, templ, nil)
	return template.HTML(blackfriday.MarkdownCommon(out))
}

func init() {
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)
}

type Cell struct {
	Log template.HTML
}

type LogRow struct {
	Time string
	// Parts over queries
	Parts [][][]Cell
}

type SourceInfo struct {
	Source  string
	Queries []string

	TotalWidth  int
	SourceWidth int
}

func (si *SourceInfo) UpdateUI(cw int) {
	wdQueries := cw * len(si.Queries)
	if wdQueries < 400 {
		wdQueries = 400
	}
	si.TotalWidth = wdQueries
	si.SourceWidth = si.TotalWidth - 80 - 10
}

func (si *SourceInfo) Type() string {
	if strings.HasPrefix(si.Source, "http://ci-builds") {
		return "jenkins"
	}
	if strings.HasPrefix(si.Source, "http://") {
		return "remote"
	}
	return "local"
}

type UIInfo struct {
	TotalLogRowWidth int
	TotalHeadWidth   int
	ColumnWidth      int
}

func calcUIInfo(sources []SourceInfo) (info UIInfo) {
	wd := 0
	for _, source := range sources {
		wd += source.TotalWidth
	}
	info.TotalLogRowWidth = wd + 150
	info.TotalHeadWidth = info.TotalLogRowWidth + 400
	return
}

func strToIntDef(s string, def int) (int, bool) {
	if s == "" {
		return def, false
	}
	vl, err := strconv.Atoi(s)
	if err != nil {
		log.Printf("Atoi(%s) failed: %v", s, err)
		return def, false
	}
	return vl, true
}

func parseInput(r *http.Request) (sources []SourceInfo, uiInfo UIInfo) {
	cw, _ := strToIntDef(r.FormValue("cw"), 200)

	srcCnt, _ := strToIntDef(r.FormValue("src-cnt"), 0)

	for i := 0; i < srcCnt; i++ {
		source := r.FormValue(fmt.Sprintf("src-%d", i))
		if source == "" {
			break
		}
		si := SourceInfo{
			Source:  source,
			Queries: removeEmpty(r.Form[fmt.Sprintf("q-%d", i)]),
		}

		newQ := r.FormValue(fmt.Sprintf("new-q-%d", i))
		if newQ != "" {
			si.Queries = append(si.Queries, newQ)
		}

		si.UpdateUI(cw)
		sources = append(sources, si)
	}

	newSrc := r.FormValue("new-src")
	if newSrc != "" {
		si := SourceInfo{
			Source: newSrc,
		}
		si.UpdateUI(cw)
		sources = append(sources, si)
	}

	uiInfo = calcUIInfo(sources)
	uiInfo.ColumnWidth = cw

	return
}

func filter(s string) string {
	s = strings.Replace(s, "org.apache.hadoop.hbase.regionserver.", "*.", -1)
	s = strings.Replace(s, "org.apache.hadoop.hbase.master.", "*.", -1)
	s = strings.Replace(s, "org.apache.hadoop.hbase.", "*.", -1)
	s = strings.Replace(s, "org.apache.hadoop.fs.", "*.", -1)
	s = strings.Replace(s, "org.apache.hadoop.", "*.", -1)
	s = strings.Replace(s, ".data.facebook.com", ".*", -1)
	s = strings.Replace(s, ".facebook.com", ".*", -1)
	s = strings.Replace(s, " INFO ", " I ", -1)
	s = strings.Replace(s, " DEBUG ", " D ", -1)
	return s
}

func mark(text string, qs []string) template.HTML {
	var res template.HTML
	for len(text) > 0 {
		minPos := len(text)
		minQ := 0
		for i, q := range qs {
			p := strings.Index(text, q)
			if p >= 0 {
				if p < minPos {
					minPos, minQ = p, i
				}
			}
		}
		if minPos == len(text) {
			res += template.HTML(template.HTMLEscapeString(filter(text)))
			text = ""
		} else {
			res += template.HTML(template.HTMLEscapeString(filter(text[:minPos])))
			res += template.HTML(fmt.Sprintf("<b class='s-%d'>", minQ+1) + template.HTMLEscapeString(qs[minQ]) + "</b>")
			text = text[minPos+len(qs[minQ]):]
		}
	}
	return res
}

func removeEmpty(qs []string) []string {
	var res []string
	for _, q := range qs {
		if q == "" {
			continue
		}
		res = append(res, q)
	}
	return res
}

func cmpTimeRange(tm, start, end string) int {
	if start != "" && tm < start {
		return -1
	}
	if end != "" && tm >= end {
		return 1
	}
	return 0
}

func grepResults(opt *GrepOpt, in io.Reader, qs []string) ([]LogRow, error) {
	log.Printf("grepResults: %v", qs)
	qsBytes := make([][][]byte, len(qs))
	for i, q := range qs {
		qsBytes[i] = [][]byte{[]byte(q)}
	}
	lastTime := ""
	var logRows []LogRow
	if err := grep(qsBytes, in,
		func(index int, lines [][]byte) error {
			ll, succ := parse(lines)
			if !succ {
				return nil
			}

			time := string(ll.Time)
			cmpTime := cmpTimeRange(time, opt.StartTime, opt.EndTime)
			if cmpTime < 0 {
				return nil
			}
			if cmpTime > 0 {
				return NEXT_FILE
			}

			if time != lastTime {
				lastTime = time
				logRows = append(logRows, LogRow{
					Time:  string(ll.Time),
					Parts: [][][]Cell{make([][]Cell, len(qs))},
				})
			}

			row := logRows[len(logRows)-1]
			for _, cnt := range ll.Contents {
				line := mark(string(cnt), qs)
				row.Parts[0][index] = append(row.Parts[0][index], Cell{
					Log: line,
				})
			}

			if len(logRows) >= MAX_LINES {
				return STOP
			}

			return nil
		}); err != nil && err != STOP && err != NEXT_FILE {
		return nil, err
	}
	log.Printf("Get %d rows for %v", len(logRows), qs)
	return logRows, nil
}

func getLocalResults(opt *GrepOpt, si SourceInfo) ([]LogRow, error) {
	log.Printf("getLocalResults: %v", si.Source)
	in, err := villa.Path(si.Source).Open()
	if err != nil {
		return nil, err
	}
	defer in.Close()

	return grepResults(opt, in, si.Queries)
}

func getJenkinsResults(opt *GrepOpt, si SourceInfo) ([]LogRow, error) {
	logs, err := loadFromJenkins(si.Source)
	if err != nil {
		return nil, err
	}
	in := villa.NewPByteSlice([]byte(logs))
	return grepResults(opt, in, si.Queries)
}

func getRemoteResults(opt *GrepOpt, si SourceInfo) ([]LogRow, error) {
	src := si.Source[len("http://"):]
	p := strings.Index(src, "/")
	if p < 0 {
		return nil, errors.New(fmt.Sprintf("Wrong source format: %v", si.Source))
	}

	log.Printf("Source: %s, src: %s, p: %v", si.Source, src, p)

	url := "http://" + src[:p+1] + "?fmt=json&src-cnt=1"
	url += "&src-0=" + template.URLQueryEscaper(src[p:])
	url += "&time-begin=" + template.URLQueryEscaper(opt.StartTime)
	url += "&time-end=" + template.URLQueryEscaper(opt.EndTime)
	for _, q := range si.Queries {
		url += "&q-0=" + template.URLQueryEscaper(q)
	}

	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var jsonData struct {
		Logs []LogRow
	}
	if err := json.Unmarshal(body, &jsonData); err != nil {
		return nil, err
	}

	return jsonData.Logs, nil
}

func getResultsOfSouce(opt *GrepOpt, si SourceInfo) ([]LogRow, error) {
	log.Printf("getResultsOfSouce: %v", si.Source)
	switch si.Type() {
	case "local":
		return getLocalResults(opt, si)
	case "jenkins":
		return getJenkinsResults(opt, si)
	case "remote":
		return getRemoteResults(opt, si)
	default:
		panic("Unknown source type: " + si.Type())
	}
}

func mergeResults(sources []SourceInfo, results [][]LogRow) (merged []LogRow) {
	if len(results) == 0 {
		return nil
	} else if len(results) == 1 {
		return results[0]
	}

	indexes := make([]int, len(results))

	pq := villa.NewIntPriorityQueueCap(
		func(a, b int) int {
			ta := results[a][indexes[a]].Time
			tb := results[b][indexes[b]].Time
			return villa.StrValueCompare(ta, tb)
		}, len(sources))

	for i, result := range results {
		if len(result) > 0 {
			pq.Push(i)
		}
	}

	lastTime := ""
	for pq.Len() > 0 {
		// Popup the smallest one
		src := pq.Pop()

		// merge in
		lr := &results[src][indexes[src]]

		if lr.Time != lastTime {
			lastTime = lr.Time
			newLR := LogRow{
				Time:  string(lr.Time),
				Parts: make([][][]Cell, len(sources)),
			}
			for i, source := range sources {
				newLR.Parts[i] = make([][]Cell, len(source.Queries))
			}
			merged = append(merged, newLR)
		}

		mlr := &merged[len(merged)-1]
		mlr.Parts[src] = lr.Parts[0]

		// Push back if not empty
		indexes[src]++
		if indexes[src] < len(results[src]) {
			pq.Push(src)
		}
	}

	return merged
}

type GrepOpt struct {
	StartTime string
	EndTime   string
}

func getResultsOfSources(opt *GrepOpt, sources []SourceInfo) []LogRow {
	log.Printf("%d sources", len(sources))
	outs := make([]chan []LogRow, len(sources))
	for i := range outs {
		outs[i] = make(chan []LogRow, 1)
	}
	for i, source := range sources {
		go func(si SourceInfo, out chan []LogRow) {
			logRows, err := getResultsOfSouce(opt, si)
			if err != nil {
				cells := make([][]Cell, len(si.Queries))
				cells[0] = []Cell{Cell{
					Log: template.HTML(`<span class="errmsg">` + template.HTMLEscapeString(err.Error()) + `<span>`),
				}}
				out <- []LogRow{LogRow{
					Time:  " ERROR",
					Parts: [][][]Cell{cells},
				}}
				return
			}

			out <- logRows
		}(source, outs[i])
	}

	log.Printf("Collecting results from %d sources: ", len(sources))

	results := make([][]LogRow, len(sources))
	for i := range results {
		results[i] = <-outs[i]
	}

	log.Printf("Merging results from %d sources: ", len(results))

	return mergeResults(sources, results)
}

func pageRoot(w http.ResponseWriter, r *http.Request) {
	loadTemplatesFromDisk()

	if r.URL.Path != "/" {
		w.WriteHeader(http.StatusNotFound)
		if err := templates.ExecuteTemplate(w, "404.html", nil); err != nil {
			w.Write([]byte(err.Error()))
		}
		return
	}

	r.ParseForm()
	format := r.FormValue("fmt")

	sources, uiInfo := parseInput(r)
	opt := GrepOpt{
		StartTime: r.FormValue("time-begin"),
		EndTime:   r.FormValue("time-end"),
	}
	logRows := getResultsOfSources(&opt, sources)

	if format == "json" {
		if jsonStr, err := json.Marshal(struct {
			Logs []LogRow
		}{
			Logs: logRows,
		}); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		} else if _, err := w.Write(jsonStr); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	} else {
		if err := templates.ExecuteTemplate(w, "index.html", struct {
			Logs       []LogRow
			TotalLines int
			Sources    []SourceInfo
			UIInfo
			GrepOpt
		}{
			Logs:       logRows,
			TotalLines: gTotalLines,
			Sources:    sources,
			UIInfo:     uiInfo,
			GrepOpt:    opt,
		}); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
}

func pageSource(w http.ResponseWriter, r *http.Request) {
	info := ""
	if localFn := r.FormValue("localfn"); localFn != "" {
		gLogFile = villa.Path(localFn)
		gJenkins = nil
		gSource = localFn
		gTotalLines = -1
		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
	} else if jenkinsURL := r.FormValue("jenkinsurl"); jenkinsURL != "" {
		if lines, err := loadFromJenkins(jenkinsURL); err != nil {
			info = fmt.Sprintf("Loading %s failed: %v", jenkinsURL, err)
		} else {
			gJenkins = []byte(lines)
			gLogFile = ""
			gSource = jenkinsURL
			gTotalLines = len(strings.Split(lines, "\n"))
			http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
		}
	}

	if err := templates.ExecuteTemplate(w, "jenkinsin.html", struct {
		Info string
	}{
		info,
	}); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}
