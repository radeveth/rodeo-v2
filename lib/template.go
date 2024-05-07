package lib

import (
	"crypto/md5"
	"embed"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"html/template"
	"io/ioutil"
	"math"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"golang.org/x/text/language"
	"golang.org/x/text/message"
)

var templateFunctions = template.FuncMap{
	"title":     strings.Title,
	"snakeCase": StringToSnakeCase,
	"slug":      StringToSlug,
	"hasPrefix": func(s string, prefix string) bool {
		return strings.HasPrefix(s, prefix)
	},
	"truncate": func(s string, length int) string {
		if len(s) > length {
			return s[0:length] + "…"
		} else {
			return s
		}
	},
	"fromJson": func(j interface{}) interface{} {
		var s string
		if v, ok := j.(string); ok {
			s = v
		} else {
			bs, err := json.Marshal(j)
			Check(err)
			s = string(bs)
		}
		var i interface{}
		err := json.Unmarshal([]byte(s), &i)
		Check(err)
		return i
	},
	"json": func(i interface{}) string {
		bs, err := json.MarshalIndent(i, "", "  ")
		Check(err)
		return string(bs)
	},
	"jsonNoIndent": func(i interface{}) string {
		bs, err := json.Marshal(i)
		Check(err)
		return string(bs)
	},
	"env": func(name string) string {
		return Env(name, "")
	},
	"now": func() time.Time {
		return time.Now()
	},
	"loc": func(timezone string) *time.Location {
		l, err := time.LoadLocation(timezone)
		Check(err)
		return l
	},
	"mod": func(i, m int) int {
		return i % m
	},
	"div": func(a, b int64) float64 {
		return float64(a) / float64(b)
	},
	"add": func(a, b int) int {
		return a + b
	},
	"add64": func(a, b int64) int64 {
		return a + b
	},
	"minus": func(a, b int) int {
		return a - b
	},
	"ago": func(t time.Time) string {
		d := time.Now().Sub(t)
		if d < 60*time.Minute {
			return fmt.Sprintf("%dm ago", d/time.Minute)
		} else if d < 24*time.Hour {
			return fmt.Sprintf("%dh ago", d/time.Hour)
		} else {
			return t.Format("2006-01-02")
		}
	},
	"markdown": func(text string) template.HTML {
		return template.HTML(MarkdownToString(text))
	},
	"stripHtml": func(s interface{}) string {
		ss := ""
		switch v := s.(type) {
		case string:
			ss = v
		case template.HTML:
			ss = string(v)
		}
		r := regexp.MustCompile(`<.*?>`).ReplaceAllString(ss, "")
		return strings.NewReplacer(">", "", "<", "").Replace(r)
	},
	"values": func(j map[string]interface{}) []interface{} {
		values := []interface{}{}
		for _, v := range j {
			values = append(values, v)
		}
		return values
	},
	"sortedValues": func(j map[string]interface{}, key string) []map[string]interface{} {
		values := []map[string]interface{}{}
		for _, v := range j {
			values = append(values, v.(map[string]interface{}))
		}
		sort.Slice(values, func(i, j int) bool {
			return IToString(values[i][key]) < IToString(values[j][key])
		})
		return values
	},
	"formatAddress": func(a string) string {
		return fmt.Sprintf("%s…%s", a[0:6], a[len(a)-4:])
	},
	"formatNumber": func(n *BigInt, s int64, d int64) string {
		p := message.NewPrinter(language.English)
		f, err := strconv.ParseFloat(n.String(), 64)
		Check(err)
		return p.Sprintf("%0."+strconv.FormatInt(d, 10)+"f", f/math.Pow(10.0, float64(s)))
	},
	"formatUnits": func(n *BigInt, s int64) string {
		f, err := strconv.ParseFloat(n.String(), 64)
		Check(err)
		return fmt.Sprintf("%f", f/math.Pow(10.0, float64(s)))
	},
	"formatSize": func(b int64) string {
		const unit = 1024
		if b < unit {
			return fmt.Sprintf("%d B", b)
		}
		div, exp := int64(unit), 0
		for n := b / unit; n >= unit; n /= unit {
			div *= unit
			exp++
		}
		return fmt.Sprintf("%.1f %ciB", float64(b)/float64(div), "KMGTPE"[exp])
	},
	"formatDBValue": func(i interface{}) template.HTML {
		if i == nil {
			return "NULL"
		}
		if v, ok := i.(time.Time); ok {
			return template.HTML(v.Format("2006-01-02 15:04"))
		}
		if v, ok := i.(float64); ok {
			return template.HTML(fmt.Sprintf("%.2f", v))
		}
		return template.HTML(fmt.Sprintf("%v", i))
	},
	"gravatar": func(email string) string {
		hasher := md5.New()
		hasher.Write([]byte(email))
		hash := hex.EncodeToString(hasher.Sum(nil))
		url := "https://www.gravatar.com/avatar/" + hash + ".jpg"
		url += "?s=120&r=pg&d=identicon"
		return url
	},
}

// NewTemplateFromFS builds a Template instance that contains all the templates from the provided file system sub-folders
func NewTemplateFromFS(fs embed.FS) *template.Template {
	t := template.New("").Funcs(templateFunctions)
	dirs, err := fs.ReadDir("views")
	Check(err)
	for _, dirInfo := range dirs {
		if !dirInfo.IsDir() {
			continue
		}
		files, err := fs.ReadDir("views/" + dirInfo.Name())
		Check(err)
		for _, fileInfo := range files {
			name := dirInfo.Name() + "/" + fileInfo.Name()
			file, err := fs.Open("views/" + name)
			Check(err)
			contents, err := ioutil.ReadAll(file)
			Check(err)
			t, err = t.New(strings.Replace(name, ".html", "", -1)).Parse(string(contents))
			Check(err)
		}
	}
	return t
}
