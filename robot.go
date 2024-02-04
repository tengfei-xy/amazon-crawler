package main

import (
	"fmt"
	"regexp"
	"strings"
)

type Robots struct {
	robot []Robot
}
type Robot struct {
	ua_name string
	ua_re   *regexp.Regexp
	ua      []uastruct
}
type uastruct struct {
	url    string
	url_re *regexp.Regexp
	allow  bool
}

func GetRobotFromTxt(txt string) Robots {
	ua := ""
	ua_name := ""
	ua_list := Robots{}
	for _, line := range strings.Split(txt, "\n") {
		if strings.HasPrefix(line, "User-agent: ") {
			ua = strings.TrimPrefix(line, "User-agent: ")
			ua_name = ua
			ua_list.robot = append(ua_list.robot, Robot{ua_name, regexp.MustCompile(strings.ReplaceAll(ua_name, "*", ".*")), []uastruct{}})
		} else if strings.HasPrefix(line, "Disallow: ") {
			u := strings.TrimPrefix(line, "Disallow: ")

			ua_list.robot[len(ua_list.robot)-1].ua = append(ua_list.robot[len(ua_list.robot)-1].ua, uastruct{u, regexp.MustCompile(strings.ReplaceAll(u, "*", ".*")), false})
		} else if strings.HasPrefix(line, "Allow: ") {
			u := strings.TrimPrefix(line, "Allow: ")
			ua_list.robot[len(ua_list.robot)-1].ua = append(ua_list.robot[len(ua_list.robot)-1].ua, uastruct{u, regexp.MustCompile(strings.ReplaceAll(u, "*", ".*")), true})
		}
	}
	return ua_list
}
func (r *Robots) IsAllow(ua string, url string) error {
	for _, robot := range r.robot {
		var allow bool
		var err error
		if robot.ua_re.MatchString(ua) {
			for _, ua := range robot.ua {

				if !ua.allow && ua.url_re.MatchString(url) {
					err = fmt.Errorf("由于robots.txt限制,不允许爬取(UA:%s URL:%s ),当前目标链接:%s", robot.ua_name, ua.url, url)
					allow = false
				}
			}
			for _, ua := range robot.ua {
				if ua.allow && ua.url_re.MatchString(url) {
					if !allow {
						return nil
					}
				}
			}
			if !allow {
				return err
			}
		}
	}
	return nil
}

func (r *Robots) Output() {
	for _, robot := range r.robot {
		fmt.Printf("User-agent: %s\n", robot.ua_name)
		for _, ua := range robot.ua {
			if ua.allow {
				fmt.Printf("Allow: %s\n", ua.url)
			} else {
				fmt.Printf("Disallow: %s\n", ua.url)
			}
		}
	}
}
