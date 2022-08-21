package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/ajtfj/graph"
)

const (
	GRAPH_FILE = "graph.txt"
)

var (
	g *graph.Graph
)

func HandleTCPConnection(conn net.Conn) {
	defer closeTCPConnection(conn)

	decoder := json.NewDecoder(conn)
	encoder := json.NewEncoder(conn)

	var requesPayload RequestPayload
	for {
		if err := decoder.Decode(&requesPayload); err != nil && err.Error() == "EOF" {
			encodeError(encoder, err)
		}

		startTime := time.Now()
		path, err := g.ShortestPath(requesPayload.Ori, requesPayload.Dest)
		if err != nil {
			encodeError(encoder, err)
		}
		duration := time.Since(startTime)

		responsePayload := ResponsePayload{
			Path:         path,
			CalcDuration: duration,
		}
		if err := encoder.Encode(responsePayload); err != nil {
			encodeError(encoder, err)
		}
	}
}

func main() {
	port, ok := os.LookupEnv("PORT")
	if !ok {
		log.Fatal("undefined PORT")
	}

	err := setupGraph()
	if err != nil {
		log.Fatal(err)
	}

	url := fmt.Sprintf("localhost:%s", port)
	addr, err := net.ResolveTCPAddr("tcp", url)
	if err != nil {
		log.Fatal(err)
	}

	ln, err := net.ListenTCP("tcp", addr)
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("waiting for connection on port %s\n", port)

	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Fatal(err)
		}

		go HandleTCPConnection(conn)
	}
}

func parceGraphInputLine(inputLine string) (graph.Node, graph.Node, int, error) {
	matches := strings.Split(inputLine, " ")
	if len(matches) < 3 {
		return graph.Node(""), graph.Node(""), 0, fmt.Errorf("invalid input")
	}

	weight, err := strconv.ParseInt(matches[2], 10, 0)
	if err != nil {
		return graph.Node(""), graph.Node(""), 0, err
	}

	return graph.Node(matches[0]), graph.Node(matches[1]), int(weight), nil
}

func setupGraph() error {
	g = graph.NewGraph()

	file, err := os.Open(GRAPH_FILE)
	if err != nil {
		return err
	}

	scanner := bufio.NewScanner(file)
	scanner.Split(bufio.ScanLines)

	for scanner.Scan() {
		inputLine := scanner.Text()
		u, v, weight, err := parceGraphInputLine(inputLine)
		if err != nil {
			return err
		}
		g.AddEdge(u, v, weight)
	}

	return nil
}

type RequestPayload struct {
	Ori  graph.Node `json:"ori"`
	Dest graph.Node `json:"dest"`
}

type ResponsePayload struct {
	Path         []graph.Node  `json:"path"`
	CalcDuration time.Duration `json:"calc-duration"`
}

type ResponseErrorPayload struct {
	Message string `json:"message"`
}

func closeTCPConnection(conn net.Conn) {
	err := conn.Close()
	if err != nil {
		log.Fatal(err)
	}
}

func encodeError(encoder *json.Encoder, err error) {
	payload := ResponseErrorPayload{
		Message: err.Error(),
	}
	if err := encoder.Encode(payload); err != nil {
		log.Fatal(err)
	}
}
