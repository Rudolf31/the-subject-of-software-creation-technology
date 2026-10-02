package handlers

import (
	"net/http"
	"time"

	"github.com/Rudolf31/myapp/utils"
)

type pingResp struct {
	Status string `json:"status"`
	Time   string `json:"time"`
}

func Ping(w http.ResponseWriter, r *http.Request) {
	utils.LogRequest(r)
	utils.WriteJSON(w, http.StatusOK, pingResp{
		Status: "ok",
		Time:   time.Now().UTC().Format(time.RFC3339),
	})
}
