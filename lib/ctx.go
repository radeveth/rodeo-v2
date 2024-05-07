package lib

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ses"
)

// Ctx represents the context of an incomming HTTP request and it's response
type Ctx struct {
	Req     *http.Request
	Res     http.ResponseWriter
	Tpl     *template.Template
	DB      *Database
	Cache   *Cache
	Storage *Storage
	Queue   *JobQueue
	Server  *Server

	Code   int
	Data   J
	params url.Values

	// Tracing
	tracingSpanID   string
	tracingTraceID  string
	tracingRootTags J
}

func NewCtx(server *Server) *Ctx {
	ctx := &Ctx{Data: J{}, params: url.Values{}}
	ctx.Tpl = server.Tpl
	ctx.DB = server.Database.WithCtx(ctx)
	ctx.Cache = server.Cache.WithCtx(ctx)
	ctx.Storage = server.Storage.WithCtx(ctx)
	ctx.Queue = server.Queue.WithCtx(ctx)
	ctx.Server = server

	ctx.tracingSpanID = NewID()
	ctx.tracingTraceID = NewID()
	ctx.tracingRootTags = J{}

	return ctx
}

// Params returns a map of all form and query params
func (c *Ctx) Params() map[string]string {
	c.Req.ParseForm()
	params := map[string]string{}
	for key := range c.Req.Form {
		params[key] = c.Req.Form.Get(key)
	}
	for key := range c.params {
		params[key] = c.params.Get(key)
	}
	return params
}

// Param returnds either a form value, a query param or a provided alternative
// string value for a given parameter name.
func (c *Ctx) Param(name, alt string) string {
	if value := c.Req.FormValue(name); value != "" {
		return value
	}
	if value := c.params.Get(name); value != "" {
		return value
	}
	return alt
}

func (c *Ctx) ParamFloat(name string, alt float64) float64 {
	if value := c.Req.FormValue(name); value != "" {
		return StringToFloat(value)
	}
	if value := c.params.Get(name); value != "" {
		return StringToFloat(value)
	}
	return alt
}

// Bind parses the request body as JSON into the provided struct/value
func (c *Ctx) Bind(data interface{}) {
	defer c.Req.Body.Close()
	Check(json.NewDecoder(c.Req.Body).Decode(data))
}

// BindJ parses the request body as JSON into a J (map[string]interface{}) value
func (c *Ctx) BindJ() J {
	body := J{}
	defer c.Req.Body.Close()
	Check(json.NewDecoder(c.Req.Body).Decode(&body))
	return body
}

// GetCookie returns the value of a cookie
func (c *Ctx) GetCookie(name string) string {
	if cookie, err := c.Req.Cookie(name); err == nil {
		return cookie.Value
	}
	return ""
}

// SetCookie sets a cookie's value (http only, so it can't be accessed by JavaScript)
// (with an expiration date far far out in the future)
func (c *Ctx) SetCookie(name, value string) {
	http.SetCookie(c.Res, &http.Cookie{
		Name:     name,
		Value:    value,
		HttpOnly: true,
		Path:     "/",
		MaxAge:   2147483647,
	})
}

// Redirect sends a redirect response to the client
func (c *Ctx) Redirect(url string, args ...interface{}) {
	if len(args) > 0 {
		url = fmt.Sprintf(url, args...)
	}
	c.Code = 302
	c.Res.Header().Set("Location", url)
	c.Res.WriteHeader(302)
	c.Res.Write([]byte("Redirecting..."))
}

// Render renders an HTML template and sends it to the client
func (c *Ctx) Render(code int, template string, data J) {
	for k, v := range c.Data {
		if _, ok := data[k]; !ok {
			data[k] = v
		}
	}
	b := bytes.NewBuffer(nil)
	Check(c.Tpl.ExecuteTemplate(b, template, data))
	c.Code = code
	c.Res.Header().Set("Content-Type", "text/html; charset=utf-8")
	c.Res.WriteHeader(code)
	c.Res.Write(b.Bytes())
}

// Text sends a text response to the client
func (c *Ctx) Text(code int, text string) {
	c.Code = code
	c.Res.Header().Set("Content-Type", "text/plain; charset=utf-8")
	c.Res.WriteHeader(code)
	c.Res.Write([]byte(text))
}

// JSON sends a JSON encoded response to the client
func (c *Ctx) JSON(code int, data interface{}) {
	bs, err := json.Marshal(data)
	Check(err)
	c.Code = code
	c.Res.Header().Set("Content-Type", "application/json; charset=utf-8")
	c.Res.WriteHeader(code)
	c.Res.Write(bs)
}

// SendEmail renders a template to HTML and sends it as an email using AWS SES
// It uses the SES_ACCESS_KEY and SES_SECRET_KEY environment variables.
// And send the email from the email in the EMAIL_FROM environment variable.
func (c *Ctx) SendEmail(to, subject, paragraphs, buttonText, buttonLink string) {
	data := J{
		"to":             to,
		"paragraphs":     strings.Split(paragraphs, "\n\n"),
		"buttonText":     buttonText,
		"buttonLink":     buttonLink,
		"companyName":    Env("COMPANY_NAME", "App"),
		"unsubscribeUrl": Env("BASE_URL", "http://localhost:"+Env("PORT", "8000")) + "/unsubscribe?email=" + to,
	}
	html := bytes.NewBuffer(nil)
	Check(c.Tpl.ExecuteTemplate(html, "other/mail", data))
	text := bytes.NewBuffer(nil)
	text.Write([]byte(paragraphs))
	if buttonText != "" {
		text.Write([]byte("\n\n" + buttonText + " " + buttonLink))
	}

	_, err := ses.New(AWSSession("SES")).SendEmail(&ses.SendEmailInput{
		Source: aws.String(Env("EMAIL_FROM", "")),
		Destination: &ses.Destination{
			CcAddresses: []*string{},
			ToAddresses: []*string{aws.String(to)},
		},
		Message: &ses.Message{
			Subject: &ses.Content{
				Charset: aws.String("UTF-8"),
				Data:    aws.String(subject),
			},
			Body: &ses.Body{
				Html: &ses.Content{
					Charset: aws.String("UTF-8"),
					Data:    aws.String(html.String()),
				},
				Text: &ses.Content{
					Charset: aws.String("UTF-8"),
					Data:    aws.String(text.String()),
				},
			},
		},
	})
	Check(err)
}

type HTTPRequestOptions struct {
	Method  string
	URL     string
	Headers J
	Body    interface{}
}

func (c *Ctx) HTTPRequest(result interface{}, options HTTPRequestOptions) {
	Check(c.HTTPRequestErr(result, options))
}
func (c *Ctx) HTTPRequestErr(result interface{}, options HTTPRequestOptions) error {
	if options.Method == "" {
		options.Method = "GET"
	}
	var reqBody io.Reader
	if options.Body != nil {
		bs, err := json.Marshal(options.Body)
		if err != nil {
			return err
		}
		reqBody = bytes.NewBuffer(bs)
	}
	req, err := http.NewRequest(options.Method, options.URL, reqBody)
	if err != nil {
		return fmt.Errorf(`HTTPRequest: %v "%s"`, err, options.URL)
	}
	req.Header.Set("Accept", "application/json")
	if options.Method != "POST" {
		req.Header.Set("Content-Type", "application/json")
	}
	if options.Headers != nil {
		for k, v := range options.Headers {
			req.Header.Set(k, fmt.Sprintf("%v", v))
		}
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf(`HTTPRequest: %v "%s"`, err, options.URL)
	}
	defer res.Body.Close()
	resBody, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return fmt.Errorf(`HTTPRequest: %v "%s"`, err, options.URL)
	}
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return fmt.Errorf(`HTTPRequest: error status code %v "%s" (%s)`,
			res.StatusCode, options.URL, string(resBody))
	}
	return json.Unmarshal(resBody, result)
}
