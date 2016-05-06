# Loadtimes

a simple server side network tab viewer for webpages , built using [Appdash](https://github.com/sourcegraph/appdash).

## Usage

```
go get github.com/nandakola/loadtimes

go run main.go

```

Now, point your browser at localhost:8699. This loads the main page of the sample web app, which loads the HTML content coded on /home handler function inside main.go.
To demonstrate the JS and CSS load times some sample JS and CSS are added in to the rendered html.You can view the load time of those JS and CSS files by clicking on the link in the interface, which opens up the Appdash UI trace page.

javascript to collect the resource information is provided in "loadPerformanceData.js"

[[https://github.com/nandakola/loadtimes/blob/master/Sample.PNG]]
