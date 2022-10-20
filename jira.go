package main

import (
	"encoding/xml"
	"errors"
	"flag"
	"fmt"
	"github.com/daichi-m/go18ds/sets/linkedhashset"
	"github.com/k3a/html2text"
	"github.com/tidwall/gjson"
	"io"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"
)

var topixFieldName = os.Getenv("J_TOPIX_FIELD_NAME")
var jobFieldName = os.Getenv("J_JOB_FIELD_NAME")
var jiraURL = os.Getenv("J_JIRA_URL")

// check here for the correct value
// jira_url/rest/activity-stream/1.0/config
var excludeConfluence = os.Getenv("J_EXCLUDE_CONFLUENCE")
var token = os.Getenv("J_JIRA_TOKEN")
var user = os.Getenv("J_DEFAULT_USER")

// invert time zone here
var timeZone = -7

func main() {
	//get input value from command line
	inUserName := flag.String("u", user, "user name")
	inDate := flag.String("d", time.Now().Format("02-01"), "date format DD-MM-YYYY or DD-MM - default is current date")
	flag.Parse()

	//prepare parameters
	user := fmt.Sprintf("streams=user+IS+%v", *inUserName)
	currentYear := time.Now().Year()
	if len(*inDate) < 10 {
		*inDate = fmt.Sprintf("%v-%v", *inDate, currentYear)
	}
	timeFrom, _ := time.Parse("02-01-2006", *inDate)
	timeFrom = timeFrom.Add(time.Hour * time.Duration(timeZone))
	timeTo := timeFrom.Add(time.Hour * 24)
	dateRange := fmt.Sprintf("streams=update-date+BETWEEN+%v+%v", timeFrom.UnixMilli(), timeTo.UnixMilli())
	maxResult := fmt.Sprintf("maxResults=%v", 100)
	url := fmt.Sprintf("%v/activity?%v&%v&%v&%v", jiraURL, user, dateRange, maxResult, excludeConfluence)

	//fmt.Println(url)
	fmt.Printf("Username: %v active from %v to %v\n", *inUserName, timeFrom.Format("02-01-2006"), timeTo.Format("02-01-2006"))

	//validate data
	if err := validateData(*inUserName); err != nil {
		log.Fatalln(err)
	}

	//make request data
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		log.Fatalln(err)
	}
	req.Header.Set("Authorization", token)
	resp, err := http.DefaultClient.Do(req)
	//We Read the response body on the line below.
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatalln(err)
	}
	//Convert the body to type string
	var feeds Feed
	// we unmarshal our byteArray which contains our
	// xmlFiles content into 'users' which we defined above
	err = xml.Unmarshal(body, &feeds)
	if err != nil {
		log.Fatalln(err)
	}

	var groupedEntries = make(map[string][]Entry)
	keys := linkedhashset.New[string]()

	for _, entry := range feeds.Entries {
		date := entry.Updated[:10]
		var ticket string
		if ticket = entry.Target.Title; ticket == "" {
			ticket = entry.Object.Title
		}
		key := fmt.Sprintf("%s_%s", date, ticket)
		groupedEntries[key] = append(groupedEntries[key], entry)
		keys.Add(key)
	}
	var date string
	for _, key := range keys.Values() {
		currentDate := key[:10]
		ticket := key[11:]
		if currentDate != date {
			fmt.Println(currentDate)
			date = currentDate
		}
		printActionDetail(ticket, groupedEntries[fmt.Sprintf("%v", key)])
	}
}

func validateData(userName string) error {
	if len(userName) == 0 {
		return errors.New("username can't be empty")
	}
	if len(topixFieldName) == 0 {
		return errors.New("topix field name can't be empty")
	}

	if len(token) == 0 {
		return errors.New("jira token can't be empty")
	}

	if len(excludeConfluence) == 0 {
		return errors.New("exclude confluence can't be empty")
	}

	if len(jiraURL) == 0 {
		return errors.New("jira URL can't be empty")
	}
	if len(jobFieldName) == 0 {
		return errors.New("job field name can't be empty")
	}
	return nil
}

func printActionDetail(ticket string, entries []Entry) {
	topixNumber, jobNumber := getJobNumber(ticket)
	printOutput(ticket, topixNumber, jobNumber, entries)
}

func printOutput(ticket string, topixNumber string, jobNumber string, entries []Entry) {
	fmt.Println("================================================================")
	fmt.Printf("%v\t%v/browse/%v\n", ticket, jiraURL, ticket)
	fmt.Printf("Topix number: %v\t Job number: %v\n", topixNumber, jobNumber)
	for _, entry := range entries {
		fmt.Println("----------------------------------------------------------------")
		fmt.Println(html2text.HTML2Text(entry.Title))
		fmt.Println("----------------------------------------------------------------")
		fmt.Println(html2text.HTML2Text(entry.Content))
	}
}

func getJobNumber(ticket string) (string, string) {
	if _, err := isValidTicket(ticket); err != nil {
		return fmt.Sprintf("Invalid ticket number %v", ticket), ""
	}

	url := fmt.Sprintf("%v/rest/api/latest/issue/%v", jiraURL, ticket)
	token := os.Getenv("JIRA_TOKEN")
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		log.Fatalln(err)
	}
	req.Header.Set("Authorization", token)
	resp, err := http.DefaultClient.Do(req)
	//We Read the response body on the line below.
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatalln(err)
	}
	topixNumber := gjson.Get(string(body), fmt.Sprintf("fields.%v", topixFieldName))
	jobNumber := gjson.Get(string(body), fmt.Sprintf("fields.%v", jobFieldName))
	parentTicket := gjson.Get(string(body), "fields.parent.key")

	if strings.TrimSpace(topixNumber.String()) == "" {
		if strings.TrimSpace(parentTicket.String()) == "" {
			return "Missing", jobNumber.String()
		}
		return getJobNumber(parentTicket.String())
	}

	return topixNumber.String(), jobNumber.String()
}

func isValidTicket(ticket string) (string, error) {
	ticketPattern := "\\w+-\\d+"
	if matched, err := regexp.Match(ticketPattern, []byte(ticket)); err != nil && matched {
		return ticket, err
	}
	return ticket, nil
}
