# jira-track

## Description

This command line tool use for listing the activities of a user on a day.
It'll also print out the extra information of the custom field > so that we can use that information for logging work into another system.
This tool only useful for the lazy boy :)

## Environment variables
* J_JIRA_TOKEN :  jira token
* J_TOPIX_FIELD_NAME : jira custom field
* J_JOB_FIELD_NAME : jira custom field
* J_JIRA_URL : jira base url
* J_EXCLUDE_CONFLUENCE : filter out the confluence actions
* J_DEFAULT_USER: default username