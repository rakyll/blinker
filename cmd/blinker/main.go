// Copyright 2015 Google Inc. All Rights Reserved.
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//     http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package main contains a server that blinks and slides LEDs in
// the provided blink rate.
package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	rpio "github.com/stianeikeland/go-rpio"
)

var pins = []rpio.Pin{
	rpio.Pin(4),
	rpio.Pin(17),
	rpio.Pin(22),
	rpio.Pin(5),
	rpio.Pin(6),
	rpio.Pin(19),
	rpio.Pin(21),
	rpio.Pin(16),
	rpio.Pin(12),
	rpio.Pin(25),
	rpio.Pin(23),
	rpio.Pin(18),
}

var tick = time.Second / 15

func main() {
	// Open and map memory to access gpio, check for errors
	if err := rpio.Open(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	go func() {
		// TODO(jbd): Use mdns after https://github.com/hashicorp/mdns/issues/44
		// is resolved for communication.
		http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			ms := r.URL.Query().Get("t")
			if ms != "" {
				t, _ := strconv.Atoi(ms)
				tick = time.Duration(t) * time.Millisecond
			}
			io.WriteString(w, "ok\n")
		})
		log.Fatal(http.ListenAndServe(":8080", nil))
	}()

	// Unmap gpio memory when done
	defer rpio.Close()

	// Set pin to output mode
	for _, p := range pins {
		p.Output()
	}

	for {
		for _, p := range pins {
			p.High()
			time.Sleep(tick)
			p.Low()
		}
	}
}
