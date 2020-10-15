# go-boids
Boid flock implemented in Go with WebAssembly. 

## Installation
1. Clone this project

2. Make sure you have Go installed. If not, [install Go](https://golang.org/doc/install)

3. Install goexec by running `go get -v -u github.com/shurcooL/goexec`

You're all set for running the program!

## Running
1. Spin up a web server by running ```goexec 'http.ListenAndServe(`:8080`, http.FileServer(http.Dir(`.`)))'```
2. Build the program by running `build.sh`
3. Navigate to localhost:8080 to see the boids! :bird:
