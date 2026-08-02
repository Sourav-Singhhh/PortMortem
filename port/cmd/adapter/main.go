package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"

	"github.com/Sourav-Singhhh/PortMortem/port"
)

type Request struct {
	ID      string                  `json:"id"`
	Action  string                  `json:"action"` // "isMatch", "compile", "scan", "parse"
	Pattern string                  `json:"pattern"`
	Input   string                  `json:"input"`
	Opts    *picomatch.ParseOptions `json:"opts,omitempty"`
}

type Response struct {
	ID      string      `json:"id"`
	Success bool        `json:"success"`
	Result  interface{} `json:"result,omitempty"`
	Error   string      `json:"error,omitempty"`
}

type ScanResultDTO struct {
	Prefix     string   `json:"prefix"`
	Input      string   `json:"input"`
	Start      int      `json:"start"`
	Base       string   `json:"base"`
	Glob       string   `json:"glob"`
	IsNegated  bool     `json:"isNegated"`
	IsGlobstar bool     `json:"isGlobstar"`
	IsExtglob  bool     `json:"isExtglob"`
	IsBrace    bool     `json:"isBrace"`
	IsBracket  bool     `json:"isBracket"`
	IsGlob     bool     `json:"isGlob"`
	Negated    bool     `json:"negated"`
	Parts      []string `json:"parts,omitempty"`
}

func main() {
	// Single-shot CLI mode via command line arguments
	if len(os.Args) >= 3 {
		action := os.Args[1]
		pattern := os.Args[2]
		input := ""
		if len(os.Args) >= 4 {
			input = os.Args[3]
		}
		optsJSON := ""
		if len(os.Args) >= 5 {
			optsJSON = os.Args[4]
		}

		var opts *picomatch.ParseOptions
		if optsJSON != "" && optsJSON != "{}" {
			_ = json.Unmarshal([]byte(optsJSON), &opts)
		}

		switch action {
		case "isMatch":
			m, err := picomatch.Compile(pattern, opts)
			if err != nil {
				fmt.Print("false")
				os.Exit(1)
			}
			if m.Match(input) {
				fmt.Print("true")
				os.Exit(0)
			} else {
				fmt.Print("false")
				os.Exit(1)
			}
		case "scan":
			var scanOpts *picomatch.ScanOptions
			if opts != nil {
				scanOpts = &picomatch.ScanOptions{
					NoExt:    opts.NoExtglob,
					NoNegate: opts.NoNegate,
				}
			}
			res := picomatch.Scan(pattern, scanOpts)
			dto := ScanResultDTO{
				Prefix:     res.Prefix,
				Input:      res.Input,
				Start:      res.Start,
				Base:       res.Base,
				Glob:       res.Glob,
				IsNegated:  res.Negated,
				IsGlobstar: res.IsGlobstar,
				IsExtglob:  res.IsExtglob,
				IsBrace:    res.IsBrace,
				IsBracket:  res.IsBracket,
				IsGlob:     res.IsGlob,
				Negated:    res.Negated,
				Parts:      res.Parts,
			}
			b, _ := json.Marshal(dto)
			fmt.Print(string(b))
			os.Exit(0)
		case "parse":
			st, err := picomatch.Parse(pattern, opts)
			if err != nil {
				os.Exit(1)
			}
			resMap := map[string]interface{}{
				"input":      st.Input,
				"output":     st.Output,
				"state":      st,
				"isGlob":     st.Globstar || st.Braces > 0 || st.Brackets > 0 || st.Parens > 0,
				"isBrace":    st.Braces > 0,
				"isBracket":  st.Brackets > 0,
				"isGlobstar": st.Globstar,
			}
			b, _ := json.Marshal(resMap)
			fmt.Print(string(b))
			os.Exit(0)
		}
	}

	// Persistent ND-JSON stream mode over stdin/stdout
	scanner := bufio.NewScanner(os.Stdin)
	buf := make([]byte, 0, 1024*1024)
	scanner.Buffer(buf, 10*1024*1024)

	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var req Request
		if err := json.Unmarshal(line, &req); err != nil {
			sendResp(Response{Error: err.Error()})
			continue
		}

		resp := Response{ID: req.ID, Success: true}

		switch req.Action {
		case "isMatch":
			m, err := picomatch.Compile(req.Pattern, req.Opts)
			if err != nil {
				resp.Success = false
				resp.Error = err.Error()
			} else {
				resp.Result = m.Match(req.Input)
			}

		case "scan":
			var scanOpts *picomatch.ScanOptions
			if req.Opts != nil {
				scanOpts = &picomatch.ScanOptions{
					NoExt:    req.Opts.NoExtglob,
					NoNegate: req.Opts.NoNegate,
				}
			}
			res := picomatch.Scan(req.Pattern, scanOpts)
			dto := ScanResultDTO{
				Prefix:     res.Prefix,
				Input:      res.Input,
				Start:      res.Start,
				Base:       res.Base,
				Glob:       res.Glob,
				IsNegated:  res.Negated,
				IsGlobstar: res.IsGlobstar,
				IsExtglob:  res.IsExtglob,
				IsBrace:    res.IsBrace,
				IsBracket:  res.IsBracket,
				IsGlob:     res.IsGlob,
				Negated:    res.Negated,
				Parts:      res.Parts,
			}
			resp.Result = dto

		case "parse":
			st, err := picomatch.Parse(req.Pattern, req.Opts)
			if err != nil {
				resp.Success = false
				resp.Error = err.Error()
			} else {
				resp.Result = map[string]interface{}{
					"input":      st.Input,
					"output":     st.Output,
					"state":      st,
					"isGlob":     st.Globstar || st.Braces > 0 || st.Brackets > 0 || st.Parens > 0,
					"isBrace":    st.Braces > 0,
					"isBracket":  st.Brackets > 0,
					"isGlobstar": st.Globstar,
				}
			}

		default:
			resp.Success = false
			resp.Error = fmt.Sprintf("unknown action: %s", req.Action)
		}

		sendResp(resp)
	}
}

func sendResp(resp Response) {
	b, _ := json.Marshal(resp)
	fmt.Println(string(b))
}
