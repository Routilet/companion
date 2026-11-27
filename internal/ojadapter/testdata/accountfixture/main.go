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
	var target, hash any
	var ready bool
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
			resp["result"] = map[string]any{"protocol": "routilet-oj-adapter/1", "adapter": "accoding", "version": "test", "methods": []string{"capabilities", "account", "login", "logout", "import", "prepare", "submit", "reconcile", "result", "status"}}
		case "result", "status":
			id := "124"
			result := "WA"
			if r.Method == "result" {
				id, _ = r.Params["id"].(string)
				result = "AC"
			}
			resp["result"] = map[string]any{"accountId": user, "target": r.Params["target"], "accepted": true, "latest": map[string]string{"id": id, "result": result, "language": "C99"}}
		case "reconcile":
			resp["result"] = map[string]string{"state": "unresolved"}
		case "prepare":
			if user == "" {
				resp["error"] = map[string]string{"code": "auth", "message": "login"}
				break
			}
			resp["result"] = map[string]any{"token": "0123456789abcdef0123456789abcdef", "target": r.Params["target"], "sourceHash": r.Params["sourceHash"], "accountId": user, "languages": []string{"C99"}}
			target = r.Params["target"]
			hash = r.Params["sourceHash"]
			ready = true
		case "submit":
			if !ready {
				resp["error"] = map[string]string{"code": "confirmation_expired", "message": "expired"}
				break
			}
			ready = false
			if user == "66" {
				return
			}
			resp["result"] = map[string]any{"id": "123", "target": target, "accountId": user, "language": r.Params["language"], "sourceHash": hash, "result": "WT"}
		case "import":
			if user == "" {
				resp["error"] = map[string]string{"code": "auth", "message": "login required"}
				break
			}
			resp["result"] = map[string]any{"accountId": user, "title": "作业", "problems": []any{map[string]any{"title": "题", "label": "A", "target": map[string]string{"kind": "contest", "contestId": "42", "problemId": "5"}, "sourceUrl": "https://accoding.buaa.edu.cn/contest-ng/index.html#/42", "samples": []any{map[string]string{"input": "", "expected": ""}}, "warnings": []string{}, "special": false}}}
		case "login":
			ready = false
			user = ""
			if r.Params["password"] != "test-secret" {
				resp["error"] = map[string]string{"code": "auth", "message": "do not display"}
				break
			}
			user, _ = r.Params["username"].(string)
			fallthrough
		case "account", "logout":
			if r.Method == "logout" {
				ready = false
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
