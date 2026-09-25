package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"gioui.org/app"
	"gioui.org/io/system"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
)

func main() {
	go func() {
		w := new(app.Window)
		w.Option(app.Title("Resing"), app.Size(unit.Dp(400), unit.Dp(400)))
		w.Perform(system.ActionCenter)
		if err := run(w); err != nil {
			log.Fatal(err)
		}
		os.Exit(0)
	}()
	app.Main()
}

func run(w *app.Window) error {
	settings, err := getSettings()
	if err != nil {
		return runError(w, fmt.Sprintf("ошибка файла настроек: %s", err))
	}
	return runMain(w, settings)
}

func runMain(w *app.Window, settings *settings) error {

	th := material.NewTheme()

	var ops op.Ops

	var input widget.Editor
	input.SingleLine = true
	input.Submit = true

	var button widget.Clickable
	var buttonEnabled bool

	var logs string
	var busy bool

	var logsC <-chan string

	for {
		switch e := w.Event().(type) {
		case app.DestroyEvent:
			return e.Err
		case app.FrameEvent:
			gtx := app.NewContext(&ops, e)

			var inputClicked bool
			for {
				ev, ok := input.Update(gtx)
				if !ok {
					break
				}
				if _, ok := ev.(widget.SubmitEvent); ok {
					inputClicked = true
				}
			}
			text := strings.TrimSpace(input.Text())
			buttonEnabled = !busy && text != ""

			if !busy && (button.Clicked(gtx) || inputClicked) {
				logs = ""
				logsC = handle(text, settings)
				busy = true
			}

			for run := busy; run; {
				select {
				case log, ok := <-logsC:
					if ok {
						logs += "\n" + log
					} else {
						logsC = nil
						busy = false
						run = false
					}
				default:
					run = false
				}
			}

			layout.Flex{Axis: layout.Vertical}.Layout(gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
						layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
							return layout.UniformInset(unit.Dp(8)).Layout(gtx,
								material.Editor(th, &input, "Введите VLESS URI или URL").Layout)
						}),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							if !buttonEnabled {
								gtx = gtx.Disabled()
							}
							return layout.UniformInset(unit.Dp(8)).Layout(gtx,
								material.Button(th, &button, "Go").Layout)
						}),
					)
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return layout.UniformInset(unit.Dp(8)).Layout(gtx,
						material.Body1(th, logs).Layout)
				}),
			)

			e.Frame(gtx.Ops)
		}
	}
}

func runError(w *app.Window, errStr string) error {
	th := material.NewTheme()
	var ops op.Ops
	for {
		switch e := w.Event().(type) {
		case app.DestroyEvent:
			return e.Err
		case app.FrameEvent:
			gtx := app.NewContext(&ops, e)
			layout.Flex{Axis: layout.Vertical}.Layout(gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return layout.UniformInset(unit.Dp(8)).Layout(gtx,
						material.Body1(th, errStr).Layout)
				}),
			)

			e.Frame(gtx.Ops)
		}
	}
}

type settings struct {
	Path    string
	Service string
}

func getSettings() (*settings, error) {
	var settings settings
	f, err := os.Open("settings.json")
	if err != nil {
		return nil, fmt.Errorf("can't open settings.json: %w", err)
	}
	defer f.Close()
	if err := json.NewDecoder(f).Decode(&settings); err != nil {
		return nil, fmt.Errorf("can't decode settings.json: %w", err)
	}
	if settings.Path == "" {
		return nil, fmt.Errorf("Path is empty")
	}
	if settings.Service == "" {
		return nil, fmt.Errorf("Service is empty")
	}
	return &settings, nil
}

func handle(s string, settings *settings) <-chan string {
	c := make(chan string, 20)
	go func() {
		c <- "Запуск"

		idx := strings.Index(s, "://")
		if idx == -1 {
			handleURL(s, c, settings)
		} else {
			switch s[:idx] {
			case "http", "https":
				handleURL(s, c, settings)
			case "vless":
				handleVLESS(s, c, settings)
			default:
				c <- "Ошибка: неподдерживаемый протокол"
			}
		}

		close(c)
	}()
	return c
}

func handleURL(url string, c chan<- string, settings *settings) {
	c <- "Загрузка"

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		c <- fmt.Sprintf("Oшибка создания запроса: %v", err)
		return
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		c <- fmt.Sprintf("Oшибка выполнения запроса: %v", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 && resp.StatusCode >= 300 {
		c <- fmt.Sprintf("Ошибка: получен ответ с кодом %s", resp.Status)
		return
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		c <- fmt.Sprintf("Oшибка чтения ответа: %v", err)
		return
	}
	if len(body) == 0 {
		c <- "Ошибка: пустой ответ"
		return
	}

	c <- "Загрузка завершена"

	handleVLESS(string(body), c, settings)
}

func handleVLESS(uri string, c chan<- string, settings *settings) {
	c <- "Обработка VLESS URI"
	parsed, err := url.Parse(uri)
	if err != nil {
		c <- fmt.Sprintf("Oшибка парсинга VLESS URI: %v", err)
		return
	}
	values, err := url.ParseQuery(parsed.RawQuery)
	if err != nil {
		c <- fmt.Sprintf("Oшибка парсинга VLESS URI query: %v", err)
		return
	}

	params := make(map[string]string)
	params["scheme"] = parsed.Scheme
	params["user"] = parsed.User.Username()
	params["host"] = parsed.Hostname()
	params["port"] = parsed.Port()
	params["path"] = first(values["path"])
	params["type"] = first(values["type"])
	params["security"] = first(values["security"])
	params["fp"] = first(values["fp"])
	params["sni"] = first(values["sni"])
	params["pbk"] = first(values["pbk"])

	for k, v := range params {
		if strings.Contains(v, "\"") {
			c <- fmt.Sprintf("Oшибка: параметр %s содержит кавычки", k)
			return
		}
	}

	c <- "Генерация конфига"
	var text string
	switch params["type"] {
	case "tcp":
		text, err = tcpTemplate(params)
		if err != nil {
			c <- fmt.Sprintf("Oшибка генерации tcp config: %v", err)
			return
		}
	case "xhttp":
		text, err = xhttpTemplate(params)
		if err != nil {
			c <- fmt.Sprintf("Oшибка генерации http config: %v", err)
			return
		}

	default:
		c <- fmt.Sprintf("Oшибка: неподдерживаемый тип %s", params["type"])
		return
	}

	c <- "Запись конфига"
	err = os.WriteFile(settings.Path, []byte(text), 0644)
	if err != nil {
		c <- fmt.Sprintf("Ошибка записи конфига: %v", err)
		return
	}

	ok := restartService(settings.Service, c)
	if !ok {
		return
	}

	c <- "Готово"
}

func first(a []string) string {
	if len(a) > 0 {
		return a[0]
	}
	return ""
}
