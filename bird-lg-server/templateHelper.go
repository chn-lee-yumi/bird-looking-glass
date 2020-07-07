package main

import (
	"net/http"
	"regexp"
	"sort"
	"strings"
	// "fmt"
)

func renderTemplate(w http.ResponseWriter, r *http.Request, content string) {
	path := r.URL.Path[1:]
	split := strings.SplitN(path, "/", 3)

	isWhois := strings.ToLower(split[0]) == "whois"
	whoisTarget := strings.Join(split[1:], "/")

	// Use a default URL if the request URL is too short
	if len(split) < 2 {
		path = "summary/" + strings.Join(setting.servers, "+") + "/"
	} else if len(split) == 2 {
		path += "/"
	}

	split = strings.SplitN(path, "/", 3)

	var args tmplArguments
	args.Options = map[string]string{
		"summary":            "show protocols",
		"detail":             "show protocols all ...",
		"route":              "show route for ...",
		"route_all":          "show route for ... all",
		"route_bgpmap":       "show route for ... (bgpmap)",
		"route_where":        "show route where net ~ [ ... ]",
		"route_where_all":    "show route where net ~ [ ... ] all",
		"route_where_bgpmap": "show route where net ~ [ ... ] (bgpmap)",
		// "route_generic":      "show route ...",
		// "generic":            "show ...",
		"whois":      "whois ...",
		"traceroute": "traceroute ...",
		"ping":       "ping -R ...",
	}
	args.Servers = setting.servers
	args.AllServersLinkActive = strings.ToLower(split[1]) == strings.ToLower(strings.Join(setting.servers, "+"))
	args.AllServersURL = strings.Join(setting.servers, "+")
	args.IsWhois = isWhois
	args.WhoisTarget = whoisTarget

	args.URLOption = strings.ToLower(split[0])
	args.URLServer = strings.ToLower(split[1])
	args.URLCommand = split[2]

	args.Content = content

	err := tmpl.Execute(w, args)
	if err != nil {
		panic(err)
	}
}

// Write the given text to http response, and add whois links for
// ASNs and IP addresses
func smartFormatter(s string) string {
	var result string
	result += "<pre>"
	for _, line := range strings.Split(s, "\n") {
		var lineFormatted string
		if strings.HasPrefix(strings.TrimSpace(line), "BGP.as_path:") || strings.HasPrefix(strings.TrimSpace(line), "Neighbor AS:") || strings.HasPrefix(strings.TrimSpace(line), "Local AS:") {
			lineFormatted = regexp.MustCompile(`(\d+)`).ReplaceAllString(line, `<a href="/whois/AS${1}" class="whois">${1}</a>`)
		} else {
			lineFormatted = regexp.MustCompile(`([a-zA-Z0-9\-]*\.([a-zA-Z]{2,3}){1,2})(\s|$)`).ReplaceAllString(line, `<a href="/whois/${1}" class="whois">${1}</a>${3}`)
			lineFormatted = regexp.MustCompile(`\[AS(\d+)`).ReplaceAllString(lineFormatted, `[<a href="/whois/AS${1}" class="whois">AS${1}</a>`)
			lineFormatted = regexp.MustCompile(`(\d+\.\d+\.\d+\.\d+)`).ReplaceAllString(lineFormatted, `<a href="/whois/${1}" class="whois">${1}</a>`)
			lineFormatted = regexp.MustCompile(`(?i)(([a-f\d]{0,4}:){3,10}[a-f\d]{0,4})`).ReplaceAllString(lineFormatted, `<a href="/whois/${1}" class="whois">${1}</a>`)
		}
		result += lineFormatted + "\n"
	}
	result += "</pre>"
	return result
}

type summaryTableArguments struct {
	Headers []string
	Lines   [][]string
}

// Output a table for the summary page
func summaryTable(data string, serverName string) string {
	var result string

	// Sort the table, excluding title row
	stringsSplitted := strings.Split(strings.TrimSpace(data), "\n")
	if len(stringsSplitted) <= 1 {
		// Likely backend returned an error message
		result = "<pre>" + strings.TrimSpace(data) + "</pre>"
	} else {
		// Draw the table head
		result += "<table class=\"table table-hover table-sm\">"
		result += "<thead>"
		for _, col := range strings.Split(stringsSplitted[0], " ") {
			colTrimmed := strings.TrimSpace(col)
			if len(colTrimmed) == 0 {
				continue
			}
			result += "<th scope=\"col\">" + colTrimmed + "</th>"
		}
		result += "</thead><tbody>"

		stringsWithoutTitle := stringsSplitted[1:]
		sort.Strings(stringsWithoutTitle)

		for _, line := range stringsWithoutTitle {
			// Ignore empty line
			line = strings.TrimSpace(line)
			if len(line) == 0 {
				continue
			}

			// Parse a total of 6 columns from bird summary
			lineSplitted := regexp.MustCompile(`(\w+)(\s+)(\w+)(\s+)([\w-]+)(\s+)(\w+)(\s+)([0-9\- :\.]+)(.*)`).FindStringSubmatch(line)
			var row = [6]string{
				strings.TrimSpace(lineSplitted[1]),
				strings.TrimSpace(lineSplitted[3]),
				strings.TrimSpace(lineSplitted[5]),
				strings.TrimSpace(lineSplitted[7]),
				strings.TrimSpace(lineSplitted[9]),
				strings.TrimSpace(lineSplitted[10]),
			}

			// Set table color
			result += "<tr class=\"" + (map[string]string{
				"down":  "table-secondary",
				"start": "table-warning",
			})[row[3]] + "\">"

			result += "<td><a href=\"/detail/" + serverName + "/" + row[0] + "\">" + row[0] + "</a></td>"

			// Draw the other cells
			for i := 1; i < 6; i++ {
				result += "<td>" + row[i] + "</td>"
			}
			result += "</tr>"
		}
		result += "</tbody></table>"
		result += "<!==" + data + "-->"
	}

	return result
}
