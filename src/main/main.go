package main

import (
	"gopkg.in/mgo.v2"
	"github.com/kataras/iris"
	"log"
	"time"
	"model"
	"strings"
	"github.com/googollee/go-socket.io"
	"fmt"
	"encoding/json"
	"io/ioutil"
)


func main(){

	session, err := mgo.Dial("127.0.0.1")
	if err != nil {
		panic(err)
	}
	defer session.Close()
	session.SetMode(mgo.Monotonic, true)

	dat, _ := ioutil.ReadFile("templates/script.js")
	CHART_SCRIPT := string(dat)

	iris.Config.IsDevelopment = true

	iris.OnError(iris.StatusForbidden, func(ctx *iris.Context) {
		ctx.HTML(iris.StatusForbidden, "<h1> You are not allowed here </h1>")
	})

	iris.Static("/assets", "./static/assets", 1)
	iris.Static("/images", "./static/images", 1)

	iris.Get("/", func(ctx *iris.Context){
		ctx.Render("index.html", nil)
	})

	iris.Get("/job/:unique", func(ctx *iris.Context){
		unique := ctx.Param("unique")
		j := model.Job{}.Find(session, unique)
		ctx.Render("job.html", map[string]interface{}{
			"Unique":unique,
			"Url":j.Url,
		})
	})

	iris.Get("/script/wrk-stats/:unique", func(ctx *iris.Context) {
		unique := ctx.Param("unique")
		chart := model.Chart{}.NewInstance(unique)

		chart.RetrieveRequestPerSec(session).
			RetrieveTransferPerSec(session).
			RetrieveLatency(session).
			RetrieveThread(session).
			RetrieveRequest(session).
			RetrieveTransfer(session).
			RetrieveSocketError(session)

		jsonrps, err := json.Marshal(chart.RequestPerSec)
		if err != nil{
			ctx.JSON(iris.StatusOK, err)
		}

		jsontps, err := json.Marshal(chart.TransferPerSec)
		if err != nil{
			ctx.JSON(iris.StatusOK, err)
		}

		jsonlm, err := json.Marshal(chart.LatencyMax)
		if err != nil{
			ctx.JSON(iris.StatusOK, err)
		}

		jsonla, err := json.Marshal(chart.LatencyAvg)
		if err != nil{
			ctx.JSON(iris.StatusOK, err)
		}

		jsonls, err := json.Marshal(chart.LatencyStd)
		if err != nil{
			ctx.JSON(iris.StatusOK, err)
		}

		jsontm, err := json.Marshal(chart.ThreadMax)
		if err != nil{
			ctx.JSON(iris.StatusOK, err)
		}

		jsonta, err := json.Marshal(chart.ThreadAvg)
		if err != nil{
			ctx.JSON(iris.StatusOK, err)
		}

		jsonts, err := json.Marshal(chart.ThreadStd)
		if err != nil{
			ctx.JSON(iris.StatusOK, err)
		}

		jsonr, err := json.Marshal(chart.Requests)
		if err != nil{
			ctx.JSON(iris.StatusOK, err)
		}

		jsontt, err := json.Marshal(chart.TotalTransfer)
		if err != nil{
			ctx.JSON(iris.StatusOK, err)
		}

		jsonec, err := json.Marshal(chart.SocketErrorsConnect)
		if err != nil{
			ctx.JSON(iris.StatusOK, err)
		}

		jsoner, err := json.Marshal(chart.SocketErrorsRead)
		if err != nil{
			ctx.JSON(iris.StatusOK, err)
		}

		jsonew, err := json.Marshal(chart.SocketErrorsWrite)
		if err != nil{
			ctx.JSON(iris.StatusOK, err)
		}

		jsonet, err := json.Marshal(chart.SocketErrorsTimeOut)
		if err != nil{
			ctx.JSON(iris.StatusOK, err)
		}

		jsonex, err := json.Marshal(chart.SocketErrorsNon2xx3xx)
		if err != nil{
			ctx.JSON(iris.StatusOK, err)
		}

		jsone, err := json.Marshal(chart.SocketErrorsTotal)
		if err != nil{
			ctx.JSON(iris.StatusOK, err)
		}

		s := CHART_SCRIPT
		s = strings.Replace(s, "{{.Unique}}", unique, -1)
		s = strings.Replace(s, "{{.rps}}", string(jsonrps), -1)
		s = strings.Replace(s, "{{.tps}}", string(jsontps), -1)
		s = strings.Replace(s, "{{.lm}}", string(jsonlm), -1)
		s = strings.Replace(s, "{{.la}}", string(jsonla), -1)
		s = strings.Replace(s, "{{.ls}}", string(jsonls), -1)
		s = strings.Replace(s, "{{.tm}}", string(jsontm), -1)
		s = strings.Replace(s, "{{.ta}}", string(jsonta), -1)
		s = strings.Replace(s, "{{.ts}}", string(jsonts), -1)
		s = strings.Replace(s, "{{.r}}", string(jsonr), -1)
		s = strings.Replace(s, "{{.tt}}", string(jsontt), -1)
		s = strings.Replace(s, "{{.ec}}", string(jsonec), -1)
		s = strings.Replace(s, "{{.er}}", string(jsoner), -1)
		s = strings.Replace(s, "{{.ew}}", string(jsonew), -1)
		s = strings.Replace(s, "{{.et}}", string(jsonet), -1)
		s = strings.Replace(s, "{{.ex}}", string(jsonex), -1)
		s = strings.Replace(s, "{{.e}}", string(jsone), -1)

		ctx.Text(iris.StatusOK, s)
	})

	iris.Get("/api/job", func(ctx *iris.Context){
		j := model.Job{}.GetAllJob(session)
		ctx.JSON(iris.StatusOK, j)
	})

	modelChan := make(chan *model.Job, 100)
	mongochan := make(chan model.WrkResult, 100)

	iris.Post("/wrk", func(ctx *iris.Context){
		url := string(ctx.FormValue("url"))
		ctx.Redirect("/")
		j := model.Job{}.NewInstance(url, session)
		modelChan <- j
	})

	server, err := socketio.NewServer(nil)
	if err != nil {
		log.Fatal(err)
	}

	server.On("connection", func (so socketio.Socket){
		so.Join("real-time")
		fmt.Println("connection in")
	})

	server.On("error", func(so socketio.Socket, err error) {
		log.Println("error:", err)
	})

	iris.Handle(iris.MethodGet, "/socket.io/", iris.ToHandler(server))
	iris.Handle(iris.MethodPost, "/socket.io/", iris.ToHandler(server))

	go func(){
		for;;{
			select {
			case j := <- modelChan:
				go func() {
					t := j.Unique
					time.Sleep(2 * time.Second)
					j.RunWrk("1", "10s", t, mongochan)
					server.BroadcastTo("real-time", t, `{"Unique":"` + t + `", "IsComplete":false, "Progress":15}`)
					time.Sleep(2 * time.Second)
					j.RunWrk("10", "10s", t, mongochan)
					server.BroadcastTo("real-time", t, `{"Unique":"` + t + `", "IsComplete":false, "Progress":30}`)
					time.Sleep(2 * time.Second)
					j.RunWrk("100", "10s", t, mongochan)
					server.BroadcastTo("real-time", t, `{"Unique":"` + t + `", "IsComplete":false, "Progress":45}`)
					time.Sleep(2 * time.Second)
					j.RunWrk("1k", "10s", t, mongochan)
					server.BroadcastTo("real-time", t, `{"Unique":"` + t + `", "IsComplete":false, "Progress":60}`)
					time.Sleep(2 * time.Second)
					j.RunWrk("10k", "10s", t, mongochan)
					server.BroadcastTo("real-time", t, `{"Unique":"` + t + `", "IsComplete":false, "Progress":75}`)
					j.RunWrk("100k", "10s", t, mongochan)
					server.BroadcastTo("real-time", t, `{"Unique":"` + t + `", "IsComplete":false, "Progress":90}`)
					j.Complete(session)
					server.BroadcastTo("real-time", t, `{"Unique":"` + t + `", "IsComplete":true, "Progress":100}`)
				}()
			case wrkResult := <- mongochan:
				go wrkResult.Save(session)
			}
		}
	}()

	iris.Listen(":8080")
}
