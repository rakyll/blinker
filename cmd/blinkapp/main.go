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

// Package main contains a Go mobile app that tweaks the
// cmd/blinker server's blink rate.
package main

import (
	"encoding/binary"
	"fmt"
	"log"
	"net/http"
	"sync"

	"golang.org/x/mobile/app"
	"golang.org/x/mobile/event/config"
	"golang.org/x/mobile/event/lifecycle"
	"golang.org/x/mobile/event/paint"
	"golang.org/x/mobile/event/touch"
	"golang.org/x/mobile/exp/f32"
	"golang.org/x/mobile/exp/gl/glutil"
	"golang.org/x/mobile/geom"
	"golang.org/x/mobile/gl"
)

var (
	program  gl.Program
	position gl.Attrib
	offset   gl.Uniform
	color    gl.Uniform
	buf      gl.Buffer

	c        config.Event
	touchLoc geom.Point
)

func main() {
	app.Main(func(a app.App) {
		for e := range a.Events() {
			switch e := app.Filter(e).(type) {
			case lifecycle.Event:
				switch e.Crosses(lifecycle.StageVisible) {
				case lifecycle.CrossOn:
					onStart()
				case lifecycle.CrossOff:
					onStop()
				}
			case config.Event:
				c = e
				touchLoc = geom.Point{c.Width / 2, c.Height / 2}
			case paint.Event:
				onPaint(c)
				a.EndPaint()
			case touch.Event:
				touchLoc = e.Loc
				go updateBlinker()
			}
		}
	})
}

func onStart() {
	var err error
	program, err = glutil.CreateProgram(vertexShader, fragmentShader)
	if err != nil {
		log.Printf("error creating GL program: %v", err)
		return
	}

	buf = gl.CreateBuffer()
	gl.BindBuffer(gl.ARRAY_BUFFER, buf)
	gl.BufferData(gl.ARRAY_BUFFER, rectData, gl.STATIC_DRAW)

	position = gl.GetAttribLocation(program, "position")
	color = gl.GetUniformLocation(program, "color")
	offset = gl.GetUniformLocation(program, "offset")
}

func onStop() {
	gl.DeleteProgram(program)
	gl.DeleteBuffer(buf)
}

func onPaint(c config.Event) {
	gl.ClearColor(1, 1, 1, 1)
	gl.Clear(gl.COLOR_BUFFER_BIT)

	gl.UseProgram(program)

	gl.Uniform4f(color, 0.3, 0.3, 0.3, 1) // color
	// position
	x := float32(touchLoc.X / c.Width)
	y := float32(touchLoc.Y / c.Height)
	gl.Uniform2f(offset, x, y)

	gl.BindBuffer(gl.ARRAY_BUFFER, buf)
	gl.EnableVertexAttribArray(position)
	gl.VertexAttribPointer(position, 2, gl.FLOAT, false, 0, 0)
	gl.DrawArrays(gl.TRIANGLE_STRIP, 0, 4)
	gl.DisableVertexAttribArray(position)
}

var mu sync.Mutex
var req *http.Request
var transport = &http.Transport{}

func updateBlinker() {
	r := float32(touchLoc.Y/c.Height) * 200
	// TODO(jbd): Don't hardcode the server.
	// TODO(jbd): Switch to mdns or another p2p protocol.
	mu.Lock()
	if req != nil {
		fmt.Printf("cancelling request: %v\n", req)
		transport.CancelRequest(req)
	}
	mu.Unlock()

	mu.Lock()
	req, _ = http.NewRequest("GET", fmt.Sprintf("http://10.0.1.9:8080?t=%d", int(r)), nil)
	mu.Unlock()

	c := &http.Client{Transport: transport}
	_, err := c.Do(req)
	if err != nil {
		fmt.Println(err)
	}

	mu.Lock()
	req = nil
	mu.Unlock()
}

var rectData = f32.Bytes(binary.LittleEndian,
	0, 0,
	0, 0.2,
	0.2, 0,
	0.2, 0.2,
)

const vertexShader = `#version 100
uniform vec2 offset;

attribute vec4 position;
void main() {
  // offset comes in with x/y values between 0 and 1.
  // position bounds are -1 to 1.
  vec4 offset4 = vec4(2.0*offset.x-1.0, 1.0-2.0*offset.y, 0, 0);
  gl_Position = position + offset4;
}`

const fragmentShader = `#version 100
precision mediump float;
uniform vec4 color;
void main() {
  gl_FragColor = color;
}`
