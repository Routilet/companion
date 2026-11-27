package main

import (
	"bufio"
	"encoding/json"
	"os"
)

func main() {
	s := bufio.NewScanner(os.Stdin)
	w := json.NewEncoder(os.Stdout)
	var user string
	for s.Scan() {
		var r struct {
			ID     uint64
			Method string
			Params map[string]any
		}
		if json.Unmarshal(s.Bytes(), &r) != nil {
			return
		}
		resp := map[string]any{"id": r.ID}
		switch r.Method {
		case "capabilities":
			resp["result"] = map[string]any{"protocol": "routilet-oj-adapter/1", "adapter": "accoding", "version": "test", "methods": []string{"capabilities", "account", "login", "logout", "import"}}
		case "import":
			if user == "" {
				resp["error"] = map[string]string{"code": "auth", "message": "login required"}
				break
			}
			resp["result"] = map[string]any{"accountId": user, "title": "作业", "problems": []any{map[string]any{"title": "题", "label": "A", "target": map[string]string{"kind": "contest", "contestId": "42", "problemId": "5"}, "sourceUrl": "https://accoding.buaa.edu.cn/contest-ng/index.html#/42", "samples": []any{map[string]string{"input": "", "expected": ""}}, "warnings": []string{}, "special": false}}}
		case "login":
			user = ""
			if r.Params["password"] != "test-secret" {
				resp["error"] = map[string]string{"code": "auth", "message": "do not display"}
				break
			}
			user, _ = r.Params["username"].(string)
			fallthrough
		case "account", "logout":
			if r.Method == "logout" {
				user = ""
			}
			if user == "99" && r.Params["refresh"] == true {
				resp["error"] = map[string]string{"code": "network", "message": "private"}
				break
			}
			if user == "88" && r.Params["refresh"] == true {
				user = ""
				resp["error"] = map[string]string{"code": "auth", "message": "expired"}
				break
			}
			if user == "" {
				resp["result"] = map[string]string{"state": "signed_out"}
			} else {
				resp["result"] = map[string]string{"state": "signed_in", "id": user, "nickname": "测试"}
			}
		default:
			return
		}
		if w.Encode(resp) != nil {
			return
		}
	}
}
